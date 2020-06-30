package limnet

// EventHandler 事件处理者接口
type EventHandler interface {
	// 建立连接
	OnConnect(c *Conn)
	// 收到包 [data]为完整的数据包的数据
	OnPacket(c *Conn, data []byte)
	// 连接关闭
	OnClose(c *Conn)
}

// DefaultEventHandler 默认event处理者实现
type DefaultEventHandler struct {
}

// OnConnect 建立连接
func (d *DefaultEventHandler) OnConnect(c *Conn) {

}

// OnPacket 收到包
func (d *DefaultEventHandler) OnPacket(c *Conn, data []byte) {

}

// OnClose 连接关闭
func (d *DefaultEventHandler) OnClose(c *Conn) {

}

// Protocol 协议
type Protocol interface {
	// 解包
	UnPacket(c *Conn) ([]byte, error)
	// 封包
	Packet(c *Conn, data []byte) []byte
}

// DefaultProtocol 默认协议
type DefaultProtocol struct {
}

// UnPacket UnPacket
func (d *DefaultProtocol) UnPacket(c *Conn) ([]byte, error) {
	buf := c.Read()
	if len(buf) == 0 {
		return nil, nil
	}
	c.ResetBuffer()
	return buf, nil
}

// Packet Packet
func (d *DefaultProtocol) Packet(c *Conn, data []byte) []byte {
	return data
}