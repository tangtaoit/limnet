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
	asyncJobQueue AsyncJobQueue
	handlers      sync.Map          // 处理者集合
	packet        []byte            // 包缓存
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
		poller:        p,
		asyncJobQueue: NewAsyncJobQueue(),
		Log:           limlog.NewLIMLog("EventLoop"),
		packet:        make([]byte, 0xFFFF),
	}, nil
}

// PacketBuf 内部使用，临时缓冲区
func (l *EventLoop) PacketBuf() []byte {
	return l.packet
}

// Poller 获取Poller对象
func (l *EventLoop) Poller() *limpoller.Poller {
	return l.poller
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

// DeleteFdInLoop 删除 fd
func (l *EventLoop) DeleteFdInLoop(fd int) {
	if err := l.poller.Del(fd); err != nil {
		limlog.Error("[DeleteFdInLoop]", zap.Error(err))
	}
	l.handlers.Delete(fd)
}

// Trigger 将job推入队列，然后唤醒eventloop去执行job 从而达到串行的目的，避免了race
func (l *EventLoop) Trigger(job Job) error {
	if l.asyncJobQueue.Push(job) == 1 && !l.eventHandling.Get() {
		return l.poller.Wake()
	}
	return nil
}

func (l *EventLoop) handleEvent(fd int, events limpoller.Event) {
	l.eventHandling.Set(true)
	if fd != -1 { // -1表示唤醒操作
		s, ok := l.handlers.Load(fd)
		if ok {
			s.(EventHandler).Handle(fd, events)
		}
	}
	l.eventHandling.Set(false)
	// 执行任务
	l.asyncJobQueue.ExecuteJobs()
}
