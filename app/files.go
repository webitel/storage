package app

import (
	"context"

	engine "github.com/webitel/engine/model"
	"github.com/webitel/storage/model"
	"github.com/webitel/storage/utils"
)

func (app *App) ListFiles(domain string, page, perPage int) ([]*model.File, engine.AppError) {
	if result := <-app.Store.File().GetAllPageByDomain(domain, page*perPage, perPage); result.Err != nil {
		return nil, result.Err
	} else {
		return result.Data.([]*model.File), nil
	}
}

func (app *App) CheckCallRecordPermissions(ctx context.Context, fileId int, currentUserId int64, domainId int64, groups []int) (bool, engine.AppError) {
	return app.Store.File().CheckCallRecordPermissions(ctx, fileId, currentUserId, domainId, groups)

}

func (app *App) GetFileWithProfile(domainId, id int64) (*model.File, utils.FileBackend, engine.AppError) {
	var file *model.FileWithProfile
	var backend utils.FileBackend
	var err engine.AppError

	if file, err = app.Store.File().GetFileWithProfile(domainId, id); err != nil {
		return nil, nil, err
	}

	if backend, err = app.GetFileBackendStore(file.ProfileId, file.ProfileUpdatedAt); err != nil {
		return nil, nil, err
	}
	//is bug ?
	return &file.File, backend, nil
}

func (app *App) GetFileByUuidWithProfile(domainId int64, uuid string) (*model.File, utils.FileBackend, engine.AppError) {
	var file *model.FileWithProfile
	var backend utils.FileBackend
	var err engine.AppError

	if file, err = app.Store.File().GetFileByUuidWithProfile(domainId, uuid); err != nil {
		return nil, nil, err
	}

	if backend, err = app.GetFileBackendStore(file.ProfileId, file.ProfileUpdatedAt); err != nil {
		return nil, nil, err
	}
	//is bug ?
	return &file.File, backend, nil
}

func (app *App) RemoveFiles(domainId int64, ids []int64) engine.AppError {
	return app.Store.File().MarkRemove(domainId, ids)
}

func (app *App) MaxUploadFileSize() int64 {
	return app.Config().MediaFileStoreSettings.MaxUploadFileSize
}

func (app *App) StoreFile(src model.File) (model.File, engine.AppError) {
	res := <-app.Store.File().Create(&src)
	if res.Err != nil {
		return model.File{}, res.Err
	}

	src.Id = res.Data.(int64)

	return src, nil
}
