package app

import (
	"fmt"
	"io"

	engine "github.com/webitel/engine/model"
	"github.com/webitel/storage/model"
)

func (app *App) SaveMediaFile(src io.Reader, mediaFile *model.MediaFile) (*model.MediaFile, engine.AppError) {
	var size int64
	var err engine.AppError

	if err = mediaFile.IsValid(); err != nil {
		return nil, err
	}

	if err = app.AllowMimeType(mediaFile.MimeType); err != nil {
		return nil, err
	}

	size, err = app.MediaFileStore.Write(src, mediaFile)
	if err != nil {
		return nil, err
	}
	mediaFile.Size = size
	mediaFile.Instance = app.GetInstanceId()

	if app.Config().MediaFileStoreSettings.MaxSizeByte != nil && *app.Config().MediaFileStoreSettings.MaxSizeByte < int(size) {
		app.MediaFileStore.Remove(mediaFile) //fixme check error
		return nil, engine.NewBadRequestError("model.media_file.size.app_error", "")
	}

	if mediaFile, err = app.Store.MediaFile().Create(mediaFile); err != nil {
		if err.GetId() != "store.sql_media_file.save.saving.duplicate" {
			app.MediaFileStore.Remove(mediaFile)
		}
		return nil, err
	} else {
		return mediaFile, nil
	}
}

func (app *App) AllowMimeType(mimeType string) engine.AppError {
	allow := app.Config().MediaFileStoreSettings.AllowMime
	if len(allow) == 0 {
		return nil
	}
	if !model.StringInSlice(mimeType, app.Config().MediaFileStoreSettings.AllowMime) {
		return engine.NewBadRequestError("model.media_file.mime_type.app_error", fmt.Sprintf("Not allowed mime type %s", mimeType))
	}

	return nil
}

func (app *App) GetMediaFilePage(domainId int64, search *model.SearchMediaFile) ([]*model.MediaFile, bool, engine.AppError) {
	files, err := app.Store.MediaFile().GetAllPage(domainId, search)
	if err != nil {
		return nil, false, err
	}

	search.RemoveLastElemIfNeed(&files)
	return files, search.EndOfList(), nil
}

func (app *App) GetMediaFile(domainId int64, id int) (*model.MediaFile, engine.AppError) {
	return app.Store.MediaFile().Get(domainId, id)
}

func (app *App) DeleteMediaFile(domainId int64, id int) (*model.MediaFile, engine.AppError) {
	file, err := app.Store.MediaFile().Get(domainId, id)
	if err != nil {
		return nil, err
	}

	if err = app.MediaFileStore.Remove(file); err != nil {
		return nil, err
	}

	if err = app.Store.MediaFile().Delete(domainId, file.Id); err != nil {
		return nil, err
	}

	return file, nil
}

func (app *App) GetMediaFileByName(name, domain string) (*model.MediaFile, engine.AppError) {
	if result := <-app.Store.MediaFile().GetByName(name, domain); result.Err != nil {
		return nil, result.Err
	} else {
		return result.Data.(*model.MediaFile), nil
	}
}

func (app *App) RemoveMediaFileByName(name, domain string) (file *model.MediaFile, err engine.AppError) {

	file, err = app.GetMediaFileByName(name, domain)
	if err != nil {
		return
	}

	err = app.MediaFileStore.Remove(file)
	if err != nil {
		return
	}

	result := <-app.Store.MediaFile().DeleteById(file.Id)
	return nil, result.Err
}
