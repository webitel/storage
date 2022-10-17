package app

import (
	"github.com/webitel/storage/model"
	"github.com/webitel/storage/utils"
)

func (app *App) ListFiles(domain string, page, perPage int) ([]*model.File, *model.AppError) {
	if result := <-app.Store.File().GetAllPageByDomain(domain, page*perPage, perPage); result.Err != nil {
		return nil, result.Err
	} else {
		return result.Data.([]*model.File), nil
	}
}

func (app *App) GetFileWithProfile(domainId, id int64) (*model.File, utils.FileBackend, *model.AppError) {
	var file *model.FileWithProfile
	var backend utils.FileBackend
	var err *model.AppError

	if file, err = app.Store.File().GetFileWithProfile(domainId, id); err != nil {
		return nil, nil, err
	}

	if backend, err = app.GetFileBackendStore(file.ProfileId, file.ProfileUpdatedAt); err != nil {
		return nil, nil, err
	}
	//is bug ?
	return &file.File, backend, nil
}

func (app *App) GetFileByUuidWithProfile(domainId int64, uuid string) (*model.File, utils.FileBackend, *model.AppError) {
	var file *model.FileWithProfile
	var backend utils.FileBackend
	var err *model.AppError

	if file, err = app.Store.File().GetFileByUuidWithProfile(domainId, uuid); err != nil {
		return nil, nil, err
	}

	if backend, err = app.GetFileBackendStore(file.ProfileId, file.ProfileUpdatedAt); err != nil {
		return nil, nil, err
	}
	//is bug ?
	return &file.File, backend, nil
}

func (app *App) RemoveFiles(domainId int64, ids []int64) *model.AppError {
	return app.Store.File().MarkRemove(domainId, ids)
}

func (app *App) MaxUploadFileSize() int64 {
	return app.Config().MediaFileStoreSettings.MaxUploadFileSize
}
