package app

import (
	"github.com/webitel/engine/auth_manager"
	engine "github.com/webitel/engine/model"
	"github.com/webitel/storage/model"
)

func (app *App) FilePolicyCheckAccess(domainId int64, id int32, groups []int, access auth_manager.PermissionAccess) (bool, engine.AppError) {
	return app.Store.FilePolicies().CheckAccess(domainId, id, groups, access)
}

func (app *App) CreateFilePolicy(domainId int64, policy *model.FilePolicy) (*model.FilePolicy, engine.AppError) {
	return app.Store.FilePolicies().Create(domainId, policy)
}

func (app *App) SearchFilePolicies(domainId int64, search *model.SearchFilePolicy) ([]*model.FilePolicy, bool, engine.AppError) {
	res, err := app.Store.FilePolicies().GetAllPage(domainId, search)
	if err != nil {
		return nil, false, err
	}
	search.RemoveLastElemIfNeed(&res)
	return res, search.EndOfList(), nil
}

func (app *App) SearchFilePoliciesByGroups(domainId int64, groups []int, search *model.SearchFilePolicy) ([]*model.FilePolicy, bool, engine.AppError) {
	res, err := app.Store.FilePolicies().GetAllPageByGroups(domainId, groups, search)
	if err != nil {
		return nil, false, err
	}
	search.RemoveLastElemIfNeed(&res)
	return res, search.EndOfList(), nil
}

func (app *App) GetFilePolicy(domainId int64, id int32) (*model.FilePolicy, engine.AppError) {
	return app.Store.FilePolicies().Get(domainId, id)
}

func (app *App) UpdateFilePolicy(domainId int64, id int32, policy *model.FilePolicy) (*model.FilePolicy, engine.AppError) {
	oldPolicy, err := app.GetFilePolicy(domainId, id)
	if err != nil {
		return nil, err
	}

	oldPolicy.UpdatedBy = policy.UpdatedBy
	oldPolicy.UpdatedAt = policy.UpdatedAt

	oldPolicy.Enabled = policy.Enabled
	oldPolicy.Name = policy.Name
	oldPolicy.Description = policy.Description
	oldPolicy.MimeTypes = policy.MimeTypes
	oldPolicy.Channels = policy.Channels
	oldPolicy.SpeedUpload = policy.SpeedUpload
	oldPolicy.SpeedDownload = policy.SpeedDownload
	oldPolicy.RetentionDays = policy.RetentionDays

	return app.Store.FilePolicies().Update(domainId, oldPolicy)

}

func (app *App) PatchFilePolicy(domainId int64, id int32, patch *model.FilePolicyPath) (*model.FilePolicy, engine.AppError) {
	oldPolicy, err := app.GetFilePolicy(domainId, id)
	if err != nil {
		return nil, err
	}

	oldPolicy.Patch(patch)

	if err = oldPolicy.IsValid(); err != nil {
		return nil, err
	}

	return app.Store.FilePolicies().Update(domainId, oldPolicy)
}

func (app *App) DeleteFilePolicy(domainId int64, id int32) (*model.FilePolicy, engine.AppError) {
	policy, err := app.GetFilePolicy(domainId, id)
	if err != nil {
		return nil, err
	}
	err = app.Store.FilePolicies().Delete(domainId, id)
	if err != nil {
		return nil, err
	}

	return policy, nil
}
