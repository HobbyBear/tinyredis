package netpoll

import (
	"bufio"
	"fmt"
	"tinyredis/log"
)

type Handler interface {
	OnConnect(msg *HandlerMsg)
	OnClose(msg *HandlerMsg)
	OnReadable(msg *HandlerMsg) error
}

type HandlerMsg struct {
	Conn *Conn
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

	buf := bufio.NewReader(msg.Conn)

	by, err := buf.ReadBytes('\n')
	if err != nil {
		log.Error(err.Error(), string(by))
	}
	fmt.Println("读取到 ", string(by))
	by, err = buf.ReadBytes('\n')
	if err != nil {
		log.Error(err.Error(), string(by))
	}
	fmt.Println("读取到2 ", string(by))
	return nil
}
