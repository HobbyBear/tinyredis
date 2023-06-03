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

func (R *RESPHandler) OnExecCmd(args []string) ([]string, byte) {
	fmt.Println("获取到命令 ", args)
	return cmd.HandleCmd(args)
}

func readConn(c *Conn) bool {
	reader := c.reader
	cmds, err := resp.ReadResp(reader)
	if err != nil {
		log.Debug("读取到了错误 err=%s ", err.Error())
		if err != syscall.EAGAIN {
			log.Info("连接读取错误,关闭连接 err=%s", err.Error())
			c.Close()
		}
		return false
	}
	c.midRes.req = cmds
	return true
}

func writeQueue(c *Conn) bool {
	if len(c.midRes.result) == 0 {
		return false
	}
	switch c.midRes.resultType {
	case resp.Error, resp.SimpleString, resp.Integer:
		resp.WriteSimple(c, c.midRes.result[0], c.midRes.resultType)
	case resp.BulkString:
		resp.WriteBulkString(c, c.midRes.result[0])
	case resp.Array:
		resp.WriteArray(c, c.midRes.result...)
	default:
		panic("resp type is not valid ")
	}
	return true
}
