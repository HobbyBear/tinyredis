package netpoll

type ProtocolMsg interface {
	Bytes() []byte
}

type Protocol interface {
	ReadConn(c *RingBuffer) (ProtocolMsg, error)
	OnExecCmd(msg ProtocolMsg) ProtocolMsg
}
