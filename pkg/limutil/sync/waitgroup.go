package sync

import (
	"sync"
)

// WaitGroupWrapper WaitGroupWrapper
type WaitGroupWrapper struct {
	wg sync.WaitGroup
}

// AddAndRun AddAndRun
func (w *WaitGroupWrapper) AddAndRun(cb func()) {
	w.wg.Add(1)
	go func() {
		cb()
		w.wg.Done()
	}()
}

// Wait Wait
func (w *WaitGroupWrapper) Wait() {
	w.wg.Wait()
}
