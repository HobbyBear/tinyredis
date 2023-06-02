package netpoll

import (
	"math"
	"sync"
	"sync/atomic"
)

type ioHandle func(conn *Conn) bool

type ioQueue struct {
	seq        atomic.Int32
	handle     ioHandle
	conns      []chan *Conn
	wg         sync.WaitGroup
	dealedConn []*Conn
}

func newIoQueue(perChannelLen int, handle ioHandle, goNum int) *ioQueue {
	res := &ioQueue{handle: handle, wg: sync.WaitGroup{}, dealedConn: make([]*Conn, 0)}
	conns := make([]chan *Conn, goNum)
	for i := range conns {
		conns[i] = make(chan *Conn, perChannelLen)
		connlist := conns[i]
		go func() {
			for conn := range connlist {
				if handle(conn) {
					res.dealedConn = append(res.dealedConn, conn)
				}
				res.wg.Done()
			}
		}()
	}
	res.conns = conns
	return res
}

func (i *ioQueue) Put(conn *Conn) {
	num := i.seq.Load()
	if num == math.MaxInt32 {
		num = 0
	}
	i.conns[int(num)%len(i.conns)] <- conn
	num++
	i.wg.Add(1)
	i.seq.Store(num)
}

func (i *ioQueue) Wait() {
	i.wg.Wait()
}
