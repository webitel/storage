package app

import (
	"context"
	engine "github.com/webitel/engine/model"
	"github.com/webitel/storage/model"
)

func (app *App) CreateFilePolicy(ctx context.Context, domainId int64, policy *model.FilePolicy) (*model.FilePolicy, engine.AppError) {
	return app.Store.FilePolicies().Create(ctx, domainId, policy)
}

func (app *App) SearchFilePolicies(ctx context.Context, domainId int64, search *model.SearchFilePolicy) ([]*model.FilePolicy, bool, engine.AppError) {
	res, err := app.Store.FilePolicies().GetAllPage(ctx, domainId, search)
	if err != nil {
		return nil, false, err
	}
	search.RemoveLastElemIfNeed(&res)
	return res, search.EndOfList(), nil
}

func (app *App) GetFilePolicy(ctx context.Context, domainId int64, id int32) (*model.FilePolicy, engine.AppError) {
	return app.Store.FilePolicies().Get(ctx, domainId, id)
}

func (app *App) ChangePositionFilePolicy(ctx context.Context, domainId int64, fromId, toId int32) engine.AppError {
	return app.Store.FilePolicies().ChangePosition(ctx, domainId, fromId, toId)
}

func (app *App) UpdateFilePolicy(ctx context.Context, domainId int64, id int32, policy *model.FilePolicy) (*model.FilePolicy, engine.AppError) {
	oldPolicy, err := app.GetFilePolicy(ctx, domainId, id)
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
	oldPolicy.MaxUploadSize = policy.MaxUploadSize

	return app.Store.FilePolicies().Update(ctx, domainId, oldPolicy)

}

func (app *App) PatchFilePolicy(ctx context.Context, domainId int64, id int32, patch *model.FilePolicyPath) (*model.FilePolicy, engine.AppError) {
	oldPolicy, err := app.GetFilePolicy(ctx, domainId, id)
	if err != nil {
		return nil, err
	}

	oldPolicy.Patch(patch)

	if err = oldPolicy.IsValid(); err != nil {
		return nil, err
	}

	return app.Store.FilePolicies().Update(ctx, domainId, oldPolicy)
}

func (app *App) DeleteFilePolicy(ctx context.Context, domainId int64, id int32) (*model.FilePolicy, engine.AppError) {
	policy, err := app.GetFilePolicy(ctx, domainId, id)
	if err != nil {
		return nil, err
	}
	err = app.Store.FilePolicies().Delete(ctx, domainId, id)
	if err != nil {
		return nil, err
	}

	return policy, nil
}

func (app *App) ApplyFilePolicy(ctx context.Context, domainId int64, id int32) (int64, engine.AppError) {
	policy, err := app.GetFilePolicy(ctx, domainId, id)
	if err != nil {
		return 0, err
	}

	if policy.RetentionDays <= 0 {
		return 0, engine.NewBadRequestError("file_policy.apply.valid.retention_days", "retention_days ")
	}

	return app.Store.FilePolicies().SetRetentionDay(ctx, domainId, policy)
}
