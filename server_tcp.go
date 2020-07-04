package limnet

import (
	"github.com/tangtaoit/limnet/pkg/limlog"
	"golang.org/x/sys/unix"
)

// TCPServer TCP服务
type TCPServer struct {
	limlog.Log
	fd       int
	addr     string
	lnet     *LIMNet
	realAddr string // 真实连接地址
}

// NewTCPServer 创建一个tcp服务
func NewTCPServer(lnet *LIMNet) *TCPServer {
	s := &TCPServer{
		Log:  limlog.NewLIMLog("TCPServer"),
		lnet: lnet,
		addr: lnet.opts.Addr,
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
	acceptFd, err := unix.Socket(unix.AF_INET, unix.SOCK_STREAM, 0)
	if err != nil {
		panic(err)
	}
	_, _, port := parseAddr(s.addr)
	sockaddrInet4 := &unix.SockaddrInet4{
		Port: port,
		Addr: [4]byte{0, 0, 0, 0},
	}
	err = unix.Bind(acceptFd, sockaddrInet4)
	if err != nil {
		panic(err)
	}
	err = unix.Listen(acceptFd, 100)
	if err != nil {
		panic(err)
	}
	// f, err := s.ln.(*net.TCPListener).File()
	// if err != nil {
	// 	panic(err)
	// }
	s.realAddr = s.addr
	s.fd = acceptFd
	if err = unix.SetNonblock(s.fd, true); err != nil {
		panic(err)
	}
	// 将tcp监听器放入loop 收到Conn将会调用 s.lnet.Handle
	err = s.lnet.listenerLoop.BindHandler(s.fd, s.lnet)
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
	unix.Close(s.fd)
	return nil
}
