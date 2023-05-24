package netpoll

import (
	"errors"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"syscall"
	"time"
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
				fmt.Println("链接关闭", e.FD)
				continue
			}
			time.Sleep(time.Minute)
			fmt.Println("开始读取")
			// 从read里读取消息
			for {
				buf := make([]byte, 10)
				n, err := conn.Read(buf)
				if err != nil {
					fmt.Println("读取到了错误 ", err, n)
					conn.Close()
					break
				}
				if n > 0 {
					fmt.Println("读取到字节", string(buf[:n]))
				}
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
