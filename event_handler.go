package limnet

// EventHandler 事件处理者接口
type EventHandler interface {
	// 建立连接
	OnConnect(c *Conn)
	// 收到包 [data]为完整的数据包的数据
	OnPacket(c *Conn, data []byte) (out []byte)
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
func (d *DefaultEventHandler) OnPacket(c *Conn, data []byte) []byte {
	return nil
}

// OnClose 连接关闭
func (d *DefaultEventHandler) OnClose(c *Conn) {

}

// UnPacket 解包
type UnPacket func(c *Conn) ([]byte, error)

// UnPacket 解包协议
// type UnPacket interface {
// 	// 解包
// 	UnPacket(c *Conn) ([]byte, error)
// }

// Packet 封包协议
type Packet interface {
	// 封包
	Packet(c *Conn) []byte
}

// DefaultPacket 默认封包协议
type DefaultPacket struct {
}

// Packet Packet
func (d *DefaultPacket) Packet(c *Conn, data []byte) []byte {
	return data
}
