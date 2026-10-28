package pool

import (
	"context"
	"github.com/webitel/storage/interfaces"
	"sync"
)

type job struct {
	ctx  context.Context
	task interfaces.TaskInterface
}

type Pool struct {
	mu    sync.Mutex
	size  int
	tasks chan job
	kill  chan struct{}
	wg    sync.WaitGroup
}

func NewPool(workers int, queueCount int) interfaces.PoolInterface {
	pool := &Pool{
		tasks: make(chan job, queueCount),
		kill:  make(chan struct{}),
	}
	pool.Resize(workers)
	return pool
}

func (p *Pool) worker() {
	defer p.wg.Done()
	for {
		select {
		case j, ok := <-p.tasks:
			if !ok {
				return
			}
			j.task.Execute(j.ctx)
		case <-p.kill:
			return
		}
	}
}

func (p *Pool) Resize(n int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	for p.size < n {
		p.size++
		p.wg.Add(1)
		go p.worker()
	}
	for p.size > n {
		p.size--
		p.kill <- struct{}{}
	}
}

func (p *Pool) Close() {
	close(p.tasks)
}

func (p *Pool) Wait() {
	p.wg.Wait()
}

func (p *Pool) Exec(ctx context.Context, task interfaces.TaskInterface) {
	p.tasks <- job{ctx: ctx, task: task}
}
