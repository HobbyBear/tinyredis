package netpoll

import (
	"net"
	"syscall"
)

type Conn struct {
	s      *Server
	conn   *net.TCPConn
	reader *BufIO
	nfd    int
}

func (c *Conn) GetReader() *BufIO {
	return c.reader
}

func (c *Conn) Write(p []byte) (n int, err error) {
	return c.conn.Write(p)
}

func (c *Conn) Close() error {
	c.s.ConnMap.Delete(c.nfd)
	return c.conn.Close()
}

func (c *Conn) Read(p []byte) (n int, err error) {
	rawConn, err := c.conn.SyscallConn()
	if err != nil {
		return 0, err
	}
	rawConn.Read(func(fd uintptr) (done bool) {
		n, err = syscall.Read(int(fd), p)
		if err != nil {
			return true
		}
		return true
	})
	return
}
