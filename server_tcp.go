package limnet

import (
	"github.com/tangtaoit/limnet/pkg/limlog"
	"go.uber.org/zap"
	"golang.org/x/sys/unix"
)

// TCPServer TCP服务
type TCPServer struct {
	limlog.Log
	acceptFd int
	addr     string
	lnet     *LIMNet
	realAddr string // 真实连接地址
	Stopped  chan struct{}
}

// NewTCPServer 创建一个tcp服务
func NewTCPServer(lnet *LIMNet) *TCPServer {
	s := &TCPServer{
		Log:     limlog.NewLIMLog("TCPServer"),
		lnet:    lnet,
		addr:    lnet.opts.Addr,
		Stopped: make(chan struct{}),
	}

	// 初始化listen和添加到listenerLoop
	s.initAndAddToLoopListen()

	return s
}

func (s *TCPServer) initAndAddToLoopListen() {

	// 开启tcp监听
	// var err error
	// s.ln, err = net.Listen("tcp", s.addr)
	// if err != nil {
	// 	panic(err)
	// }
	var err error
	s.acceptFd, err = unix.Socket(unix.AF_INET, unix.SOCK_STREAM, 0)
	if err != nil {
		panic(err)
	}
	_, _, port := parseAddr(s.addr)
	sockaddrInet4 := &unix.SockaddrInet4{
		Port: port,
		Addr: [4]byte{0, 0, 0, 0},
	}
	err = unix.Bind(s.acceptFd, sockaddrInet4)
	if err != nil {
		panic(err)
	}
	err = unix.Listen(s.acceptFd, 100)
	if err != nil {
		panic(err)
	}
	// f, err := s.ln.(*net.TCPListener).File()
	// if err != nil {
	// 	panic(err)
	// }
	s.realAddr = s.addr
	if err = unix.SetNonblock(s.acceptFd, true); err != nil {
		panic(err)
	}
	// 将tcp监听器放入loop 收到Conn将会调用 s.lnet.Handle
	err = s.lnet.listenerLoop.BindHandler(s.acceptFd, s.lnet)
	if err != nil {
		panic(err)
	}
}

// GetRealAddr 获取真实连接地址
func (s *TCPServer) GetRealAddr() string {
	return s.addr
}

// Stop Stop
func (s *TCPServer) Stop() error {
	s.lnet.listenerLoop.Trigger(func() error {
		s.lnet.listenerLoop.DeleteFdInLoop(s.acceptFd)
		err := unix.Close(s.acceptFd)
		if err != nil {
			s.Error("Quit fail", zap.Error(err))
			return err
		}
		s.Info("Quit")
		close(s.Stopped)
		return nil
	})
	return nil
}
