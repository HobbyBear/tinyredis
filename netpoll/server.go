package netpoll

import (
	"bufio"
	"errors"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"syscall"
	"time"
	"tinyredis/log"
	"tinyredis/resp"
)

type Server struct {
	Poll     *poll
	addr     string
	Handler  Handler
	ListenFd int
}

func NewServ(addr string) *Server {
	return &Server{addr: addr}
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
	s.Handler = Handler{}
	s.ListenFd = listenFD
	go s.accept()
	go s.handler()
	time.Sleep(time.Hour)
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
			fmt.Println(err)
			return
		}

	}
}

func (s *Server) handler() {
	for {
		events, err := s.Poll.WaitEvents()
		if err != nil {
			fmt.Println(err)
			continue
		}
		for _, e := range events {
			file := os.NewFile(uintptr(e.FD), "")
			netFD, err := net.FileConn(file)
			if err != nil {
				// 处理错误
			}
			conn := netFD.(net.Conn)
			if e.Type == EventClose {
				conn.Close()
				log.Debug("链接关闭 fd=%d", e.FD)
				continue
			}
			// 从read里读取消息
			reader := bufio.NewReader(conn)
			for {
				payloadCh := resp.ParseStream(reader)
				payload := <-payloadCh
				if payload.Err != nil {
					log.Error("读取到命令行时发生错误 err=%s", payload.Err)
					conn.Close()
					break
				}
				log.Debug("读取到命令\n data=%s", string(payload.Data.ToBytes()))
				r, ok := payload.Data.(*resp.MultiBulkReply)
				if !ok {
					log.Error("require multi bulk protocol")
				}
				cmd := string(r.Args[0])
				switch cmd {
				case "set":
					conn.Write(resp.MakeOkReply().ToBytes())
				default:
					log.Info("not support cmd=%s", cmd)
				}
				if reader.Buffered() == 0 {
					break
				}
				log.Debug("缓冲区还没有读取完，继续读取下一个命令 bufferlen=%d", reader.Buffered())
			}
			log.Debug("读取完毕")
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
