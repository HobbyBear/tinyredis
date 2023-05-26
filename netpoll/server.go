package netpoll

import (
	"bufio"
	"errors"
	"io"
	"net"
	"os"
	"strconv"
	"strings"
	"syscall"
	"tinyredis/log"
)

type Server struct {
	Poll     *poll
	addr     string
	Handler  Handler
	ListenFd int
	ConnMap  map[int]*Conn
}

func NewServ(addr string, handler Handler) *Server {
	return &Server{addr: addr, ConnMap: map[int]*Conn{}, Handler: handler}
}

func (s *Server) Run() error {
	listenFD, err := syscall.Socket(syscall.AF_INET, syscall.SOCK_STREAM, 0)
	if err != nil {
		return err
	}
	err = syscall.SetsockoptInt(listenFD, syscall.SOL_SOCKET, syscall.SO_REUSEADDR, 1)
	if err != nil {
		return err
	}

	addr, port, err := getIPPort(s.addr)
	if err != nil {
		return err
	}
	err = syscall.Bind(listenFD, &syscall.SockaddrInet4{
		Port: port,
		Addr: addr,
	})
	if err != nil {
		return err
	}
	err = syscall.Listen(listenFD, 1024)
	if err != nil {
		return err
	}

	epollFD, err := syscall.EpollCreate1(0)
	if err != nil {
		return err
	}
	s.Poll = &poll{EpollFd: epollFD}
	s.ListenFd = listenFD
	go s.accept()
	go s.handler()
	ch := make(chan int)
	<-ch
	return nil
}

func (s *Server) accept() {
	for {
		nfd, _, err := syscall.Accept(s.ListenFd)
		if err != nil {
			return
		}

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
		file := os.NewFile(uintptr(nfd), "")
		netFD, err := net.FileConn(file)
		if err != nil {
			log.Error(err.Error())
			continue
		}
		err = file.Close()
		if err != nil {
			log.Error(err.Error())
		}
		s.ConnMap[nfd] = &Conn{
			conn:   netFD.(net.Conn),
			reader: bufio.NewReader(netFD.(net.Conn)),
		}
		s.Handler.OnConnect(&HandlerMsg{Conn: netFD.(net.Conn), Fd: nfd})
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
			conn := s.ConnMap[int(e.FD)]
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
