package netpoll

import (
	"fmt"
	"strings"
	"testing"
)

func TestIoQueue(t *testing.T) {
	rd := strings.NewReader("wudiaaaa")
	buf := NewRingBuffer(20, rd)
	data, err := buf.PeekBytes(2, 'a')
	fmt.Println(string(data), err, buf.UnReadSize())
}
