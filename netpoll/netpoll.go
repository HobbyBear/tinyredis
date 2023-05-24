package netpoll

import (
	"fmt"
	"golang.org/x/sys/unix"
	"syscall"
)

const (
	EpollClose = uint32(syscall.EPOLLIN | syscall.EPOLLRDHUP | syscall.EPOLLHUP)
)

const (
	EventIn    = 1 // 数据流入
	EventClose = 2 // 断开连接
)

type poll struct {
	EpollFd int
}

func (p *poll) Open() error {
	efd, err := syscall.EpollCreate1(0)
	if err != nil {
		return fmt.Errorf("epoll create fail err=%s", err)
	}
	p.EpollFd = efd
	return nil
}

func (p *poll) AddListen(fd int) error {
	err := syscall.EpollCtl(p.EpollFd, syscall.EPOLL_CTL_ADD, fd, &syscall.EpollEvent{
		Events: syscall.EPOLLIN | syscall.EPOLLERR | unix.EPOLLET | syscall.EPOLLRDHUP,
		Fd:     int32(fd),
	})
	if err != nil {
		return err
	}
	return nil
}

type event struct {
	FD   int32
	Type int32
}

func (p *poll) WaitEvents() ([]event, error) {
	epollEvents := make([]syscall.EpollEvent, 100)
	num, err := syscall.EpollWait(p.EpollFd, epollEvents, -1)
	if err != nil {
		return nil, err
	}

	events := make([]event, 0, len(epollEvents))
	for i := 0; i < num; i++ {
		event := event{
			FD: epollEvents[i].Fd,
		}
		if epollEvents[i].Events == EpollClose {
			event.Type = EventClose
		} else {
			event.Type = EventIn
		}
		events = append(events, event)
	}

	return events, nil
}
