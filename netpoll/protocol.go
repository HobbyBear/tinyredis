package netpoll

type ProtocolMsg interface {
	Bytes() []byte
}

type Protocol interface {
	ReadConn(c *BufIO) (ProtocolMsg, error)
	OnExecCmd(msg ProtocolMsg) ProtocolMsg
}
