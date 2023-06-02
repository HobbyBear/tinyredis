package netpoll

import (
	"fmt"
	"testing"
	"time"
)

func TestIoQueue(t *testing.T) {
	h := func(c *Conn) {
		fmt.Println("收到连接")
		time.Sleep(10 * time.Second)
	}
	queue := newIoQueue(1, h, 3)
	queue.Put(&Conn{})
	queue.Wait()
	fmt.Println("执行完毕")
}
