package netpoll

import (
	"net"
	"tinyredis/log"
)

type Handler interface {
	OnConnect(msg *HandlerMsg)
	OnClose(msg *HandlerMsg)
	OnReadable(msg *HandlerMsg) error
}

type HandlerMsg struct {
	Conn net.Conn
	Fd   int
}

type SimpleHandler struct {
}

func (s SimpleHandler) OnConnect(msg *HandlerMsg) {
	log.Info("连接到达 fd=%d", msg.Fd)
}

func (s SimpleHandler) OnClose(msg *HandlerMsg) {
	log.Info("连接关闭 fd=%d", msg.Fd)
}

func (s SimpleHandler) OnReadable(msg *HandlerMsg) error {
	buf := make([]byte, 1024)
	n, err := msg.Conn.Read(buf)
	if err != nil {
		return err
	}
	log.Info("读取到了数据 data=%s", string(buf[:n]))
	return nil
}
