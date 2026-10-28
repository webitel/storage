package uploader

import (
	"context"
	"io"
	"time"

	"github.com/webitel/webitel-go-kit/pkg/logging"
	"github.com/webitel/webitel-go-kit/pkg/watcher"

	"github.com/webitel/storage/app"
	"github.com/webitel/storage/model"
	"github.com/webitel/storage/utils"
)

type UploadTask struct {
	app    *app.App
	job    *model.JobUploadFileWithProfile
	logger *logging.Logger
}

func (u *UploadTask) Name() string {
	return u.job.Uuid
}

// TODO added max count attempts ?
func (u *UploadTask) Execute(ctx context.Context) {
	store, err := u.app.GetFileBackendStore(u.job.ProfileId, u.job.ProfileUpdatedAt)
	if err != nil {
		u.storeError(ctx, err)
		return
	}

	u.logger.DebugCtx(ctx, "upload task started", logging.Str("store", store.Name()))

	r, err := u.app.FileCache.Reader(u.job, 0)
	if err != nil {
		u.storeError(ctx, err)
		return
	}
	defer r.Close()

	f := &model.File{
		DomainId:  u.job.DomainId,
		Uuid:      u.job.Uuid,
		ProfileId: u.job.ProfileId,
		CreatedAt: u.job.CreatedAt,
		BaseFile: model.BaseFile{
			Size:       u.job.Size,
			Name:       u.job.Name,
			MimeType:   u.job.MimeType,
			Properties: model.StringInterface{},
			Instance:   u.job.Instance,
			Channel:    u.job.Channel,
		},
	}
	var reader io.ReadCloser
	reader, err = u.app.FilePolicyForUpload(ctx, f.DomainId, &f.BaseFile, r)
	if err != nil {
		u.cancelUpload(ctx, err)
		return
	}
	defer reader.Close()

	if _, err = store.Write(reader, f); err != nil && err.GetId() != utils.ErrFileWriteExistsId {
		if model.IsFilePolicyError(err) {
			u.cancelUpload(ctx, err)
		} else {
			u.storeError(ctx, err)
		}
		return
	}

	if f.RetentionUntil == nil && store.ExpireDay() > 0 {
		t := time.Now().AddDate(0, 0, store.ExpireDay())
		f.RetentionUntil = &t
	}

	u.logger.DebugCtx(ctx, "upload task stored",
		logging.Str("file_name", u.job.GetStoreName()),
		logging.Str("store", store.Name()),
		logging.Int64("size", u.job.Size),
		logging.Bool("encrypted", f.IsEncrypted()),
	)

	result := <-u.app.Store.File().MoveFromJob(u.job.Id, u.job.ProfileId, f.Properties, f.RetentionUntil)
	if result.Err != nil {
		store.Remove(f)
		u.storeError(ctx, result.Err)
		return
	}

	u.removeCacheFile(ctx)
	u.logger.DebugCtx(ctx, "upload task finished")

	if *u.job.GetChannel() == model.UploadFileChannelCall {
		u.app.WatcherManager().Notify(model.PermissionScopeFiles, watcher.EventTypeRecordCall, app.NewFileWatcherData(f))
	}
}

func (u *UploadTask) cancelUpload(ctx context.Context, err model.AppError) {
	u.logger.ErrorCtx(ctx, "upload task canceled", logging.Err(err))
	err = u.app.Store.UploadJob().RemoveById(u.job.Id)
	if err != nil {
		u.logger.ErrorCtx(ctx, "remove upload job", logging.Err(err))
	}

	u.removeCacheFile(ctx)
}

func (u *UploadTask) removeCacheFile(ctx context.Context) {
	if err := u.app.FileCache.Remove(u.job); err != nil {
		u.logger.ErrorCtx(ctx, "remove cache file", logging.Err(err))
	}
}

func (u *UploadTask) storeError(ctx context.Context, err model.AppError) {
	u.logger.ErrorCtx(ctx, "upload task failed", logging.Err(err))
	u.app.Store.UploadJob().SetStateError(int(u.job.Id), err.Error())
}
