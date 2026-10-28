package uploader

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
	"github.com/webitel/storage/store"
)

const logComponent logging.Component = "storage.uploader"

type UploaderInterfaceImpl struct {
	App               *app.App
	betweenAttemptSec int64
	limit             int
	schedule          chan struct{}
	pollingInterval   time.Duration
	stopSignal        chan struct{}
	pool              interfaces.PoolInterface
	mx                sync.RWMutex
	stopped           bool
	logger            *logging.Logger
}

func init() {
	app.RegisterUploader(func(a *app.App) interfaces.UploadRecordingsFilesInterface {
		logger := a.Logger().WithComponent(logComponent)
		logger.DebugCtx(context.Background(), "initialize uploader")
		return &UploaderInterfaceImpl{
			App:               a,
			limit:             100,
			betweenAttemptSec: 60,
			schedule:          make(chan struct{}, 1),
			stopSignal:        make(chan struct{}),
			pollingInterval:   time.Second * 2,
			pool:              pool.NewPool(100, 10), //FIXME added config
			logger:            logger,
		}
	})
}

func (u *UploaderInterfaceImpl) Start(ctx context.Context) {
	u.logger.DebugCtx(ctx, "run uploader")
	go u.run(ctx)
}

func (u *UploaderInterfaceImpl) run(ctx context.Context) {
	var result store.StoreResult
	var jobs []*model.JobUploadFileWithProfile
	var count int
	var i int
	for {
		select {
		case <-u.schedule:
		case <-time.After(u.pollingInterval):
		start:
			if result = <-u.App.Store.UploadJob().UpdateWithProfile(u.limit, u.App.GetInstanceId(), u.betweenAttemptSec, u.App.UseDefaultStore()); result.Err != nil {
				u.logger.ErrorCtx(ctx, "fetch upload jobs", logging.Err(result.Err))
				continue
			}
			jobs = result.Data.([]*model.JobUploadFileWithProfile)

			count = len(jobs)
			if count > 0 {
				u.logger.DebugCtx(ctx, "fetched upload jobs", logging.Int("count", count))
				for i = 0; i < count; i++ {
					j := jobs[i]
					u.pool.Exec(correlation.WithIdentity(ctx, j.DomainId, 0), &UploadTask{
						app: u.App,
						job: j,
						logger: u.logger.With(
							logging.Int64("file_id", j.Id),
							logging.Str("call_id", j.Uuid), // TODO
						),
					})
				}

				if count == u.limit && !u.isStopped() {
					goto start
				}
			}
		case <-u.stopSignal:
			u.logger.DebugCtx(ctx, "uploader received stop signal")
			return
		}
	}
}

func (u *UploaderInterfaceImpl) isStopped() bool {
	u.mx.RLock()
	defer u.mx.RUnlock()
	return u.stopped
}

func (u *UploaderInterfaceImpl) Stop() {
	u.mx.Lock()
	u.stopped = true
	u.mx.Unlock()

	u.stopSignal <- struct{}{}
	u.pool.Close()
	u.pool.Wait()
	u.logger.DebugCtx(context.Background(), "uploader stopped")
}
