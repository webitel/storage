package synchronizer

import (
	"context"
	"sync"
	"time"

	"github.com/webitel/webitel-go-kit/infra/otel/correlation"
	"github.com/webitel/webitel-go-kit/pkg/logging"

	"github.com/webitel/storage/app"
	"github.com/webitel/storage/interfaces"
	"github.com/webitel/storage/model"
	"github.com/webitel/storage/pool"
)

const logComponent logging.Component = "storage.synchronizer"

type synchronizer struct {
	App             *app.App
	limit           int
	schedule        chan struct{}
	pollingInterval time.Duration
	stopSignal      chan struct{}
	pool            interfaces.PoolInterface
	mx              sync.RWMutex
	stopped         bool
	logger          *logging.Logger
}

func init() {
	app.RegisterSynchronizer(func(a *app.App) interfaces.SynchronizerFilesInterface {
		logger := a.Logger().WithComponent(logComponent)
		logger.DebugCtx(context.Background(), "initialize synchronizer")
		return &synchronizer{
			App:             a,
			limit:           100,
			schedule:        make(chan struct{}, 1),
			stopSignal:      make(chan struct{}),
			pollingInterval: time.Second * 1,
			pool:            pool.NewPool(5, 10), //FIXME added config
			logger:          logger,
		}
	})
}

func (s *synchronizer) Start(ctx context.Context) {
	s.logger.DebugCtx(ctx, "run synchronizer")
	go s.run(ctx)
}

func (s *synchronizer) run(ctx context.Context) {
	var i int
	for {
		select {
		case <-s.schedule:
		case <-time.After(s.pollingInterval):
		start:
			var err model.AppError
			var jobs []*model.SyncJob

			if err = s.App.RemoveFileJobErrors(); err != nil {
				s.logger.ErrorCtx(ctx, "remove file job errors", logging.Err(err))
			}

			if err = s.App.SetRemoveFileJobs(); err != nil {
				s.logger.ErrorCtx(ctx, "set remove file jobs", logging.Err(err))
			}

			jobs, err = s.App.FetchFileJobs(s.limit)
			if err != nil {
				s.logger.ErrorCtx(ctx, "fetch file jobs", logging.Err(err))
				continue
			}
			count := len(jobs)

			if count > 0 {
				s.logger.DebugCtx(ctx, "fetched file jobs", logging.Int("count", count))
				for i = 0; i < count; i++ {
					jobCtx := correlation.WithIdentity(ctx, jobs[i].DomainId, 0)
					if task := s.getTask(jobs[i]); task != nil {
						s.pool.Exec(jobCtx, task)
					} else {
						s.logger.ErrorCtx(jobCtx, "bad job action",
							logging.Int64("job_id", jobs[i].Id),
							logging.Int64("file_id", jobs[i].FileId),
							logging.Str("action", jobs[i].Action),
						)
						s.App.Store.SyncFile().Remove(jobs[i].Id)
					}
				}

				if count == s.limit && !s.isStopped() {
					goto start
				}
			}

		case <-s.stopSignal:
			s.logger.DebugCtx(ctx, "synchronizer received stop signal")
			return
		}
	}
}

func (s *synchronizer) isStopped() bool {
	s.mx.RLock()
	defer s.mx.RUnlock()
	return s.stopped
}

func (s *synchronizer) Stop() {
	s.mx.Lock()
	s.stopped = true
	s.mx.Unlock()

	s.stopSignal <- struct{}{}
	s.pool.Close()
	s.pool.Wait()
	s.logger.DebugCtx(context.Background(), "synchronizer stopped")
}

func (s *synchronizer) getTask(src *model.SyncJob) interfaces.TaskInterface {
	logger := s.logger.With(
		logging.Int64("job_id", src.Id),
		logging.Int64("file_id", src.FileId),
		logging.Str("action", src.Action),
	)

	switch src.Action {
	case model.SyncJobRemove:
		return &removeFileJob{
			app:    s.App,
			file:   *src,
			logger: logger,
		}

	case model.SyncJobSTT:
		return &SttJob{
			app:    s.App,
			file:   *src,
			logger: logger,
		}

	case model.Transcoding:
		return &transcoding{
			app:    s.App,
			file:   *src,
			logger: logger,
		}

	case model.Restore:
		return &restoreFileJob{
			app:    s.App,
			file:   *src,
			logger: logger,
		}

	default:
		return nil
	}
}
