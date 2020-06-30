package limnet

import (
	"runtime"

	"github.com/tangtaoit/limnet/pkg/eventloop"
	"github.com/tangtaoit/limnet/pkg/limlog"
	"github.com/tangtaoit/limnet/pkg/limpoller"
	"go.uber.org/zap"
	"golang.org/x/sys/unix"
)

// LIMNet limnet erver
type LIMNet struct {
	connectLoops []*eventloop.EventLoop // 连接的eventloop
	opts         *Options
	limlog.Log
	nextLoopIndex int
}

// New 创建server
func New(optFuncs ...Option) *LIMNet {
	opts := NewOption()
	for _, opt := range optFuncs {
		if opt != nil {
			if err := opt(opts); err != nil {
				panic(err)
			}
		}
	}
	l := &LIMNet{
		opts: opts,
		Log:  limlog.NewLIMLog("LIMNet"),
	}
	// 初始化连接的eventLoop
	l.initConnectEventLoop()

	return l
}

func (l *LIMNet) initConnectEventLoop() {
	if l.opts.ConnEventLoopNum <= 0 {
		l.opts.ConnEventLoopNum = runtime.NumCPU()
	}
	wloops := make([]*eventloop.EventLoop, l.opts.ConnEventLoopNum)
	for i := 0; i < l.opts.ConnEventLoopNum; i++ {
		l, err := eventloop.New()
		if err != nil {
			panic(err)
		}
		wloops[i] = l
	}
	l.connectLoops = wloops
}

// ---------- 实现 EventHandler ----------

// Handle 处理事件通知
func (l *LIMNet) Handle(fd int, event limpoller.Event) {
	if event&limpoller.EventRead != 0 {
		connfd, sa, err := unix.Accept(fd) // 接受连接的fd
		if err != nil {
			if err != unix.EAGAIN {
				l.Error("accept:", zap.Error(err))
			}
			return
		}
		if err := unix.SetNonblock(connfd, true); err != nil { // 连接设置为不阻塞
			_ = unix.Close(connfd)
			l.Error("set nonblock:", zap.Error(err))
			return
		}
		// 处理新的连接
		l.handleNewConnection(connfd, sa)
	}
}

// Close 关闭
func (l *LIMNet) Close() error {
	return nil
}

// ---------- 处理新的连接 ----------

func (l *LIMNet) nextLoop() *eventloop.EventLoop {
	loop := l.connectLoops[l.nextLoopIndex]
	l.nextLoopIndex = (l.nextLoopIndex + 1) % len(l.connectLoops)
	return loop
}

func (l *LIMNet) handleNewConnection(connfd int, sa unix.Sockaddr) {
	loop := l.nextLoop() // 获取conn的eventloop
	conn := NewConn(connfd, loop, l)

	// 绑定连接fd对应的处理者
	if err := loop.BindHandler(connfd, conn); err != nil {
		l.Error("连接添加失败！", zap.Error(err))
	}
}
