package limnet

import (
	"net"

	"github.com/tangtaoit/limnet/pkg/limlog"
	"golang.org/x/sys/unix"
)

// TCPServer TCP服务
type TCPServer struct {
	ln net.Listener
	limlog.Log

	lnet *LIMNet
}

// NewTCPServer 创建一个tcp服务
func NewTCPServer(lnet *LIMNet) *TCPServer {
	s := &TCPServer{
		Log:  limlog.NewLIMLog("TCPServer"),
		lnet: lnet,
	}

	// 初始化listen和添加到listenerLoop
	s.initAndAddToLoopListen()

	return s
}

func (s *TCPServer) initAndAddToLoopListen() {
	// 开启tcp监听
	var err error
	s.ln, err = net.Listen("tcp", s.lnet.opts.Addr)
	if err != nil {
		panic(err)
	}
	f, err := s.ln.(*net.TCPListener).File()
	if err != nil {
		panic(err)
	}
	fd := int(f.Fd())
	if err = unix.SetNonblock(fd, true); err != nil {
		panic(err)
	}
	// 将tcp监听器放入loop 收到Conn将会调用 s.lnet.Handle
	err = s.lnet.listenerLoop.BindHandler(fd, s.lnet)
	if err != nil {
		panic(err)
	}
}
