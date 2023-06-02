package netpoll

import (
	"fmt"
	"syscall"
	"tinyredis/cmd"
	"tinyredis/log"
	"tinyredis/resp"
)

type RESPHandler struct {
}

func (R *RESPHandler) OnConnect(conn *Conn) {

}

func (R *RESPHandler) OnClose(conn *Conn) {
	log.Info("连接关闭")
}

func (R *RESPHandler) DeCodeRespProtocal(conn *Conn) {

}

func (R *RESPHandler) OnExecCmd(args [][]byte) [][]byte {
	fmt.Println("获取到命令", string(args[0]))
	result := cmd.HandleCmd(args)
	return result
}

func readConn(c *Conn) bool {
	reader := c.reader
	payload := resp.ParseStream(reader)
	if payload.Err != nil {
		log.Debug("读取到了错误 err=%s ", payload.Err.Error())
		if payload.Err != syscall.EAGAIN {
			log.Info("连接读取错误,关闭连接 err=%s", payload.Err.Error())
			c.Close()
		}
		return false
	}
	mutilReply := payload.Data.(*resp.MultiBulkReply)
	c.midRes.req = mutilReply.Args
	return true
}

func writeQueue(c *Conn) bool {
	if len(c.midRes.result) == 0 {
		return false
	}
	_, err := c.Write([]byte("+OK"))
	if err != nil {
		log.Error(err.Error())
		return false
	}
	return true
}

//func (R *RESPHandler) OnExecCmd(conn *Conn) error {
//	reader := conn.reader
//	payload := resp.ParseStream(reader)
//	if payload.Err != nil {
//		log.Debug("读取到了错误 err=%s ", payload.Err.Error())
//		if payload.Err != syscall.EAGAIN {
//			log.Info("连接读取错误,关闭连接 err=%s", payload.Err.Error())
//			conn.Close()
//		}
//		return nil
//	}
//	mutilReply := payload.Data.(*resp.MultiBulkReply)
//	result := cmd.HandleCmd(mutilReply.Args)
//	conn.Write(result)
//	fmt.Println("获取到命令", string(payload.Data.ToBytes()), "回复", string(result))
//	fmt.Println("继续读取下一个命令")
//	return nil
//}
