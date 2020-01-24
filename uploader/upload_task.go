package uploader

import (
	"fmt"
	"github.com/webitel/storage/app"
	"github.com/webitel/storage/model"
	"github.com/webitel/wlog"
)

type UploadTask struct {
	app *app.App
	job *model.JobUploadFileWithProfile
}

func (u *UploadTask) Name() string {
	return u.job.Uuid
}

//TODO added max count attempts ?

func (u *UploadTask) Execute() {
	store, err := u.app.GetFileBackendStore(u.job.ProfileId, u.job.ProfileUpdatedAt)

	if err != nil {
		wlog.Critical(err.Error())
		u.app.Store.UploadJob().SetStateError(int(u.job.Id), err.Error())
		return
	}

	wlog.Debug(fmt.Sprintf("Execute upload task %s to store %s", u.Name(), store.Name()))

	r, err := u.app.FileCache.Reader(u.job, 0)
	if err != nil {
		wlog.Critical(err.Error())
		u.app.Store.UploadJob().SetStateError(int(u.job.Id), err.Error())
		return
	}
	defer r.Close()

	f := &model.File{
		Uuid:      u.job.Uuid,
		ProfileId: u.job.ProfileId,
		CreatedAt: u.job.CreatedAt,
		BaseFile: model.BaseFile{
			Size:       u.job.Size,
			Domain:     u.job.Domain,
			Name:       u.job.Name,
			MimeType:   u.job.MimeType,
			Properties: model.StringInterface{},
			Instance:   u.job.Instance,
		},
	}

	if _, err = store.Write(r, f); err != nil {
		wlog.Critical(err.Error())
		u.app.Store.UploadJob().SetStateError(int(u.job.Id), err.Error())
		return
	}

	wlog.Debug(fmt.Sprintf("Store %s to %s %d bytes", u.job.GetStoreName(), store.Name(), u.job.Size))

	result := <-u.app.Store.File().MoveFromJob(int(u.job.Id), u.job.ProfileId, f.Properties)
	if result.Err != nil {
		wlog.Critical(result.Err.Error())
		store.Remove(f)
		u.app.Store.UploadJob().SetStateError(int(u.job.Id), result.Err.Error())
		return
	}

	err = u.app.FileCache.Remove(u.job)
	if err != nil {
		wlog.Critical(err.Error())
	}

	wlog.Debug(fmt.Sprintf("End execute upload task %s", u.Name()))
}
