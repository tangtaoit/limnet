package limnet

import "time"

// Options 配置
type Options struct {
	Network          string        // 网络类型 例如 tcp,websocket 等等
	Addr             string        // 连接地址 例如 127.0.0.1:6666
	ConnEventLoopNum int           // 连接事件loop数量 ， 如果为小于或等于0则为 runtime.NumCPU() 的值
	TimingWheelTick  time.Duration // 时间轮轮训间隔 必须大于等于1ms
	TimingWheelSize  int64         // 时间轮大小
	ConnIdleTime     time.Duration // 连接闲置时间，如果大于此闲置时间将自动关闭连接
	proto            Protocol      // 协议
}

// Option 参数项
type Option func(*Options) error

// NewOption 创建一个默认配置
func NewOption() *Options {
	return &Options{
		Network:          "tcp",
		Addr:             "127.0.0.1:6666",
		ConnEventLoopNum: 0,
		TimingWheelTick:  time.Millisecond * 1,
		TimingWheelSize:  1000,
		ConnIdleTime:     60 * time.Second,
		proto:            &DefaultProtocol{},
	}
}

// WithNetwork 设置网络类型
func WithNetwork(network string) Option {
	return func(opts *Options) error {
		opts.Network = network
		return nil
	}
}

// WithAddr 设置连接地址
func WithAddr(addr string) Option {
	return func(opts *Options) error {
		opts.Addr = addr
		return nil
	}
}

// WithProto 设置协议
func WithProto(proto Protocol) Option {
	return func(opts *Options) error {
		opts.proto = proto
		return nil
	}
}
