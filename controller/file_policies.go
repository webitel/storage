package controller

import (
	"context"
	"github.com/webitel/engine/auth_manager"
	engine "github.com/webitel/engine/model"
	"github.com/webitel/storage/model"
	"time"
)

func (c *Controller) CreateFilePolicy(ctx context.Context, session *auth_manager.Session, policy *model.FilePolicy) (*model.FilePolicy, engine.AppError) {
	var err engine.AppError
	permission := session.GetPermission(model.PermissionScopeFilePolicy)
	if !permission.CanCreate() {
		return nil, c.app.MakePermissionError(session, permission, auth_manager.PERMISSION_ACCESS_CREATE)
	}

	t := time.Now()
	policy.CreatedAt = &t
	policy.CreatedBy = &model.Lookup{
		Id: int(session.UserId),
	}
	policy.UpdatedAt = policy.CreatedAt
	policy.UpdatedBy = policy.CreatedBy

	if err = policy.IsValid(); err != nil {
		return nil, err
	}

	return c.app.CreateFilePolicy(ctx, session.Domain(0), policy)
}

func (c *Controller) SearchFilePolicies(ctx context.Context, session *auth_manager.Session, search *model.SearchFilePolicy) ([]*model.FilePolicy, bool, engine.AppError) {
	permission := session.GetPermission(model.PermissionScopeFilePolicy)
	if !permission.CanRead() {
		return nil, false, c.app.MakePermissionError(session, permission, auth_manager.PERMISSION_ACCESS_READ)
	}

	return c.app.SearchFilePolicies(ctx, session.Domain(0), search)
}

func (c *Controller) GetFilePolicy(ctx context.Context, session *auth_manager.Session, id int32) (*model.FilePolicy, engine.AppError) {
	permission := session.GetPermission(model.PermissionScopeFilePolicy)
	if !permission.CanRead() {
		return nil, c.app.MakePermissionError(session, permission, auth_manager.PERMISSION_ACCESS_READ)
	}

	return c.app.GetFilePolicy(ctx, session.Domain(0), id)
}

func (c *Controller) UpdateFilePolicy(ctx context.Context, session *auth_manager.Session, id int32, policy *model.FilePolicy) (*model.FilePolicy, engine.AppError) {
	var err engine.AppError
	permission := session.GetPermission(model.PermissionScopeFilePolicy)
	if !permission.CanRead() {
		return nil, c.app.MakePermissionError(session, permission, auth_manager.PERMISSION_ACCESS_READ)
	}

	if !permission.CanUpdate() {
		return nil, c.app.MakePermissionError(session, permission, auth_manager.PERMISSION_ACCESS_UPDATE)
	}

	t := time.Now()
	policy.UpdatedAt = &t
	policy.UpdatedBy = &model.Lookup{
		Id: int(session.UserId),
	}

	if err = policy.IsValid(); err != nil {
		return nil, err
	}

	return c.app.UpdateFilePolicy(ctx, session.Domain(0), id, policy)
}

func (c *Controller) PatchFilePolicy(ctx context.Context, session *auth_manager.Session, id int32, patch *model.FilePolicyPath) (*model.FilePolicy, engine.AppError) {
	permission := session.GetPermission(model.PermissionScopeFilePolicy)
	if !permission.CanRead() {
		return nil, c.app.MakePermissionError(session, permission, auth_manager.PERMISSION_ACCESS_READ)
	}

	if !permission.CanUpdate() {
		return nil, c.app.MakePermissionError(session, permission, auth_manager.PERMISSION_ACCESS_UPDATE)
	}

	patch.UpdatedAt = time.Now()
	patch.UpdatedBy = model.Lookup{
		Id: int(session.UserId),
	}

	return c.app.PatchFilePolicy(ctx, session.Domain(0), id, patch)
}

func (c *Controller) DeleteFilePolicy(ctx context.Context, session *auth_manager.Session, id int32) (*model.FilePolicy, engine.AppError) {
	permission := session.GetPermission(model.PermissionScopeFilePolicy)
	if !permission.CanRead() {
		return nil, c.app.MakePermissionError(session, permission, auth_manager.PERMISSION_ACCESS_READ)
	}

	if !permission.CanDelete() {
		return nil, c.app.MakePermissionError(session, permission, auth_manager.PERMISSION_ACCESS_DELETE)
	}

	return c.app.DeleteFilePolicy(ctx, session.Domain(0), id)
}

func (c *Controller) ChangePositionFilePolicy(ctx context.Context, session *auth_manager.Session, fromId, toId int32) engine.AppError {
	permission := session.GetPermission(model.PermissionScopeFilePolicy)
	if !permission.CanRead() {
		return c.app.MakePermissionError(session, permission, auth_manager.PERMISSION_ACCESS_READ)
	}

	if !permission.CanUpdate() {
		return c.app.MakePermissionError(session, permission, auth_manager.PERMISSION_ACCESS_DELETE)
	}

	return c.app.ChangePositionFilePolicy(ctx, session.Domain(0), fromId, toId)
}