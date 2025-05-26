package app

import (
	"context"
	"fmt"
	watcherkit "github.com/webitel/webitel-go-kit/pkg/watcher"
	"github.com/webitel/wlog"

	"github.com/webitel/storage/model"
	"github.com/webitel/storage/utils"
)

func (app *App) SearchFiles(ctx context.Context, domainId int64, search *model.SearchFile) ([]*model.File, bool, model.AppError) {
	res, err := app.Store.File().GetAllPage(ctx, domainId, search)
	if err != nil {
		return nil, false, err
	}
	search.RemoveLastElemIfNeed(&res)
	return res, search.EndOfList(), nil
}

func (app *App) CheckCallRecordPermissions(ctx context.Context, fileId int, currentUserId int64, domainId int64, groups []int) (bool, model.AppError) {
	return app.Store.File().CheckCallRecordPermissions(ctx, fileId, currentUserId, domainId, groups)

}

func (app *App) GetFileWithProfile(domainId, id int64) (*model.File, utils.FileBackend, model.AppError) {
	var file *model.FileWithProfile
	var backend utils.FileBackend
	var err model.AppError

	if file, err = app.Store.File().GetFileWithProfile(domainId, id); err != nil {
		return nil, nil, err
	}

	if backend, err = app.GetFileBackendStore(file.ProfileId, file.ProfileUpdatedAt); err != nil {
		return nil, nil, err
	}
	//is bug ?
	return &file.File, backend, nil
}

func (app *App) GetFileByUuidWithProfile(domainId int64, uuid string) (*model.File, utils.FileBackend, model.AppError) {
	var file *model.FileWithProfile
	var backend utils.FileBackend
	var err model.AppError

	if file, err = app.Store.File().GetFileByUuidWithProfile(domainId, uuid); err != nil {
		return nil, nil, err
	}

	if backend, err = app.GetFileBackendStore(file.ProfileId, file.ProfileUpdatedAt); err != nil {
		return nil, nil, err
	}
	//is bug ?
	return &file.File, backend, nil
}

func (app *App) RemoveFiles(domainId int64, ids []int64) model.AppError {
	files, _, err := app.SearchFiles(context.Background(), domainId, &model.SearchFile{Ids: ids})
	if err != nil {
		return err
	}

	// Soft-delete files
	if err := app.Store.File().MarkRemove(domainId, ids); err != nil {
		return err
	}

	// Notify watchers for specific files
	for _, file := range files {
		if file.Channel != nil && *file.Channel == model.UploadFileChannelCases {
			if notifyErr := app.watcherManager.Notify(
				model.PermissionScopeFiles,
				watcherkit.EventTypeDelete,
				NewFileWatcherData(file),
			); notifyErr != nil {
				wlog.Error(fmt.Sprintf("could not notify file store: %s", notifyErr.Error()))
			}
		}
	}

	return nil
}

func (app *App) MaxUploadFileSize() int64 {
	return app.Config().MediaFileStoreSettings.MaxUploadFileSize
}

func (app *App) StoreFile(src model.File) (model.File, model.AppError) {
	res := <-app.Store.File().Create(&src)
	if res.Err != nil {
		return model.File{}, res.Err
	}

	src.Id = res.Data.(int64)

	return src, nil
}
