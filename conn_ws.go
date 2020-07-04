package limnet

import (
	"time"

	"github.com/gorilla/websocket"
	"github.com/tangtaoit/limnet/pkg/bytebuffer"
	"github.com/tangtaoit/limnet/pkg/limlog"
	"github.com/tangtaoit/limnet/pkg/limutil/sync/atomic"
	"github.com/tangtaoit/limnet/pkg/ringbuffer"
	"go.uber.org/zap"
)

// WSConn websocket连接
type WSConn struct {
	id            int64
	conn          *websocket.Conn
	lnet          *LIMNet
	inboundBuffer *ringbuffer.RingBuffer // 来自客户端的数据
	connected     atomic.Bool
	activeTime    atomic.Int64 // 连接最后一次活动时间，单位秒
	limlog.Log
	buffer     []byte                 // inbound的临时buffer
	byteBuffer *bytebuffer.ByteBuffer // 临时读到的buffer
	ctx        interface{}            // 用户自定义的内容
	status     int                    // 用户自定义的连接状态
	version    uint8                  // 连接使用协议的版本
}

// NewWSConn 创建websocket连接
func NewWSConn(id int64, conn *websocket.Conn, lnet *LIMNet) *WSConn {
	w := &WSConn{
		id:            id,
		Log:           limlog.NewLIMLog("WSConn"),
		conn:          conn,
		lnet:          lnet,
		inboundBuffer: ringbuffer.Get(),
	}
	w.connected.Set(true)
	if lnet.opts.ConnIdleTime > 0 {
		_ = w.activeTime.Swap(int(time.Now().Unix()))
		lnet.timingWheel.AfterFunc(lnet.opts.ConnIdleTime, w.closeTimeoutConn())
	}
	go w.msgLoop()
	return w
}

func (c *WSConn) closeTimeoutConn() func() {
	return func() {
		if !c.connected.Get() { // 如果已关闭，什么都不做
			return
		}
		now := time.Now()
		intervals := now.Sub(time.Unix(c.activeTime.Get(), 0))
		if intervals >= c.lnet.opts.ConnIdleTime {
			_ = c.Close()
		} else {
			c.lnet.timingWheel.AfterFunc(c.lnet.opts.ConnIdleTime-intervals, c.closeTimeoutConn())
		}
	}
}

func (c *WSConn) msgLoop() {
	for true {
		_, data, err := c.conn.ReadMessage()
		if err != nil {
			c.Debug("客户端断开", zap.Error(err))
			c.handleClose()
			return
		}
		if c.lnet.opts.ConnIdleTime > 0 {
			_ = c.activeTime.Swap(int(time.Now().Unix()))
		}
		c.buffer = data
		for packet, _ := c.read(); packet != nil; packet, _ = c.read() {
			out := c.lnet.eventHandler.OnPacket(c, packet)
			if len(out) > 0 {
				c.write(out)
			}
		}
	}
}

func (c *WSConn) write(buf []byte) error {
	if !c.connected.Get() {
		return nil
	}
	err := c.conn.WriteMessage(websocket.BinaryMessage, buf)
	if err != nil {
		return err
	}
	return nil
}

func (c *WSConn) handleClose() error {
	if c.connected.Get() {
		c.connected.Set(false)

		c.lnet.eventHandler.OnClose(c) // 连接关闭
		c.conn.Close()
		c.release() // 释放连接
	}
	return nil
}

func (c *WSConn) read() ([]byte, error) {
	return c.lnet.opts.unPacket(c)
}

func (c *WSConn) release() {
	c.buffer = nil
	c.ctx = nil
	ringbuffer.Put(c.inboundBuffer)
	c.inboundBuffer = nil
	bytebuffer.Put(c.byteBuffer)
	c.byteBuffer = nil
}

// ==================== 公开方法 ====================

// GetID 获取客户端唯一ID
func (c *WSConn) GetID() int64 {
	return c.id
}

// Read Read
func (c *WSConn) Read() []byte {
	if !c.connected.Get() {
		return nil
	}
	if c.inboundBuffer.IsEmpty() {
		return c.buffer
	}
	// 将当前临时的buffer与inboundBuffer的数据合并后返回（虚读 不改变ringbuffer的长度）
	c.byteBuffer = c.inboundBuffer.WithByteBuffer(c.buffer)
	return c.byteBuffer.Bytes()
}

// ResetBuffer 重置buffer
func (c *WSConn) ResetBuffer() {
	c.buffer = c.buffer[:0]
	c.inboundBuffer.Reset()
	bytebuffer.Put(c.byteBuffer)
	c.byteBuffer = nil
}

// ReadN 读取指定长度的数据
func (c *WSConn) ReadN(n int) (size int, buf []byte) {
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
func (c *WSConn) ShiftN(n int) (size int) {
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

// 写数据
func (c *WSConn) Write(buf []byte) error {
	if !c.connected.Get() {
		return ErrConnectionClosed
	}
	return c.Write(buf)
}

// Close 关闭连接
func (c *WSConn) Close() error {
	if !c.connected.Get() {
		return ErrConnectionClosed
	}
	return c.handleClose()
}

// Context 获取用户上下文内容
func (c *WSConn) Context() interface{} { return c.ctx }

// SetContext 设置用户上下文内容
func (c *WSConn) SetContext(ctx interface{}) { c.ctx = ctx }

// Status 自定义连接状态
func (c *WSConn) Status() int { return c.status }

// SetStatus 设置状态
func (c *WSConn) SetStatus(status int) { c.status = status }

// Version 协议版本
func (c *WSConn) Version() uint8 { return c.version }

// SetVersion 设置连接的协议版本
func (c *WSConn) SetVersion(version uint8) { c.version = version }
