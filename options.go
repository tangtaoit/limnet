package limnet

// Options 配置
type Options struct {
	Network          string // 网络类型 例如 tcp,websocket 等等
	Addr             string // 连接地址 例如 127.0.0.1:6666
	ConnEventLoopNum int    // 连接事件loop数量 ， 如果为小于或等于0则为 runtime.NumCPU() 的值
}

// Option 参数项
type Option func(*Options) error

// NewOption 创建一个默认配置
func NewOption() *Options {
	return &Options{
		Network:          "tcp",
		Addr:             "127.0.0.1:6666",
		ConnEventLoopNum: 0,
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
