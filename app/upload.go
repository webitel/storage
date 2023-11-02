package app

import (
	"fmt"
	"io"

	engine "github.com/webitel/engine/model"
	"github.com/webitel/storage/utils"

	"github.com/webitel/storage/model"
	"github.com/webitel/wlog"
)

func (app *App) AddUploadJobFile(src io.Reader, file *model.JobUploadFile) engine.AppError {
	size, err := app.FileCache.Write(src, file)
	if err != nil {
		return err
	}

	file.Size = size
	file.Instance = app.GetInstanceId()

	file, err = app.Store.UploadJob().Create(file)
	if err != nil {
		wlog.Error(fmt.Sprintf("Failed to store file %s, %v", file.Uuid, err))
		if errRem := app.FileCache.Remove(file); errRem != nil {
			wlog.Error(fmt.Sprintf("Failed to remove cache file %v", err))
		}
	} else {
		wlog.Debug(fmt.Sprintf("create new file job %d upload file: %s [%d %s]", file.Id, file.Name, file.Size, file.MimeType))
	}

	return err
}

func (app *App) SyncUpload(src io.Reader, file *model.JobUploadFile) engine.AppError {
	if app.UseDefaultStore() {
		// error
	}

	return app.syncUpload(app.DefaultFileStore, src, file, nil)
}

func (app *App) SyncUploadToProfile(src io.Reader, profileId int, file *model.JobUploadFile) engine.AppError {
	store, err := app.GetFileBackendStoreById(file.DomainId, profileId)
	if err != nil {
		return err
	}

	return app.syncUpload(store, src, file, &profileId)
}

func (app *App) syncUpload(store utils.FileBackend, src io.Reader, file *model.JobUploadFile, profileId *int) engine.AppError {
	f := &model.File{
		DomainId:  file.DomainId,
		Uuid:      file.Uuid,
		CreatedAt: model.GetMillis(),
		BaseFile: model.BaseFile{
			Size:       file.Size,
			Name:       file.Name,
			ViewName:   file.ViewName,
			MimeType:   file.MimeType,
			Properties: model.StringInterface{},
			Instance:   app.GetInstanceId(),
		},
		ProfileId: profileId,
	}

	size, err := store.Write(src, f)
	if err != nil && err.GetId() != utils.ErrFileWriteExistsId {
		return err
	}
	// fixme
	file.Size = size
	f.Size = file.Size

	res := <-app.Store.File().Create(f)
	if res.Err != nil {
		return res.Err
	} else {
		file.Id = res.Data.(int64)
	}

	wlog.Debug(fmt.Sprintf("store %s to %s %d bytes", file.GetStoreName(), store.Name(), file.Size))
	return nil
}
