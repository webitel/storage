package synchronizer

import (
	"fmt"
	"sync"
	"time"

	"github.com/webitel/storage/app"
	"github.com/webitel/storage/interfaces"
	"github.com/webitel/storage/model"
	"github.com/webitel/storage/pool"
	"github.com/webitel/wlog"
)

type synchronizer struct {
	App             *app.App
	limit           int
	schedule        chan struct{}
	pollingInterval time.Duration
	stopSignal      chan struct{}
	pool            interfaces.PoolInterface
	mx              sync.RWMutex
	stopped         bool
}

func init() {
	app.RegisterSynchronizer(func(a *app.App) interfaces.SynchronizerFilesInterface {
		wlog.Debug("Initialize synchronizer")
		return &synchronizer{
			App:             a,
			limit:           100,
			schedule:        make(chan struct{}, 1),
			stopSignal:      make(chan struct{}),
			pollingInterval: time.Second * 1,
			pool:            pool.NewPool(5, 10), //FIXME added config
		}
	})
}

func (s *synchronizer) Start() {
	wlog.Debug("Run synchronizer")
	go s.run()
}

func (s *synchronizer) run() {
	var i int
	for {
		select {
		case <-s.schedule:
		case <-time.After(s.pollingInterval):
		start:
			var err model.AppError
			var jobs []*model.SyncJob

			if err = s.App.RemoveFileJobErrors(); err != nil {
				wlog.Error(err.Error())
			}

			if err = s.App.SetRemoveFileJobs(); err != nil {
				wlog.Error(err.Error())
			}

			jobs, err = s.App.FetchFileJobs(s.limit)
			if err != nil {
				wlog.Error(err.Error())
				continue
			}
			count := len(jobs)

			if count > 0 {
				wlog.Debug(fmt.Sprintf("fetch %d file jobs", count))
				for i = 0; i < count; i++ {
					if task := s.getTask(jobs[i]); task != nil {
						s.pool.Exec(task)
					} else {
						wlog.Error(fmt.Sprintf("bad job action: %v", jobs[i]))
						s.App.Store.SyncFile().Remove(jobs[i].Id)
					}
				}

				if count == s.limit && !s.isStopped() {
					goto start
				}
			}

		case <-s.stopSignal:
			wlog.Debug("Synchronizer received stop signal.")
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
	wlog.Debug("Synchronizer stopped.")
}

func (s *synchronizer) getTask(src *model.SyncJob) interfaces.TaskInterface {
	switch src.Action {
	case model.SyncJobRemove:
		return &removeFileJob{
			app:  s.App,
			file: *src,
		}

	case model.SyncJobSTT:
		return &SttJob{
			app:  s.App,
			file: *src,
		}

	case model.Transcoding:
		return &transcoding{
			app:  s.App,
			file: *src,
		}

	default:
		return nil
	}
}
