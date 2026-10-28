package interfaces

import "context"

type TaskInterface interface {
	Execute(ctx context.Context)
}

type PoolInterface interface {
	Resize(n int)
	Close()
	Wait()
	Exec(ctx context.Context, task TaskInterface)
}
