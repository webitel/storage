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
	enc := app.Config().EncryptedRecordKey
	size, err := app.FileCache.Write(src, file, enc)
	if err != nil {
		return err
	}

	file.Size = size
	file.Instance = app.GetInstanceId()
	file.Encrypted = enc != nil

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
	}

	size, err := app.DefaultFileStore.Write(src, f, nil)
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

	wlog.Debug(fmt.Sprintf("store %s to %s %d bytes", file.GetStoreName(), app.DefaultFileStore.Name(), file.Size))
	return nil
}

func (app *App) RemoveUploadJob(id int) engine.AppError {
	return nil
}
