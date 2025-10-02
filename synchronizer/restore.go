package synchronizer

import (
	"encoding/json"
	"fmt"
	"github.com/webitel/storage/app"
	"github.com/webitel/storage/model"
	"github.com/webitel/storage/utils"
	"github.com/webitel/wlog"
	"strings"
)

type RestoreConfig struct {
	UserId *int64 `json:"user_id"`
}

type restoreFileJob struct {
	file model.SyncJob
	app  *app.App
}

func (j *restoreFileJob) Execute() {
	var file *model.FileWithProfile
	var backend utils.FileBackend
	var err model.AppError
	app := j.app

	log := app.Log.With(wlog.Int64("file_id", j.file.Id),
		wlog.String("action", model.Restore),
	)

	defer func() {
		err := j.app.Store.SyncFile().Remove(j.file.Id)
		if err != nil {
			log.Error(fmt.Sprintf("[restore] file %d, error: %s", j.file.FileId, err.Error()))
		}
	}()

	if file, err = app.Store.File().GetFileWithProfile(j.file.DomainId, j.file.FileId); err != nil {
		log.Error(fmt.Sprintf("[restore] file %d, error: %s", j.file.FileId, err.Error()))
		return
	}

	if backend, err = app.GetFileBackendStore(file.ProfileId, file.ProfileUpdatedAt); err != nil {
		log.Error(fmt.Sprintf("[restore] file %d, error: %s", j.file.FileId, err.Error()))
		return
	}

	if file.Malware == nil {
		log.Error(fmt.Sprintf("[restore] file %d, malware is nil", j.file.FileId))
		return
	}

	prop := file.Properties.Copy()
	var conf RestoreConfig
	json.Unmarshal(j.file.Config, &conf)

	err = backend.CopyTo(file, func(s string) string {
		return strings.ReplaceAll(s, "quarantine/", "")
	})

	if err != nil {
		log.Error(fmt.Sprintf("[restore] copy file %d, error: %s", j.file.FileId, err.String()))
		return
	}

	err = app.Store.File().Restored(file.Id, file.Properties, conf.UserId)
	if err != nil {
		log.Error(fmt.Sprintf("[restore] set properties file %d, error: %s", j.file.FileId, err.String()))
		return
	}

	file.Properties = prop
	err = backend.Remove(file)
	if err != nil {
		log.Error(fmt.Sprintf("[restore] remove file %d, error: %s", j.file.FileId, err.String()))
		return
	}

	wlog.Debug(fmt.Sprintf("file %d restore \"%s\" from store \"%s\"", j.file.FileId, j.file.Name, backend.Name()))

}
