package synchronizer

import (
	"context"

	"github.com/webitel/webitel-go-kit/pkg/logging"

	"github.com/webitel/storage/app"
	"github.com/webitel/storage/model"
)

type removeFileJob struct {
	file   model.SyncJob
	app    *app.App
	logger *logging.Logger
}

func (j *removeFileJob) Execute(ctx context.Context) {
	store, err := j.app.GetFileBackendStore(j.file.ProfileId, j.file.ProfileUpdatedAt)
	if err != nil {
		j.logger.ErrorCtx(ctx, "get file backend store", logging.Err(err))
		//todo set db error

		return
	}

	err = store.Remove(&model.File{
		BaseFile:  j.file.BaseFile,
		Id:        j.file.Id,
		DomainId:  j.file.DomainId,
		Uuid:      "",
		ProfileId: j.file.ProfileId,
		CreatedAt: 0,
	})

	if err != nil {
		j.logger.ErrorCtx(ctx, "remove file from store", logging.Err(err))
	}

	err = j.app.Store.SyncFile().Clean(j.file.Id)
	if err != nil {
		j.logger.ErrorCtx(ctx, "clean sync job", logging.Err(err))
	}

	j.logger.DebugCtx(ctx, "file removed", logging.Str("file_name", j.file.Name), logging.Str("store", store.Name()))
}
