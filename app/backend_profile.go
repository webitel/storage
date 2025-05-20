package app

import (
	"fmt"

	"github.com/webitel/engine/pkg/wbt/auth_manager"
	"github.com/webitel/storage/model"
	"github.com/webitel/storage/utils"
	"github.com/webitel/wlog"
	"golang.org/x/sync/singleflight"
)

var (
	backendStoreGroup singleflight.Group
)

func (app *App) FileBackendProfileCheckAccess(domainId, id int64, groups []int, access auth_manager.PermissionAccess) (bool, model.AppError) {
	return app.Store.FileBackendProfile().CheckAccess(domainId, id, groups, access)
}

func (app *App) CreateFileBackendProfile(profile *model.FileBackendProfile) (*model.FileBackendProfile, model.AppError) {
	return app.Store.FileBackendProfile().Create(profile)
}

func (app *App) SearchFileBackendProfiles(domainId int64, search *model.SearchFileBackendProfile) ([]*model.FileBackendProfile, bool, model.AppError) {
	res, err := app.Store.FileBackendProfile().GetAllPage(domainId, search)
	if err != nil {
		return nil, false, err
	}
	search.RemoveLastElemIfNeed(&res)
	return res, search.EndOfList(), nil
}

func (app *App) GetFileBackendProfilePageByGroups(domainId int64, groups []int, search *model.SearchFileBackendProfile) ([]*model.FileBackendProfile, bool, model.AppError) {
	res, err := app.Store.FileBackendProfile().GetAllPageByGroups(domainId, groups, search)
	if err != nil {
		return nil, false, err
	}
	search.RemoveLastElemIfNeed(&res)
	return res, search.EndOfList(), nil
}

func (app *App) GetFileBackendProfile(id, domain int64) (*model.FileBackendProfile, model.AppError) {
	return app.Store.FileBackendProfile().Get(id, domain)
}

func (app *App) UpdateFileBackendProfile(profile *model.FileBackendProfile) (*model.FileBackendProfile, model.AppError) {
	oldProfile, err := app.GetFileBackendProfile(profile.Id, profile.DomainId)
	if err != nil {
		return nil, err
	}

	oldProfile.UpdatedBy = profile.UpdatedBy
	oldProfile.UpdatedAt = profile.UpdatedAt

	oldProfile.Name = profile.Name
	oldProfile.ExpireDay = profile.ExpireDay
	oldProfile.Priority = profile.Priority
	oldProfile.Disabled = profile.Disabled
	oldProfile.MaxSizeMb = profile.MaxSizeMb
	// if access_key of profile is empty do not let reset access key (task: WTEL-4344)
	if oldAccessKey, newAccessKey := oldProfile.Properties.GetString(model.BackendProfileAccessKeyField), profile.Properties.GetString(model.BackendProfileAccessKeyField); newAccessKey == "" {
		profile.Properties[model.BackendProfileAccessKeyField] = oldAccessKey
	}
	oldProfile.Properties = profile.Properties
	oldProfile.Description = profile.Description

	return app.Store.FileBackendProfile().Update(oldProfile)

}

func (app *App) PatchFileBackendProfile(domainId, id int64, patch *model.FileBackendProfilePath) (*model.FileBackendProfile, model.AppError) {
	oldProfile, err := app.GetFileBackendProfile(id, domainId)
	if err != nil {
		return nil, err
	}

	oldProfile.Patch(patch)

	if err = oldProfile.IsValid(); err != nil {
		return nil, err
	}

	return app.Store.FileBackendProfile().Update(oldProfile)
}

func (app *App) DeleteFileBackendProfiles(domainId, id int64) (*model.FileBackendProfile, model.AppError) {
	profile, err := app.GetFileBackendProfile(id, domainId)
	if err != nil {
		return nil, err
	}
	err = app.Store.FileBackendProfile().Delete(domainId, id)
	if err != nil {
		return nil, err
	}

	return profile, nil
}

func (app *App) GetFileBackendProfileById(id int) (*model.FileBackendProfile, model.AppError) {
	return app.Store.FileBackendProfile().GetById(id)
}

func (app *App) PathFileBackendProfile(profile *model.FileBackendProfile, path *model.FileBackendProfilePath) (*model.FileBackendProfile, model.AppError) {
	profile.Patch(path)
	profile, err := app.UpdateFileBackendProfile(profile)
	if err != nil {
		return nil, err
	}
	return profile, nil
}

func (app *App) GetFileBackendStoreById(domainId int64, id int) (store utils.FileBackend, appError model.AppError) {
	sync, err := app.Store.FileBackendProfile().GetSyncTime(domainId, id)
	if err != nil {
		return nil, err
	}

	if sync.Disabled {
		return nil, model.NewBadRequestError("app.backend_profile.valid.disabled", "profile is disabled")
	}

	return app.GetFileBackendStore(&id, &sync.UpdatedAt)
}

func (app *App) GetFileBackendStore(id *int, syncTime *int64) (store utils.FileBackend, appError model.AppError) {
	var ok bool
	var cache interface{}
	var shared bool
	var err error

	if id == nil && app.UseDefaultStore() {
		return app.DefaultFileStore, nil
	}

	if id == nil || syncTime == nil {
		return nil, model.NewInternalError("app.backend_profile.get_backend", "id or syncTime isnull")
	}

	cache, ok = app.fileBackendCache.Get(*id)
	if ok {
		store = cache.(utils.FileBackend)
		if store.GetSyncTime() == *syncTime {
			return
		}
	}

	cache, err, shared = backendStoreGroup.Do(fmt.Sprintf("backendStore-%v-%v", id, syncTime), func() (interface{}, error) {
		profile, err := app.GetFileBackendProfileById(*id)
		if err != nil {
			return nil, err
		}
		return utils.NewBackendStore(profile, app.fileChipher)
	})

	if err != nil {
		switch err.(type) {
		case model.AppError:
			return nil, err.(model.AppError)
		default:
			return nil, model.NewInternalError("app.backend_profile.get_backend", err.Error())
		}
	}

	store = cache.(utils.FileBackend)

	if !shared {
		app.fileBackendCache.Add(*id, store)
		wlog.Info("Added to cache", wlog.String("name", store.Name()))
	}

	return store, nil
}

func (app *App) SetRemoveFileJobs() model.AppError {
	return app.Store.SyncFile().SetRemoveJobs(app.DefaultFileStore.ExpireDay())
}

func (app *App) FetchFileJobs(limit int) ([]*model.SyncJob, model.AppError) {
	return app.Store.SyncFile().FetchJobs(limit)
}
func (app *App) RemoveFileJobErrors() model.AppError {
	return app.Store.SyncFile().RemoveErrors()
}
