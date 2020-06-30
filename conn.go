package limnet

import (
	"errors"
	"fmt"
	"time"

	"github.com/tangtaoit/limnet/pkg/bytebuffer"
	"github.com/tangtaoit/limnet/pkg/eventloop"
	"github.com/tangtaoit/limnet/pkg/limlog"
	"github.com/tangtaoit/limnet/pkg/limpoller"
	"github.com/tangtaoit/limnet/pkg/limutil/sync/atomic"
	"github.com/tangtaoit/limnet/pkg/ringbuffer"
	"go.uber.org/zap"
	"golang.org/x/sys/unix"
)

// ErrConnectionClosed 连接已关闭
var ErrConnectionClosed = errors.New("连接已关闭")

// Conn Conn
type Conn struct {
	fd        int // 连接fd
	loop      *eventloop.EventLoop
	connected atomic.Bool
	lnet      *LIMNet
	ctx       interface{} // 用户自定义的内容
	buffer    []byte      // inbound的临时buffer
	limlog.Log
	inboundBuffer  *ringbuffer.RingBuffer // 来自客户端的数据
	outboundBuffer *ringbuffer.RingBuffer // 将要写到客户端的数据
	byteBuffer     *bytebuffer.ByteBuffer // 临时读到的buffer
	activeTime     atomic.Int64           // 连接最后一次活动时间，单位秒
}

// NewConn 创建连接
func NewConn(connfd int, loop *eventloop.EventLoop, lnet *LIMNet) *Conn {
	conn := &Conn{
		fd:             connfd,
		loop:           loop,
		lnet:           lnet,
		Log:            limlog.NewLIMLog(fmt.Sprintf("Conn[connfd:%d]", connfd)),
		inboundBuffer:  ringbuffer.Get(),
		outboundBuffer: ringbuffer.Get(),
	}
	conn.connected.Set(true)
	if lnet.opts.ConnIdleTime > 0 {
		_ = conn.activeTime.Swap(int(time.Now().Unix()))
		lnet.timingWheel.AfterFunc(lnet.opts.ConnIdleTime, conn.closeTimeoutConn())
	}
	return conn
}

func (c *Conn) closeTimeoutConn() func() {
	return func() {
		now := time.Now()
		intervals := now.Sub(time.Unix(c.activeTime.Get(), 0))
		if intervals >= c.lnet.opts.ConnIdleTime {
			_ = c.Close()
		} else {
			c.lnet.timingWheel.AfterFunc(c.lnet.opts.ConnIdleTime-intervals, c.closeTimeoutConn())
		}
	}
}

// ---------- 实现 EventHandler ----------

// Handle 处理事件通知
func (c *Conn) Handle(connfd int, events limpoller.Event) {
	if c.lnet.opts.ConnIdleTime > 0 {
		_ = c.activeTime.Swap(int(time.Now().Unix()))
	}

	if events&limpoller.EventErr != 0 {
		c.handleClose(connfd)
		return
	}
	switch c.outboundBuffer.IsEmpty() {
	case false:
		if events&limpoller.EventWrite != 0 {
			c.handleWrite()
		}
		return
	case true:
		if events&limpoller.EventRead != 0 {
			c.handleRead()
		}
	}

}

func (c *Conn) handleRead() error {
	buf := c.loop.PacketBuf()
	n, err := unix.Read(c.fd, buf)
	if n == 0 || err != nil {
		if err == unix.EAGAIN {
			return nil
		}
		return c.handleClose(c.fd)
	}
	c.buffer = buf[:n]

	// c.read()会触发c.lnet.proto.UnPacket UnPacket会触发当前的 Read
	for packet, _ := c.read(); packet != nil; packet, _ = c.read() {
		c.lnet.eventHandler.OnPacket(c, packet)
	}
	_, err = c.inboundBuffer.Write(c.buffer)
	return err
}

func (c *Conn) read() ([]byte, error) {
	return c.lnet.opts.proto.UnPacket(c)
}

func (c *Conn) handleWrite() error {
	head, tail := c.outboundBuffer.LazyReadAll()
	n, err := unix.Write(c.fd, head)
	if err != nil {
		if err == unix.EAGAIN {
			return nil
		}
		return c.handleClose(c.fd)
	}
	c.outboundBuffer.Shift(n)
	if len(head) == n && tail != nil {
		n, err = unix.Write(c.fd, tail)
		if err != nil {
			if err == unix.EAGAIN {
				return nil
			}
			return c.handleClose(c.fd)
		}
		c.outboundBuffer.Shift(n)
	}
	if c.outboundBuffer.IsEmpty() {
		err := c.loop.Poller().EnableRead(c.fd)
		if err != nil {
			limlog.Error("[EnableRead]", zap.Error(err))
		}
	}
	return nil
}

func (c *Conn) handleClose(fd int) error {
	if c.connected.Get() {
		c.connected.Set(false)

		c.loop.DeleteFdInLoop(fd) // 删除eventloop里的此连接

		c.lnet.eventHandler.OnClose(c) // 连接关闭

		if err := unix.Close(c.fd); err != nil {
			limlog.Error("[close fd]", zap.Error(err))
		}
		c.release() // 释放连接
	}
	return nil
}

func (c *Conn) write(buf []byte) {
	if !c.outboundBuffer.IsEmpty() { // 如果输出buffer不为空，则写入到输出buffer里等下次event的时候真正写出去
		_, _ = c.outboundBuffer.Write(buf)
		return
	}
	// 如果输出buffer为空，则数据可以立马写出去
	n, err := unix.Write(c.fd, buf)
	if err != nil {
		if err == unix.EAGAIN {
			_, _ = c.outboundBuffer.Write(buf)
			_ = c.loop.Poller().EnableReadWrite(c.fd)
			return
		}
		_ = c.handleClose(c.fd)
		return
	}
	if n < len(buf) {
		_, _ = c.outboundBuffer.Write(buf[n:])
		_ = c.loop.Poller().EnableReadWrite(c.fd)
	}
}

// 释放连接
func (c *Conn) release() {
	c.buffer = nil
	c.ctx = nil
	ringbuffer.Put(c.inboundBuffer)
	ringbuffer.Put(c.outboundBuffer)
	c.inboundBuffer = nil
	c.outboundBuffer = nil
	bytebuffer.Put(c.byteBuffer)
	c.byteBuffer = nil
}

// ---------- 公用方法 ----------

// Read 读取数据
func (c *Conn) Read() []byte {
	if c.inboundBuffer.IsEmpty() {
		return c.buffer
	}
	// 将当前临时的buffer与inboundBuffer的数据合并后返回（虚读 不改变ringbuffer的长度）
	c.byteBuffer = c.inboundBuffer.WithByteBuffer(c.buffer)
	return c.byteBuffer.Bytes()
}

// ResetBuffer 重置客户端写入的buffer
func (c *Conn) ResetBuffer() {
	c.buffer = c.buffer[:0]
	c.inboundBuffer.Reset()
	bytebuffer.Put(c.byteBuffer)
	c.byteBuffer = nil
}

// ReadN 读取指定长度的数据
func (c *Conn) ReadN(n int) (size int, buf []byte) {
	inBufferLen := c.inboundBuffer.Length()
	tempBufferLen := len(c.buffer)
	if totalLen := inBufferLen + tempBufferLen; totalLen < n || n <= 0 {
		n = totalLen
	}
	size = n
	if c.inboundBuffer.IsEmpty() {
		buf = c.buffer[:n]
		return
	}
	head, tail := c.inboundBuffer.LazyRead(n)
	c.byteBuffer = bytebuffer.Get()
	_, err := c.byteBuffer.Write(head)
	if err != nil {
		c.Warn("Write head fail", zap.Error(err))
	}
	_, err = c.byteBuffer.Write(tail)
	if err != nil {
		c.Warn("Write head fail", zap.Error(err))
	}
	if inBufferLen >= n {
		buf = c.byteBuffer.Bytes()
		return
	}
	restSize := n - inBufferLen
	_, _ = c.byteBuffer.Write(c.buffer[:restSize])
	buf = c.byteBuffer.Bytes()
	return
}

// ShiftN 移动指定长度的下标
func (c *Conn) ShiftN(n int) (size int) {
	inBufferLen := c.inboundBuffer.Length()
	tempBufferLen := len(c.buffer)
	if inBufferLen+tempBufferLen < n || n <= 0 {
		c.ResetBuffer()
		size = inBufferLen + tempBufferLen
		return
	}
	size = n
	if c.inboundBuffer.IsEmpty() {
		c.buffer = c.buffer[n:]
		return
	}

	bytebuffer.Put(c.byteBuffer)
	c.byteBuffer = nil

	if inBufferLen >= n {
		c.inboundBuffer.Shift(n)
		return
	}
	c.inboundBuffer.Reset()

	restSize := n - inBufferLen
	c.buffer = c.buffer[restSize:]
	return
}

// Write Write
func (c *Conn) Write(buf []byte) (err error) {
	if !c.connected.Get() {
		return ErrConnectionClosed
	}
	return c.loop.Trigger(func() error {
		c.write(buf)
		return nil
	})
}

// Connected 是否已连接
func (c *Conn) Connected() bool {
	return c.connected.Get()
}

// BufferLength buffer长度
func (c *Conn) BufferLength() int {
	return c.inboundBuffer.Length() + len(c.buffer)
}

// Close 关闭
func (c *Conn) Close() error {
	if !c.connected.Get() {
		return ErrConnectionClosed
	}
	return c.loop.Trigger(func() error {
		return c.handleClose(c.fd)
	})
}

// Context 获取用户上下文内容
func (c *Conn) Context() interface{} { return c.ctx }

// SetContext 设置用户上下文内容
func (c *Conn) SetContext(ctx interface{}) { c.ctx = ctx }
