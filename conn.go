package limnet

import (
	"fmt"

	"github.com/tangtaoit/limnet/pkg/eventloop"
	"github.com/tangtaoit/limnet/pkg/limlog"
	"github.com/tangtaoit/limnet/pkg/limpoller"
	"github.com/tangtaoit/limnet/pkg/ringbuffer"
)

// Conn Conn
type Conn struct {
	connfd int // 连接fd
	loop   *eventloop.EventLoop
	lnet   *LIMNet
	limlog.Log
	inboundBuffer  *ringbuffer.RingBuffer // buffer for data from client
	outboundBuffer *ringbuffer.RingBuffer // buffer for data that is ready to write to client
}

// NewConn 创建连接
func NewConn(connfd int, loop *eventloop.EventLoop, lnet *LIMNet) *Conn {
	return &Conn{
		connfd:         connfd,
		loop:           loop,
		lnet:           lnet,
		Log:            limlog.NewLIMLog(fmt.Sprintf("Conn[connfd:%d]", connfd)),
		inboundBuffer:  ringbuffer.Get(),
		outboundBuffer: ringbuffer.Get(),
	}
}

// ---------- 实现 EventHandler ----------

// Handle 处理事件通知
func (c *Conn) Handle(connfd int, events limpoller.Event) {
	if events&limpoller.EventErr != 0 {
		c.handleClose(connfd)
		return
	}
	switch c.outboundBuffer.IsEmpty() {
	case false:
		if events&limpoller.EventWrite != 0 {
			c.loopRead()
		}
		return
	case true:
		if events&limpoller.EventRead != 0 {
			c.loopWrite()
		}
	}

}

func (c *Conn) loopRead() {
	// n, err := unix.Read(c.connfd, c.loop.PacketBuf)

}

func (c *Conn) loopWrite() {

}

// Close 关闭
func (c *Conn) Close() error {
	return nil
}

func (c *Conn) String() string {
	return fmt.Sprintf("connfd:%d", c.connfd)
}

func (c *Conn) handleClose(fd int) {

}
