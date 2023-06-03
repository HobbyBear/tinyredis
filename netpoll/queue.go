package netpoll

import (
	"math"
	"sync"
	"sync/atomic"
	"syscall"
)

type ioReadHandle func(c *BufIO) (ProtocolMsg, error)

type ioReadQueue struct {
	seq      atomic.Int32
	handle   ioReadHandle
	conns    []chan *Conn
	wg       sync.WaitGroup
	requests []*Request
}

type Request struct {
	msg  ProtocolMsg
	conn *Conn
}

func (i *ioReadQueue) Clear() {
	i.requests = make([]*Request, 0)
}

func newIoReadQueue(perChannelLen int, handle ioReadHandle, goNum int) *ioReadQueue {
	res := &ioReadQueue{handle: handle, wg: sync.WaitGroup{}, requests: make([]*Request, 0)}
	conns := make([]chan *Conn, goNum)
	for i := range conns {
		conns[i] = make(chan *Conn, perChannelLen)
		connlist := conns[i]
		go func() {
			for conn := range connlist {
				for {
					// todo
					msg, err := handle(conn.reader)
					if msg == nil || err == syscall.EAGAIN {
						break
					}
					if err != nil {
						conn.Close()
						break
					}
					if err == nil {
						res.requests = append(res.requests, &Request{msg: msg, conn: conn})
					}
				}
				res.wg.Done()
			}
		}()
	}
	res.conns = conns
	return res
}

func (i *ioReadQueue) Put(c *Conn) {
	num := i.seq.Load()
	if num == math.MaxInt32 {
		num = 0
	}
	i.conns[int(num)%len(i.conns)] <- c
	num++
	i.wg.Add(1)
	i.seq.Store(num)
}

func (i *ioReadQueue) Wait() {
	i.wg.Wait()
}

type ioWriteHandle func(msg *Request) error

type ioWriteQueue struct {
	seq    atomic.Int32
	handle ioWriteHandle
	conns  []chan *Request
	wg     sync.WaitGroup
}

func newIoWriteQueue(perChannelLen int, handle ioWriteHandle, goNum int) *ioWriteQueue {
	res := &ioWriteQueue{handle: handle, wg: sync.WaitGroup{}}
	conns := make([]chan *Request, goNum)
	for i := range conns {
		conns[i] = make(chan *Request, perChannelLen)
		connlist := conns[i]
		go func() {
			for conn := range connlist {
				// todo
				handle(conn)
				res.wg.Done()
			}
		}()
	}
	res.conns = conns
	return res
}

func (i *ioWriteQueue) Put(r *Request) {
	num := i.seq.Load()
	if num == math.MaxInt32 {
		num = 0
	}
	i.conns[int(num)%len(i.conns)] <- r
	num++
	i.wg.Add(1)
	i.seq.Store(num)
}

func (i *ioWriteQueue) Wait() {
	i.wg.Wait()
}
