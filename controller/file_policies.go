package controller

import (
	"github.com/webitel/engine/auth_manager"
	engine "github.com/webitel/engine/model"
	"github.com/webitel/storage/model"
	"time"
)

func (c *Controller) CreateFilePolicy(session *auth_manager.Session, policy *model.FilePolicy) (*model.FilePolicy, engine.AppError) {
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

	return c.app.CreateFilePolicy(session.Domain(0), policy)
}

func (c *Controller) SearchFilePolicies(session *auth_manager.Session, search *model.SearchFilePolicy) ([]*model.FilePolicy, bool, engine.AppError) {
	permission := session.GetPermission(model.PermissionScopeFilePolicy)
	if !permission.CanRead() {
		return nil, false, c.app.MakePermissionError(session, permission, auth_manager.PERMISSION_ACCESS_READ)
	}

	var list []*model.FilePolicy
	var err engine.AppError
	var endOfList bool

	if session.UseRBAC(auth_manager.PERMISSION_ACCESS_READ, permission) {
		list, endOfList, err = c.app.SearchFilePoliciesByGroups(session.Domain(0), session.RoleIds, search)
	} else {
		list, endOfList, err = c.app.SearchFilePolicies(session.Domain(0), search)
	}

	return list, endOfList, err
}

func (c *Controller) GetFilePolicy(session *auth_manager.Session, id int32) (*model.FilePolicy, engine.AppError) {
	var err engine.AppError
	permission := session.GetPermission(model.PermissionScopeFilePolicy)
	if !permission.CanRead() {
		return nil, c.app.MakePermissionError(session, permission, auth_manager.PERMISSION_ACCESS_READ)
	}

	if session.UseRBAC(auth_manager.PERMISSION_ACCESS_READ, permission) {
		var perm bool
		if perm, err = c.app.FilePolicyCheckAccess(session.Domain(0), id, session.RoleIds, auth_manager.PERMISSION_ACCESS_READ); err != nil {
			return nil, err
		} else if !perm {
			return nil, c.app.MakeResourcePermissionError(session, int64(id), permission, auth_manager.PERMISSION_ACCESS_READ)
		}
	}

	return c.app.GetFilePolicy(session.Domain(0), id)
}

func (c *Controller) UpdateFilePolicy(session *auth_manager.Session, id int32, policy *model.FilePolicy) (*model.FilePolicy, engine.AppError) {
	var err engine.AppError
	permission := session.GetPermission(model.PermissionScopeFilePolicy)
	if !permission.CanRead() {
		return nil, c.app.MakePermissionError(session, permission, auth_manager.PERMISSION_ACCESS_READ)
	}

	if !permission.CanUpdate() {
		return nil, c.app.MakePermissionError(session, permission, auth_manager.PERMISSION_ACCESS_UPDATE)
	}

	if session.UseRBAC(auth_manager.PERMISSION_ACCESS_UPDATE, permission) {
		var perm bool
		if perm, err = c.app.FilePolicyCheckAccess(session.Domain(0), id, session.RoleIds, auth_manager.PERMISSION_ACCESS_READ); err != nil {
			return nil, err
		} else if !perm {
			return nil, c.app.MakeResourcePermissionError(session, int64(id), permission, auth_manager.PERMISSION_ACCESS_READ)
		}
	}
	t := time.Now()
	policy.UpdatedAt = &t
	policy.UpdatedBy = &model.Lookup{
		Id: int(session.UserId),
	}

	if err = policy.IsValid(); err != nil {
		return nil, err
	}

	return c.app.UpdateFilePolicy(session.Domain(0), id, policy)
}

func (c *Controller) PatchFilePolicy(session *auth_manager.Session, id int32, patch *model.FilePolicyPath) (*model.FilePolicy, engine.AppError) {
	var err engine.AppError
	permission := session.GetPermission(model.PermissionScopeFilePolicy)
	if !permission.CanRead() {
		return nil, c.app.MakePermissionError(session, permission, auth_manager.PERMISSION_ACCESS_READ)
	}

	if !permission.CanUpdate() {
		return nil, c.app.MakePermissionError(session, permission, auth_manager.PERMISSION_ACCESS_UPDATE)
	}

	if session.UseRBAC(auth_manager.PERMISSION_ACCESS_UPDATE, permission) {
		var perm bool
		if perm, err = c.app.FilePolicyCheckAccess(session.Domain(0), id, session.RoleIds, auth_manager.PERMISSION_ACCESS_READ); err != nil {
			return nil, err
		} else if !perm {
			return nil, c.app.MakeResourcePermissionError(session, int64(id), permission, auth_manager.PERMISSION_ACCESS_READ)
		}
	}

	patch.UpdatedAt = time.Now()
	patch.UpdatedBy = model.Lookup{
		Id: int(session.UserId),
	}

	return c.app.PatchFilePolicy(session.Domain(0), id, patch)
}

func (c *Controller) DeleteFilePolicy(session *auth_manager.Session, id int32) (*model.FilePolicy, engine.AppError) {
	var err engine.AppError
	permission := session.GetPermission(model.PermissionScopeFilePolicy)
	if !permission.CanRead() {
		return nil, c.app.MakePermissionError(session, permission, auth_manager.PERMISSION_ACCESS_READ)
	}

	if !permission.CanDelete() {
		return nil, c.app.MakePermissionError(session, permission, auth_manager.PERMISSION_ACCESS_DELETE)
	}

	if session.UseRBAC(auth_manager.PERMISSION_ACCESS_DELETE, permission) {
		var perm bool
		if perm, err = c.app.FilePolicyCheckAccess(session.Domain(0), id, session.RoleIds, auth_manager.PERMISSION_ACCESS_DELETE); err != nil {
			return nil, err
		} else if !perm {
			return nil, c.app.MakeResourcePermissionError(session, int64(id), permission, auth_manager.PERMISSION_ACCESS_DELETE)
		}
	}

	return c.app.DeleteFilePolicy(session.Domain(0), id)
}
