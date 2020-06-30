package atomic

import "sync/atomic"

// Int32 提供原子操作
type Int32 struct {
	v int32
}

// Add 计数增加 i ，减操作：Add(-1)
func (a *Int32) Add(i int) {
	atomic.AddInt32(&a.v, int32(i))
}

// Swap 交换值，并返回原来的值
func (a *Int32) Swap(i int) int32 {
	return atomic.SwapInt32(&a.v, int32(i))
}

// Get 获取值
func (a *Int32) Get() int32 {
	return atomic.LoadInt32(&a.v)
}

// Int64 提供原子操作
type Int64 struct {
	v int64
}

// Add 计数增加 i ，减操作：Add(-1)
func (a *Int64) Add(i int) {
	atomic.AddInt64(&a.v, int64(i))
}

// Swap 交换值，并返回原来的值
func (a *Int64) Swap(i int) int64 {
	return atomic.SwapInt64(&a.v, int64(i))
}

// Get 获取值
func (a *Int64) Get() int64 {
	return atomic.LoadInt64(&a.v)
}

// Bool Bool
type Bool struct {
	b int32
}

// Set Set
func (a *Bool) Set(b bool) {
	var newV int32
	if b {
		newV = 1
	}
	atomic.SwapInt32(&a.b, newV)
}

// Get Get
func (a *Bool) Get() bool {
	return atomic.LoadInt32(&a.b) == 1
}
