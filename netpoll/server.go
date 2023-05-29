package netpoll

import (
	"bufio"
	"errors"
	"io"
	"net"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"tinyredis/log"
)

type Server struct {
	Poll     *poll
	addr     string
	Handler  Handler
	listener net.Listener
	ConnMap  sync.Map
}

func NewServ(addr string, handler Handler) *Server {
	return &Server{addr: addr, ConnMap: sync.Map{}, Handler: handler}
}

func (s *Server) Run() error {
	listener, err := net.Listen("tcp", s.addr)
	if err != nil {
		return err
	}
	s.listener = listener
	epollFD, err := syscall.EpollCreate1(0)
	if err != nil {
		return err
	}
	s.Poll = &poll{EpollFd: epollFD}
	go s.accept()
	go s.handler()
	ch := make(chan int)
	<-ch
	return nil
}

func (s *Server) accept() {
	for {
		acceptConn, err := s.listener.Accept()
		if err != nil {
			return
		}
		var nfd int
		rawConn, err := acceptConn.(*net.TCPConn).SyscallConn()
		if err != nil {
			log.Error(err.Error())
			continue
		}
		rawConn.Control(func(fd uintptr) {
			nfd = int(fd)
		})
		// 设置为非阻塞状态
		err = syscall.SetNonblock(nfd, true)
		if err != nil {
			return
		}
		err = s.Poll.AddListen(nfd)
		if err != nil {
			log.Error(err.Error())
			continue
		}
		s.ConnMap.Store(nfd, &Conn{
			conn:   acceptConn,
			reader: bufio.NewReader(acceptConn),
		})
		s.Handler.OnConnect(&HandlerMsg{Conn: acceptConn, Fd: nfd})
	}
}

func (s *Server) handler() {
	for {
		events, err := s.Poll.WaitEvents()
		if err != nil {
			log.Error(err.Error())
			continue
		}
		for _, e := range events {
			connInf, ok := s.ConnMap.Load(int(e.FD))
			if !ok {
				continue
			}
			conn := connInf.(*Conn)
			log.Debug("fd有数据到达 fd=%d", e.FD)
			if IsReadableEvent(e.Type) {
				err = s.Handler.OnReadable(&HandlerMsg{Conn: conn.conn, Fd: int(e.FD)})
				if err != nil && err != io.EOF {
					log.Error("read bytes fail err=%s", err)
					continue
				}
				if err == io.EOF {
					conn.conn.Close()
					s.Handler.OnClose(&HandlerMsg{Conn: conn.conn, Fd: int(e.FD)})
					continue
				}
				if err != nil {
					log.Error("read bytes fatal err=%s", err)
					continue
				}
			}
			if IsClosedEvent(e.Type) {
				conn.conn.Close()
				s.Handler.OnClose(&HandlerMsg{Conn: conn.conn, Fd: int(e.FD)})
			}
		}
	}
}

func getIPPort(addr string) (ip [4]byte, port int, err error) {
	strs := strings.Split(addr, ":")
	if len(strs) != 2 {
		err = errors.New("addr error")
		return
	}

	if len(strs[0]) != 0 {
		ips := strings.Split(strs[0], ".")
		if len(ips) != 4 {
			err = errors.New("addr error")
			return
		}
		for i := range ips {
			data, err := strconv.Atoi(ips[i])
			if err != nil {
				return ip, 0, err
			}
			ip[i] = byte(data)
		}
	}
	port, err = strconv.Atoi(strs[1])
	return
}
