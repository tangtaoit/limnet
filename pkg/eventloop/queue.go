// Copyright 2019 Andy Pan. All rights reserved.
// Copyright 2018 Joshua J Baker. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package eventloop

import (
	"sync"

	"github.com/tangtaoit/limnet/pkg/limlog"
	"github.com/tangtaoit/limnet/pkg/limutil"
	"go.uber.org/zap"
)

// Job is a asynchronous function.
type Job func() error

// AsyncJobQueue queues pending tasks.
type AsyncJobQueue struct {
	lock sync.Locker
	jobs []func() error
}

// NewAsyncJobQueue creates a note-queue.
func NewAsyncJobQueue() AsyncJobQueue {
	return AsyncJobQueue{lock: &limutil.SpinLock{}}
}

// Push pushes a item into queue.
func (q *AsyncJobQueue) Push(job Job) (jobsNum int) {
	q.lock.Lock()
	q.jobs = append(q.jobs, job)
	jobsNum = len(q.jobs)
	q.lock.Unlock()
	return
}

// ExecuteJobs 执行所有job
func (q *AsyncJobQueue) ExecuteJobs() {
	q.lock.Lock()
	jobs := q.jobs
	q.jobs = nil
	q.lock.Unlock()
	for i := range jobs {
		if err := jobs[i](); err != nil {
			limlog.Warn("执行job失败！", zap.Error(err))
		}
	}
	return
}
