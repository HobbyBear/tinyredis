package netpoll

import (
	"fmt"
	"syscall"
	"tinyredis/log"
	"tinyredis/resp"
)

type RESPHandler struct {
}

func (R *RESPHandler) OnConnect(msg *HandlerMsg) {

}

func (R *RESPHandler) OnClose(msg *HandlerMsg) {
	log.Info("连接关闭 fd=%d", msg.Fd)
}

func (R *RESPHandler) OnReadable(msg *HandlerMsg) error {
	reader := msg.Conn.reader
	for {
		payload := resp.ParseStream(reader)
		if payload.Err != nil {
			log.Debug("读取到了错误 err=%s ", payload.Err.Error())
			if payload.Err != syscall.EAGAIN {
				log.Info("连接读取错误,关闭连接 err=%s", payload.Err.Error())
				msg.Conn.Close()
			}
			break
		}
		fmt.Println("获取到命令", string(payload.Data.ToBytes()))
		fmt.Println("继续读取下一个命令")
	}
	return nil
}
