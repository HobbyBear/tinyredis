package netpoll

import (
	"net"
	"runtime"
	"sync"
	"syscall"
	"tinyredis/log"
)

type Server struct {
	Poll       *poll
	addr       string
	protocol   Protocol
	listener   net.Listener
	ConnMap    sync.Map
	readQueue  *ioReadQueue
	writeQueue *ioWriteQueue
}

func ioWriterHandle(r *Request) error {
	_, err := r.conn.Write(r.msg.Bytes())
	return err
}

func NewServ(addr string, protocol Protocol) *Server {
	return &Server{addr: addr, ConnMap: sync.Map{}, protocol: protocol,
		readQueue:  newIoReadQueue(100, protocol.ReadConn, runtime.NumCPU()),
		writeQueue: newIoWriteQueue(100, ioWriterHandle, runtime.NumCPU()),
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
			conn: acceptConn.(*net.TCPConn),
			nfd:  nfd,
			s:    s,
		}
		c.reader = NewBufIO(100, c)
		s.ConnMap.Store(nfd, c)
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
				continue
			}
			if IsReadableEvent(e.Type) {
				s.readQueue.Put(conn)
			}
		}
		s.readQueue.Wait()
		requests := s.readQueue.requests
		replyMsgs := make([]*Request, 0)
		for _, request := range requests {
			replyMsgs = append(replyMsgs, &Request{msg: s.protocol.OnExecCmd(request.msg), conn: request.conn})
		}
		for _, replyMsg := range replyMsgs {
			s.writeQueue.Put(replyMsg)
		}
		s.writeQueue.Wait()
		s.readQueue.Clear()
	}
}
