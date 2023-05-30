package netpoll

import (
	"net"
	"syscall"
	"tinyredis/resp"
)

type Conn struct {
	conn   *net.TCPConn
	reader *resp.BufIO
}

func (c *Conn) Close() error {
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
