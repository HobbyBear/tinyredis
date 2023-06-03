package netpoll

type Handler interface {
	OnConnect(conn *Conn)
	OnClose(conn *Conn)
	OnExecCmd(cmd []string) ([]string, byte)
}

//type SimpleHandler struct {
//}
//
//func (s SimpleHandler) OnConnect(conn *Conn) {
//	log.Info("连接到达 ")
//}
//
//func (s SimpleHandler) OnClose(conn *Conn) {
//	log.Info("连接关闭")
//}
//
//func (s SimpleHandler) OnExecCmd(conn *Conn) error {
//
//	buf := bufio.NewReader(conn)
//
//	by, err := buf.ReadBytes('\n')
//	if err != nil {
//		log.Error(err.Error(), string(by))
//	}
//	fmt.Println("读取到 ", string(by))
//	by, err = buf.ReadBytes('\n')
//	if err != nil {
//		log.Error(err.Error(), string(by))
//	}
//	fmt.Println("读取到2 ", string(by))
//	return nil
//}
