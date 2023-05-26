package netpoll

import (
	"bufio"
	"net"
)

type Conn struct {
	conn   net.Conn
	reader *bufio.Reader
}
