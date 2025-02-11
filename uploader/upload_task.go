package uploader

import (
	"fmt"
	"io"

	"github.com/webitel/storage/utils"

	"github.com/webitel/storage/app"
	"github.com/webitel/storage/model"
	"github.com/webitel/wlog"
)

type UploadTask struct {
	app *app.App
	job *model.JobUploadFileWithProfile
	log *wlog.Logger
}

func (u *UploadTask) Name() string {
	return u.job.Uuid
}

//TODO added max count attempts ?

func (u *UploadTask) Execute() {
	store, err := u.app.GetFileBackendStore(u.job.ProfileId, u.job.ProfileUpdatedAt)

	if err != nil {
		u.log.Error(err.Error(),
			wlog.Err(err),
		)
		u.app.Store.UploadJob().SetStateError(int(u.job.Id), err.Error())
		return
	}

	u.log.Debug(fmt.Sprintf("start upload task %d [%s] to store %s", u.job.Id, u.Name(), store.Name()))

	r, err := u.app.FileCache.Reader(u.job, 0)
	if err != nil {
		u.log.Error(err.Error(),
			wlog.Err(err),
		)
		u.app.Store.UploadJob().SetStateError(int(u.job.Id), err.Error())
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
	reader, err = u.app.FilePolicyForUpload(f.DomainId, &f.BaseFile, r)
	if err != nil {
		u.log.Error(err.Error(),
			wlog.Err(err),
		)
		// TODO drop call file ???
		u.app.Store.UploadJob().SetStateError(int(u.job.Id), err.Error())
		return
	}
	defer reader.Close()

	if _, err = store.Write(reader, f); err != nil && err.GetId() != utils.ErrFileWriteExistsId {
		u.log.Error(err.Error(),
			wlog.Err(err),
		)
		u.app.Store.UploadJob().SetStateError(int(u.job.Id), err.Error())
		return
	}

	u.log.Debug(fmt.Sprintf("store %s to %s %d bytes", u.job.GetStoreName(), store.Name(), u.job.Size))

	result := <-u.app.Store.File().MoveFromJob(u.job.Id, u.job.ProfileId, f.Properties, f.RetentionUntil)
	if result.Err != nil {
		u.log.Error(result.Err.Error(),
			wlog.Err(result.Err),
		)
		store.Remove(f)
		u.app.Store.UploadJob().SetStateError(int(u.job.Id), result.Err.Error())
		return
	}

	err = u.app.FileCache.Remove(u.job)
	if err != nil {
		u.log.Error(err.Error(),
			wlog.Err(err),
		)
	}

	u.log.Debug(fmt.Sprintf("finish upload task %d [%s]", u.job.Id, u.Name()))
}
