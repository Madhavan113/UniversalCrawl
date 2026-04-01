package jobs

import (
	"context"
	"sync"
)

// Job represents a unit of async work.
type Job struct {
	ID      string
	Execute func(ctx context.Context) error
}

// Queue is an in-process, channel-based job queue.
type Queue struct {
	jobs    chan *Job
	wg      sync.WaitGroup
	ctx     context.Context
	cancel  context.CancelFunc
	workers int
}

// NewQueue creates a job queue with the given buffer size.
func NewQueue(bufferSize int) *Queue {
	if bufferSize < 1 {
		bufferSize = 100
	}
	ctx, cancel := context.WithCancel(context.Background())
	return &Queue{
		jobs:   make(chan *Job, bufferSize),
		ctx:    ctx,
		cancel: cancel,
	}
}

// Submit adds a job to the queue. Blocks if the queue is full.
func (q *Queue) Submit(job *Job) {
	q.jobs <- job
}

// Start launches n worker goroutines to process jobs.
func (q *Queue) Start(n int) {
	q.workers = n
	for i := 0; i < n; i++ {
		q.wg.Add(1)
		go q.worker()
	}
}

// Stop signals all workers to finish and waits for them to drain.
func (q *Queue) Stop() {
	q.cancel()
	close(q.jobs)
	q.wg.Wait()
}

func (q *Queue) worker() {
	defer q.wg.Done()
	for job := range q.jobs {
		if q.ctx.Err() != nil {
			return
		}
		job.Execute(q.ctx)
	}
}
