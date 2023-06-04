package respprotocol

import (
	"fmt"
	"syscall"
	"tinyredis/log"
	"tinyredis/netpoll"
)

var Instance = &Protocol{}

func init() {
	RegisterCommands()
}

type Msg struct {
	cmds       []string
	resultByte byte
}

func (r *Msg) Bytes() []byte {
	switch r.resultByte {
	case Error, SimpleString, Integer:
		return ToSimpleBytes(r.cmds[0], r.resultByte)
	case BulkString:
		return ToBulkStringBytes(r.cmds[0])
	case Array:
		ToArrayBytes(r.cmds...)
	default:
		panic("resp type is not valid ")
	}
	return nil
}

type Protocol struct {
}

func (r *Protocol) ReadConn(reader *netpoll.RingBuffer) (netpoll.ProtocolMsg, error) {
	cmds, err := ReadResp(reader)
	if err != nil {
		log.Debug("读取到了错误 err=%s ", err.Error())
		if err != syscall.EAGAIN {
			log.Info("连接读取错误,关闭连接 err=%s", err.Error())
			return nil, err
		}
		return nil, nil
	}
	return &Msg{cmds: cmds}, nil
}

func (r *Protocol) OnExecCmd(msg netpoll.ProtocolMsg) netpoll.ProtocolMsg {
	respMsg := msg.(*Msg)
	fmt.Println("获取到命令 ", respMsg.cmds)
	reply := &Msg{
		cmds:       respMsg.cmds,
		resultByte: 0,
	}
	reply.cmds, reply.resultByte = handleCmd(respMsg.cmds)
	return reply
}
