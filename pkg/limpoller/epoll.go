// +build linux

package limpoller

import (
	"unsafe"

	"github.com/tangtaoit/limnet/pkg/limlog"
	"github.com/tangtaoit/limnet/pkg/limutil/sync/atomic"
	"go.uber.org/zap"
	"golang.org/x/sys/unix"
)

// Make the endianness of bytes compatible with more linux OSs under different processor-architectures,
// according to http://man7.org/linux/man-pages/man2/eventfd.2.html.
var (
	u uint64 = 1
	b        = (*(*[8]byte)(unsafe.Pointer(&u)))[:]
)

const readEvent = unix.EPOLLIN | unix.EPOLLPRI
const writeEvent = unix.EPOLLOUT

// Poller Epoll封装
type Poller struct {
	fd       int
	wfd      int    // wake fd
	wfdBuf   []byte // wfd buffer to read packet
	running  atomic.Bool
	waitDone chan struct{}
}

// Create 创建Poller
func Create() (*Poller, error) {
	fd, err := unix.EpollCreate1(unix.EPOLL_CLOEXEC)
	if err != nil {
		return nil, err
	}

	var wfd int // wake fd
	if wfd, err = unix.Eventfd(0, unix.EFD_NONBLOCK|unix.EFD_CLOEXEC); err != nil {
		panic(err)
	}

	err = unix.EpollCtl(fd, unix.EPOLL_CTL_ADD, wfd, &unix.EpollEvent{
		Events: unix.EPOLLPRI | unix.EPOLLIN,
		Fd:     int32(wfd),
	})
	if err != nil {
		_ = unix.Close(fd)
		_ = unix.Close(wfd)
		return nil, err
	}

	return &Poller{
		fd:       fd,
		wfd:      wfd,
		wfdBuf:   make([]byte, 8),
		waitDone: make(chan struct{}),
	}, nil
}

// var wakeBytes = []byte{1, 0, 0, 0, 0, 0, 0, 0}

// Wake 唤醒 epoll
func (ep *Poller) Wake() error {
	_, err := unix.Write(ep.wfd, b)
	return err
}

var buf = make([]byte, 8)

func (ep *Poller) wakeHandlerRead() {
	n, err := unix.Read(ep.wfd, ep.wfdBuf)
	if err != nil || n != 8 {
		limlog.Error("wakeHandlerRead", zap.Error(err), zap.Int("n", n))
	}
}

// Close 关闭 epoll
func (ep *Poller) Close() (err error) {
	if !ep.running.Get() {
		return ErrClosed
	}

	ep.running.Set(false)
	if err = ep.Wake(); err != nil {
		return
	}

	<-ep.waitDone
	_ = unix.Close(ep.fd)
	_ = unix.Close(ep.wfd)
	return
}

func (ep *Poller) add(fd int, events uint32) error {
	return unix.EpollCtl(ep.fd, unix.EPOLL_CTL_ADD, fd, &unix.EpollEvent{
		Events: events,
		Fd:     int32(fd),
	})
}

// AddRead 注册fd到epoll，并注册可读事件
func (ep *Poller) AddRead(fd int) error {
	return ep.add(fd, readEvent)
}

// AddWrite 注册fd到epoll，并注册可写事件
func (ep *Poller) AddWrite(fd int) error {
	return ep.add(fd, writeEvent)
}

// Del 从epoll中删除fd
func (ep *Poller) Del(fd int) error {
	return unix.EpollCtl(ep.fd, unix.EPOLL_CTL_DEL, fd, nil)
}

func (ep *Poller) mod(fd int, events uint32) error {
	return unix.EpollCtl(ep.fd, unix.EPOLL_CTL_MOD, fd, &unix.EpollEvent{
		Events: events,
		Fd:     int32(fd),
	})
}

// EnableReadWrite 修改fd注册事件为可读可写事件
func (ep *Poller) EnableReadWrite(fd int) error {
	return ep.mod(fd, readEvent|writeEvent)
}

// EnableWrite 修改fd注册事件为可写事件
func (ep *Poller) EnableWrite(fd int) error {
	return ep.mod(fd, writeEvent)
}

// EnableRead 修改fd注册事件为可读事件
func (ep *Poller) EnableRead(fd int) error {
	return ep.mod(fd, readEvent)
}

// Poll 启动 epoll wait 循环
func (ep *Poller) Poll(handler func(fd int, event Event)) {
	defer func() {
		close(ep.waitDone)
	}()

	events := make([]unix.EpollEvent, waitEventsBegin)
	var wake bool
	ep.running.Set(true)
	for {
		n, err := unix.EpollWait(ep.fd, events, -1)

		if err != nil && err != unix.EINTR {
			limlog.Error("EpollWait: ", zap.Error(err))
			continue
		}

		for i := 0; i < n; i++ {
			fd := int(events[i].Fd)
			if fd != ep.wfd {
				var rEvents Event
				if ((events[i].Events & unix.POLLHUP) != 0) && ((events[i].Events & unix.POLLIN) == 0) {
					rEvents |= EventErr
				}
				if (events[i].Events&unix.EPOLLERR != 0) || (events[i].Events&unix.EPOLLOUT != 0) {
					rEvents |= EventWrite
				}
				if events[i].Events&(unix.EPOLLIN|unix.EPOLLPRI|unix.EPOLLRDHUP) != 0 {
					rEvents |= EventRead
				}

				handler(fd, rEvents)
			} else {
				ep.wakeHandlerRead()
				wake = true
			}
		}

		if wake {
			handler(-1, 0)
			wake = false
			if !ep.running.Get() {
				return
			}
		}

		if n == len(events) {
			events = make([]unix.EpollEvent, n*2)
		}
	}
}
