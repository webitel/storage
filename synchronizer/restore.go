package synchronizer

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/webitel/webitel-go-kit/pkg/logging"

	"github.com/webitel/storage/app"
	"github.com/webitel/storage/model"
	"github.com/webitel/storage/utils"
)

type RestoreConfig struct {
	UserId *int64 `json:"user_id"`
}

type restoreFileJob struct {
	file   model.SyncJob
	app    *app.App
	logger *logging.Logger
}

func (j *restoreFileJob) Execute(ctx context.Context) {
	var file *model.FileWithProfile
	var backend utils.FileBackend
	var err model.AppError
	app := j.app

	defer func() {
		err := j.app.Store.SyncFile().Remove(j.file.Id)
		if err != nil {
			j.logger.ErrorCtx(ctx, "remove sync job", logging.Err(err))
		}
	}()

	if file, err = app.Store.File().GetFileWithProfile(j.file.DomainId, j.file.FileId); err != nil {
		j.logger.ErrorCtx(ctx, "get file with profile", logging.Err(err))
		return
	}

	if backend, err = app.GetFileBackendStore(file.ProfileId, file.ProfileUpdatedAt); err != nil {
		j.logger.ErrorCtx(ctx, "get file backend store", logging.Err(err))
		return
	}

	if file.Malware == nil {
		j.logger.ErrorCtx(ctx, "restore file without malware scan")
		return
	}

	prop := file.Properties.Copy()
	var conf RestoreConfig
	json.Unmarshal(j.file.Config, &conf)

	err = backend.CopyTo(file, func(s string) string {
		return strings.ReplaceAll(s, "quarantine/", "")
	})

	if err != nil {
		j.logger.ErrorCtx(ctx, "copy file", logging.Err(err))
		return
	}

	err = app.Store.File().Restored(file.Id, file.Properties, conf.UserId)
	if err != nil {
		j.logger.ErrorCtx(ctx, "set restored file properties", logging.Err(err))
		return
	}

	file.Properties = prop
	err = backend.Remove(file)
	if err != nil {
		j.logger.ErrorCtx(ctx, "remove quarantined file", logging.Err(err))
		return
	}

	j.logger.DebugCtx(ctx, "file restored", logging.Str("file_name", j.file.Name), logging.Str("store", backend.Name()))

}
