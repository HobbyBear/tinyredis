package netpoll

import (
	"net"
	"runtime"
	"sync"
	"syscall"
	"tinyredis/log"
	"tinyredis/resp"
)

type Server struct {
	Poll       *poll
	addr       string
	Handler    Handler
	listener   net.Listener
	ConnMap    sync.Map
	readQueue  *ioQueue
	writeQueue *ioQueue
}

func NewServ(addr string, handler Handler) *Server {
	return &Server{addr: addr, ConnMap: sync.Map{}, Handler: handler,
		readQueue:  newIoQueue(100, readConn, runtime.NumCPU()),
		writeQueue: newIoQueue(100, writeQueue, runtime.NumCPU()),
	}
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
			conn:   acceptConn.(*net.TCPConn),
			midRes: &middleDealRes{req: nil, result: nil},
			nfd:    nfd,
			s:      s,
		}
		c.reader = resp.NewBufIO(100, c)
		s.ConnMap.Store(nfd, c)
		s.Handler.OnConnect(c)
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
			if IsClosedEvent(e.Type) {
				conn.Close()
				s.Handler.OnClose(conn)
				continue
			}
			if IsReadableEvent(e.Type) {
				s.readQueue.Put(conn)
			}
		}
		s.readQueue.Wait()
		readDealedConns := s.readQueue.dealedConn
		for _, conn := range readDealedConns {
			conn.midRes.result, conn.midRes.resultType = s.Handler.OnExecCmd(conn.midRes.req)
		}
		for _, conn := range readDealedConns {
			s.writeQueue.Put(conn)
		}
		s.writeQueue.Wait()
		// 清理中间缓存结果
		writeDealedConns := s.writeQueue.dealedConn
		for _, conn := range writeDealedConns {
			conn.midRes = &middleDealRes{}
			s.readQueue.dealedConn = make([]*Conn, 0)
			s.writeQueue.dealedConn = make([]*Conn, 0)
		}
	}
}
