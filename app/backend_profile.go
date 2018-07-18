package app

import (
	"github.com/webitel/storage/mlog"
	"github.com/webitel/storage/model"
	"github.com/webitel/storage/utils"
)

func (app *App) CreateFileBackendProfile(profile *model.FileBackendProfile) (*model.FileBackendProfile, *model.AppError) {
	if err := profile.IsValid(); err != nil {
		return nil, err
	}

	if result := <-app.Store.FileBackendProfile().Save(profile); result.Err != nil {
		return nil, result.Err
	} else {
		return result.Data.(*model.FileBackendProfile), nil
	}
}

func (app *App) GetFileBackendProfile(id int, domain string) (*model.FileBackendProfile, *model.AppError) {
	if result := <-app.Store.FileBackendProfile().Get(id, domain); result.Err != nil {
		return nil, result.Err
	} else {
		return result.Data.(*model.FileBackendProfile), nil
	}
}

func (app *App) GetFileBackendProfileById(id int) (*model.FileBackendProfile, *model.AppError) {
	if result := <-app.Store.FileBackendProfile().GetById(id); result.Err != nil {
		return nil, result.Err
	} else {
		return result.Data.(*model.FileBackendProfile), nil
	}
}

func (app *App) ListFileBackendProfiles(domain string, page, perPage int) ([]*model.FileBackendProfile, *model.AppError) {
	if result := <-app.Store.FileBackendProfile().List(domain, page*perPage, perPage); result.Err != nil {
		return nil, result.Err
	} else {
		return result.Data.([]*model.FileBackendProfile), nil
	}
}

func (app *App) RemoveFileBackendProfiles(id int, domain string) *model.AppError {
	result := <-app.Store.FileBackendProfile().Delete(id, domain)
	return result.Err
}

func (app *App) UpdateFileBackendProfile(profile *model.FileBackendProfile) (*model.FileBackendProfile, *model.AppError) {
	if err := profile.IsValid(); err != nil {
		return nil, err
	}

	if result := <-app.Store.FileBackendProfile().Update(profile); result.Err != nil {
		return nil, result.Err
	} else {
		return profile, nil
	}
}

func (app *App) PathFileBackendProfile(profile *model.FileBackendProfile, path *model.FileBackendProfilePath) (*model.FileBackendProfile, *model.AppError) {
	profile.Path(path)
	profile, err := app.UpdateFileBackendProfile(profile)
	if err != nil {
		return nil, err
	}
	return profile, nil
}

func (app *App) GetFileBackendStore(id int, syncTime int64) (store utils.FileBackend, appError *model.AppError) {
	var ok bool
	var cache interface{}
	cache, ok = app.fileBackendCache.Get(id)
	if ok {
		store = cache.(utils.FileBackend)
		if store.GetSyncTime() != syncTime {
			store = nil
		} else {
			return
		}
	}

	if store == nil {
		var profile *model.FileBackendProfile
		profile, appError = app.GetFileBackendProfileById(id)
		if appError != nil {
			return
		}
		store, appError = utils.NewBackendStore(profile)
	}

	if appError != nil {
		return
	}

	app.fileBackendCache.Add(id, store)
	mlog.Info("Added to cache", mlog.String("name", store.Name()))
	return store, nil
}
