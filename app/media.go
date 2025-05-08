package app

import (
	"io"

	"github.com/webitel/storage/model"
)

func (app *App) SaveMediaFile(src io.ReadCloser, mediaFile *model.MediaFile) (*model.MediaFile, model.AppError) {
	var size int64
	var err model.AppError

	mediaFile.Channel = model.NewString(model.UploadFileChannelMedia)

	if err = mediaFile.IsValid(); err != nil {
		return nil, err
	}

	src, err = app.FilePolicyForUpload(mediaFile.DomainId, &mediaFile.BaseFile, src)
	if err != nil {
		return nil, err
	}

	size, err = app.MediaFileStore.Write(src, mediaFile)
	if err != nil {
		return nil, err
	}
	mediaFile.Size = size
	mediaFile.Instance = app.GetInstanceId()

	if mediaFile, err = app.Store.MediaFile().Create(mediaFile); err != nil {
		if err.GetId() != "store.sql_media_file.save.saving.duplicate" {
			app.MediaFileStore.Remove(mediaFile)
		}
		return nil, err
	} else {
		return mediaFile, nil
	}
}

func (app *App) GetMediaFilePage(domainId int64, search *model.SearchMediaFile) ([]*model.MediaFile, bool, model.AppError) {
	files, err := app.Store.MediaFile().GetAllPage(domainId, search)
	if err != nil {
		return nil, false, err
	}

	search.RemoveLastElemIfNeed(&files)
	return files, search.EndOfList(), nil
}

func (app *App) GetMediaFile(domainId int64, id int) (mf *model.MediaFile, err model.AppError) {
	mf, err = app.Store.MediaFile().Get(domainId, id)
	if mf != nil {
		mf.Channel = model.NewString(model.UploadFileChannelMedia)
	}
	return
}

func (app *App) DeleteMediaFile(domainId int64, id int) (*model.MediaFile, model.AppError) {
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

func (app *App) GetMediaFileByName(name, domain string) (*model.MediaFile, model.AppError) {
	if result := <-app.Store.MediaFile().GetByName(name, domain); result.Err != nil {
		return nil, result.Err
	} else {
		return result.Data.(*model.MediaFile), nil
	}
}

func (app *App) RemoveMediaFileByName(name, domain string) (file *model.MediaFile, err model.AppError) {

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
