package app

import (
	"github.com/webitel/engine/pkg/wbt/auth_manager"
	"github.com/webitel/storage/model"
)

func (app *App) CognitiveProfileCheckAccess(domainId, id int64, groups []int, access auth_manager.PermissionAccess) (bool, model.AppError) {
	return app.Store.CognitiveProfile().CheckAccess(domainId, id, groups, access)
}

func (app *App) CreateCognitiveProfile(profile *model.CognitiveProfile) (*model.CognitiveProfile, model.AppError) {
	return app.Store.CognitiveProfile().Create(profile)
}

func (app *App) SearchCognitiveProfiles(domainId int64, search *model.SearchCognitiveProfile) ([]*model.CognitiveProfile, bool, model.AppError) {
	res, err := app.Store.CognitiveProfile().GetAllPage(domainId, search)
	if err != nil {
		return nil, false, err
	}
	search.RemoveLastElemIfNeed(&res)
	return res, search.EndOfList(), nil
}

func (app *App) SearchCognitiveProfilesByGroups(domainId int64, groups []int, search *model.SearchCognitiveProfile) ([]*model.CognitiveProfile, bool, model.AppError) {
	res, err := app.Store.CognitiveProfile().GetAllPageByGroups(domainId, groups, search)
	if err != nil {
		return nil, false, err
	}
	search.RemoveLastElemIfNeed(&res)
	return res, search.EndOfList(), nil
}

func (app *App) GetCognitiveProfile(id, domain int64) (*model.CognitiveProfile, model.AppError) {
	return app.Store.CognitiveProfile().Get(id, domain)
}

func (app *App) UpdateCognitiveProfile(profile *model.CognitiveProfile) (*model.CognitiveProfile, model.AppError) {
	oldProfile, err := app.GetCognitiveProfile(profile.Id, profile.DomainId)
	if err != nil {
		return nil, err
	}

	oldProfile.UpdatedBy = profile.UpdatedBy
	oldProfile.UpdatedAt = profile.UpdatedAt

	oldProfile.Provider = profile.Provider
	// if access_key of profile is empty do not let reset access key (task: WTEL-4344)
	if oldAccessKey, newAccessKey := oldProfile.Properties.GetString(model.CognitiveProfileKeyField), profile.Properties.GetString(model.CognitiveProfileKeyField); newAccessKey == "" {
		profile.Properties[model.CognitiveProfileKeyField] = oldAccessKey
	}
	oldProfile.Properties = profile.Properties
	oldProfile.Enabled = profile.Enabled
	oldProfile.Name = profile.Name
	oldProfile.Description = profile.Description
	oldProfile.Service = profile.Service
	oldProfile.Default = profile.Default

	return app.Store.CognitiveProfile().Update(oldProfile)

}

func (app *App) PatchCognitiveProfile(domainId, id int64, patch *model.CognitiveProfilePath) (*model.CognitiveProfile, model.AppError) {
	oldProfile, err := app.GetCognitiveProfile(id, domainId)
	if err != nil {
		return nil, err
	}

	oldProfile.Patch(patch)

	if err = oldProfile.IsValid(); err != nil {
		return nil, err
	}

	return app.Store.CognitiveProfile().Update(oldProfile)
}

func (app *App) DeleteCognitiveProfile(domainId, id int64) (*model.CognitiveProfile, model.AppError) {
	profile, err := app.GetCognitiveProfile(id, domainId)
	if err != nil {
		return nil, err
	}
	err = app.Store.CognitiveProfile().Delete(domainId, id)
	if err != nil {
		return nil, err
	}

	return profile, nil
}
