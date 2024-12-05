package uploader

import (
	"fmt"
	"sync"
	"time"

	"github.com/webitel/storage/app"
	"github.com/webitel/storage/interfaces"
	"github.com/webitel/storage/model"
	"github.com/webitel/storage/pool"
	"github.com/webitel/storage/store"
	"github.com/webitel/wlog"
)

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
	log               *wlog.Logger
}

func init() {
	app.RegisterUploader(func(a *app.App) interfaces.UploadRecordingsFilesInterface {
		wlog.Debug("Initialize uploader")
		return &UploaderInterfaceImpl{
			App:               a,
			limit:             100,
			betweenAttemptSec: 60,
			schedule:          make(chan struct{}, 1),
			stopSignal:        make(chan struct{}),
			pollingInterval:   time.Second * 2,
			pool:              pool.NewPool(100, 10), //FIXME added config
			log: a.Log.With(
				wlog.Namespace("context"),
				wlog.String("scope", "uploader"),
			),
		}
	})
}

func (u *UploaderInterfaceImpl) Start() {
	u.log.Debug("Run uploader")
	go u.run()
}

func (u *UploaderInterfaceImpl) run() {
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
				u.log.Critical(result.Err.Error(),
					wlog.Err(result.Err),
				)
				continue
			}
			jobs = result.Data.([]*model.JobUploadFileWithProfile)

			count = len(jobs)
			if count > 0 {
				u.log.Debug(fmt.Sprintf("fetch %d jobs upload files", count))
				for i = 0; i < count; i++ {
					j := jobs[i]
					u.pool.Exec(&UploadTask{
						app: u.App,
						job: jobs[i],
						log: u.log.With(
							wlog.Int64("file_id", j.Id),
							wlog.String("call_id", j.Uuid), // TODO
						),
					})
				}

				if count == u.limit && !u.isStopped() {
					goto start
				}
			}
		case <-u.stopSignal:
			u.log.Debug("Uploader received stop signal.")
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
	u.log.Debug("Uploader stopped.")
}
