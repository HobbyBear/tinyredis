package netpoll

import (
	"net"
	"sync"
	"syscall"
	"tinyredis/log"
	"tinyredis/resp"
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
		c := &Conn{
			conn: acceptConn.(*net.TCPConn),
		}
		c.reader = resp.NewBufIO(100, c)
		s.ConnMap.Store(nfd, c)
		s.Handler.OnConnect(&HandlerMsg{Conn: c, Fd: nfd})
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
			if IsReadableEvent(e.Type) {
				s.Handler.OnReadable(&HandlerMsg{Conn: conn, Fd: int(e.FD)})
			}
			if IsClosedEvent(e.Type) {
				conn.Close()
				s.Handler.OnClose(&HandlerMsg{Conn: conn, Fd: int(e.FD)})
			}
		}
	}
}
