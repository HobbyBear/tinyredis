package lineprotocol

import (
	"fmt"
	"tinyredis/netpoll"
)

type Msg struct {
	line string
}

func (p *Msg) Bytes() []byte {
	return append([]byte(p.line), []byte{'\r', '\n'}...)
}

var Instance = &Protocol{}

type Protocol struct {
}

func (p *Protocol) ReadConn(c *netpoll.RingBuffer) (netpoll.ProtocolMsg, error) {
	data, err := c.PeekBytes(0, '\n')
	if err != nil {
		return nil, err
	}
	c.AddReadPosition(len(data))
	fmt.Println("收到了消息", string(data))
	return &Msg{line: string(data[:len(data)-2])}, nil
}

func (p *Protocol) OnExecCmd(msg netpoll.ProtocolMsg) netpoll.ProtocolMsg {
	lineMsg := msg.(*Msg)
	return &Msg{line: fmt.Sprintf("收到消息:%s", lineMsg.line)}
}
