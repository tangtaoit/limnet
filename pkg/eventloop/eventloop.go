package eventloop

import (
	"sync"

	"github.com/Allenxuxu/toolkit/sync/spinlock"
	"github.com/tangtaoit/limnet/pkg/limlog"
	"github.com/tangtaoit/limnet/pkg/limpoller"
	"github.com/tangtaoit/limnet/pkg/limutil/sync/atomic"
	"go.uber.org/zap"
)

// Event 事件
type Event uint32

// ReadEvent 可读
const ReadEvent Event = 0x1

// WriteEvent 可写
const WriteEvent Event = 0x2

// EventHandler 事件处理者接口
type EventHandler interface {
	Handle(fd int, eventID limpoller.Event)
	Close() error
}

// EventLoop 事件循环
type EventLoop struct {
	poller        *limpoller.Poller
	handlers      sync.Map          // 处理者集合
	PacketBuf     []byte            // 包缓存
	writeJobs     []func()          // 写jobs集合
	jobLock       spinlock.SpinLock // job自旋锁
	eventHandling atomic.Bool       // 事件是否处理中
	limlog.Log
}

// New 创建
func New() (*EventLoop, error) {
	p, err := limpoller.Create()
	if err != nil {
		return nil, err
	}

	return &EventLoop{
		poller:    p,
		Log:       limlog.NewLIMLog("EventLoop"),
		PacketBuf: make([]byte, 0xFFFF),
	}, nil
}

// Run 运行事件循环
func (l *EventLoop) Run() {
	l.poller.Poll(l.handleEvent)
}

// BindHandler 绑定连接对应的处理者
func (l *EventLoop) BindHandler(fd int, h EventHandler) error {
	var err error
	l.handlers.Store(fd, h)
	if err = l.poller.AddRead(fd); err != nil {
		l.handlers.Delete(fd)
		return err
	}

	return nil
}

// Register 注册事件
func (l *EventLoop) Register(fd int, event Event) error {
	if event&ReadEvent != 0 {
		return l.poller.EnableRead(fd)
	}
	if event == ReadEvent|WriteEvent {
		return l.poller.EnableReadWrite(fd)
	}
	if event&WriteEvent != 0 {
		return l.poller.EnableWrite(fd)
	}
	return nil
}

// WriteJob WriteJob
func (l *EventLoop) WriteJob(f func()) {
	l.jobLock.Lock()
	l.writeJobs = append(l.writeJobs, f)
	l.jobLock.Unlock()

	if !l.eventHandling.Get() {
		if err := l.poller.Wake(); err != nil {
			l.Error("Job Wake error, ", zap.Error(err))
		}
	}
}

func (l *EventLoop) handleEvent(fd int, events limpoller.Event) {
	println("fd:%d  event: %d", fd, events)

	l.eventHandling.Set(true)
	if fd != -1 {
		s, ok := l.handlers.Load(fd)
		if ok {
			s.(EventHandler).Handle(fd, events)
		}
	}
	l.eventHandling.Set(false)
	// 执行写任务
	l.doWriteJobs()
}

func (l *EventLoop) doWriteJobs() {
	l.jobLock.Lock()
	jobs := l.writeJobs
	l.writeJobs = nil
	l.jobLock.Unlock()

	length := len(jobs)
	for i := 0; i < length; i++ {
		jobs[i]()
	}
}
