package netpoll

import (
	"fmt"
	"golang.org/x/sys/unix"
	"syscall"
	"tinyredis/log"
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
		Events: syscall.EPOLLIN | unix.EPOLLET | syscall.EPOLLRDHUP,
		Fd:     int32(fd),
	})
	if err != nil {
		return err
	}
	return nil
}

type event struct {
	FD   int32
	Type uint32
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
		debugEvent(epollEvents[i].Events)
		event.Type = epollEvents[i].Events
		events = append(events, event)
	}

	return events, nil
}

func IsReadableEvent(event uint32) bool {
	if event&syscall.EPOLLIN != 0 {
		return true
	}
	return false
}

func IsClosedEvent(event uint32) bool {
	if event&syscall.EPOLLHUP != 0 {
		return true
	}
	if event&syscall.EPOLLRDHUP != 0 {
		return true
	}
	return false
}

func debugEvent(event uint32) {
	if event&syscall.EPOLLHUP != 0 {
		log.Debug("receive EPOLLHUP")
	}
	if event&syscall.EPOLLERR != 0 {
		log.Debug("receive EPOLLERR")
	}
	if event&syscall.EPOLLIN != 0 {
		log.Debug("receive EPOLLIN")
	}
	if event&syscall.EPOLLOUT != 0 {
		log.Debug("receive EPOLLOUT")
	}
	if event&syscall.EPOLLRDHUP != 0 {
		log.Debug("receive EPOLLRDHUP")
	}
}
