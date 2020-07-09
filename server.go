package limnet

import (
	"fmt"
	"net"
	"runtime"
	"strconv"
	"strings"
	"sync/atomic"

	"github.com/RussellLuo/timingwheel"
	"github.com/tangtaoit/limnet/pkg/eventloop"
	"github.com/tangtaoit/limnet/pkg/limlog"
	"github.com/tangtaoit/limnet/pkg/limpoller"
	"github.com/tangtaoit/limnet/pkg/limutil/sync"
	"go.uber.org/zap"
	"golang.org/x/sys/unix"
)

// LIMNet limnet erver
type LIMNet struct {
	listenerLoop *eventloop.EventLoop   // listener的 eventLoop
	connectLoops []*eventloop.EventLoop // 连接的eventloop
	opts         *Options
	limlog.Log
	nextLoopIndex int
	tcp           *TCPServer
	ws            *WSServer
	eventHandler  EventHandler
	timingWheel   *timingwheel.TimingWheel
	idGen         int64
}

// New 创建server
func New(eventHandler EventHandler, optFuncs ...Option) *LIMNet {
	opts := NewOption()
	for _, opt := range optFuncs {
		if opt != nil {
			if err := opt(opts); err != nil {
				panic(err)
			}
		}
	}
	l := &LIMNet{
		opts:         opts,
		eventHandler: eventHandler,
		timingWheel:  timingwheel.NewTimingWheel(opts.TimingWheelTick, opts.TimingWheelSize),
		Log:          limlog.NewLIMLog("LIMNet"),
	}
	var err error
	l.listenerLoop, err = eventloop.New()
	if err != nil {
		panic(err)
	}
	// 初始化连接的eventLoop
	l.initConnectEventLoop()

	l.tcp = NewTCPServer(l)
	l.ws = NewWSServer(l)

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
				l.Error("accept:", zap.Error(err), zap.Int("fd", fd))
				panic(err)
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

// Run 运行
func (l *LIMNet) Run() {
	sw := sync.WaitGroupWrapper{}
	l.timingWheel.Start()
	l.ws.Start()
	length := len(l.connectLoops)
	for i := 0; i < length; i++ {
		sw.AddAndRun(l.connectLoops[i].Run)
	}
	sw.AddAndRun(l.listenerLoop.Run)
	sw.Wait()
	l.Error("狸猫IM退出！")
}

// Stop 停止服务
func (l *LIMNet) Stop() error {
	l.timingWheel.Stop()
	l.listenerLoop.Stop()
	for k := range l.connectLoops {
		if err := l.connectLoops[k].Stop(); err != nil {
			l.Error("stop conn fail ", zap.Error(err))
		}
	}
	l.tcp.Stop()
	return nil
}

// Close 关闭
func (l *LIMNet) Close() error {
	return nil
}

// GetTCPServer 获取当前tcp服务
func (l *LIMNet) GetTCPServer() *TCPServer {
	return l.tcp
}

// ---------- 处理新的连接 ----------

func (l *LIMNet) nextLoop() *eventloop.EventLoop {
	loop := l.connectLoops[l.nextLoopIndex]
	l.nextLoopIndex = (l.nextLoopIndex + 1) % len(l.connectLoops)
	return loop
}

func (l *LIMNet) handleNewConnection(connfd int, sa unix.Sockaddr) {
	loop := l.nextLoop() // 获取conn的eventloop
	clientID := atomic.AddInt64(&l.idGen, 1)
	conn := NewTCPConn(clientID, connfd, sockAddrToString(sa), loop, l) // 创建一个新的连接

	l.eventHandler.OnConnect(conn) // 触发连接事件

	// 绑定连接fd对应的处理者
	if err := loop.BindHandler(connfd, conn); err != nil {
		l.Error("连接添加失败！", zap.Error(err))
	}
}

// ---------- other ----------
func sockAddrToString(sa unix.Sockaddr) string {
	switch sa := (sa).(type) {
	case *unix.SockaddrInet4:
		return net.JoinHostPort(net.IP(sa.Addr[:]).String(), strconv.Itoa(sa.Port))
	case *unix.SockaddrInet6:
		return net.JoinHostPort(net.IP(sa.Addr[:]).String(), strconv.Itoa(sa.Port))
	default:
		return fmt.Sprintf("(unknown - %T)", sa)
	}
}
func parseAddr(addr string) (network, address string, port int) {
	network = "tcp"
	address = strings.ToLower(addr)
	if strings.Contains(address, "://") {
		pair := strings.Split(address, "://")
		network = pair[0]
		address = pair[1]
		pair2 := strings.Split(address, ":")
		portStr := pair2[1]
		portInt64, _ := strconv.ParseInt(portStr, 10, 64)
		port = int(portInt64)
	}
	return
}
