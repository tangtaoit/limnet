package limutil

import (
	"runtime"
	"sync/atomic"
)

// SpinLock 自旋锁
type SpinLock struct {
	lock uintptr
}

// Lock Lock
func (l *SpinLock) Lock() {
	for !atomic.CompareAndSwapUintptr(&l.lock, 0, 1) {
		runtime.Gosched()
	}
}

// Unlock Unlock
func (l *SpinLock) Unlock() {
	atomic.StoreUintptr(&l.lock, 0)
}
