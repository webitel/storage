package controller

import (
	"github.com/webitel/engine/pkg/wbt/auth_manager"
	"github.com/webitel/storage/model"
)

func (c *Controller) CreateBackendProfile(session *auth_manager.Session, profile *model.FileBackendProfile) (*model.FileBackendProfile, model.AppError) {
	var err model.AppError
	permission := session.GetPermission(model.PERMISSION_SCOPE_BACKEND_PROFILE)
	if !permission.CanCreate() {
		return nil, c.app.MakePermissionError(session, permission, auth_manager.PERMISSION_ACCESS_CREATE)
	}

	profile.DomainRecord = model.DomainRecord{
		DomainId:  session.Domain(profile.DomainId),
		CreatedAt: model.GetMillis(),
		CreatedBy: &model.Lookup{
			Id: int(session.UserId),
		},
		UpdatedAt: model.GetMillis(),
		UpdatedBy: &model.Lookup{
			Id: int(session.UserId),
		},
	}

	if err = profile.IsValid(); err != nil {
		return nil, err
	}

	return c.app.CreateFileBackendProfile(profile)
}

func (c *Controller) SearchBackendProfile(session *auth_manager.Session, domainId int64, search *model.SearchFileBackendProfile) ([]*model.FileBackendProfile, bool, model.AppError) {
	permission := session.GetPermission(model.PERMISSION_SCOPE_BACKEND_PROFILE)
	if !permission.CanRead() {
		return nil, false, c.app.MakePermissionError(session, permission, auth_manager.PERMISSION_ACCESS_READ)
	}

	var list []*model.FileBackendProfile
	var err model.AppError
	var endOfList bool

	// TODO DELME
	if session.UseRBAC(auth_manager.PERMISSION_ACCESS_READ, permission) {
		list, endOfList, err = c.app.GetFileBackendProfilePageByGroups(session.Domain(domainId), session.RoleIds, search)
	} else {
		list, endOfList, err = c.app.SearchFileBackendProfiles(session.Domain(domainId), search)
	}

	return list, endOfList, err
}

func (c *Controller) GetBackendProfile(session *auth_manager.Session, id int64, domainId int64) (*model.FileBackendProfile, model.AppError) {
	var err model.AppError
	permission := session.GetPermission(model.PERMISSION_SCOPE_BACKEND_PROFILE)
	if !permission.CanRead() {
		return nil, c.app.MakePermissionError(session, permission, auth_manager.PERMISSION_ACCESS_READ)
	}

	if session.UseRBAC(auth_manager.PERMISSION_ACCESS_READ, permission) {
		var perm bool
		if perm, err = c.app.FileBackendProfileCheckAccess(session.Domain(domainId), id, session.RoleIds, auth_manager.PERMISSION_ACCESS_READ); err != nil {
			return nil, err
		} else if !perm {
			return nil, c.app.MakeResourcePermissionError(session, id, permission, auth_manager.PERMISSION_ACCESS_READ)
		}
	}

	return c.app.GetFileBackendProfile(id, session.Domain(domainId))
}

func (c *Controller) UpdateBackendProfile(session *auth_manager.Session, profile *model.FileBackendProfile) (*model.FileBackendProfile, model.AppError) {
	var err model.AppError
	permission := session.GetPermission(model.PERMISSION_SCOPE_BACKEND_PROFILE)
	if !permission.CanRead() {
		return nil, c.app.MakePermissionError(session, permission, auth_manager.PERMISSION_ACCESS_READ)
	}

	if !permission.CanUpdate() {
		return nil, c.app.MakePermissionError(session, permission, auth_manager.PERMISSION_ACCESS_UPDATE)
	}

	if session.UseRBAC(auth_manager.PERMISSION_ACCESS_UPDATE, permission) {
		var perm bool
		if perm, err = c.app.FileBackendProfileCheckAccess(session.Domain(profile.DomainId), profile.Id, session.RoleIds, auth_manager.PERMISSION_ACCESS_UPDATE); err != nil {
			return nil, err
		} else if !perm {
			return nil, c.app.MakeResourcePermissionError(session, profile.Id, permission, auth_manager.PERMISSION_ACCESS_UPDATE)
		}
	}

	profile.UpdatedAt = model.GetMillis()
	profile.UpdatedBy = &model.Lookup{
		Id: int(session.UserId),
	}

	if err = profile.IsValid(); err != nil {
		return nil, err
	}

	return c.app.UpdateFileBackendProfile(profile)
}

func (c *Controller) PatchBackendProfile(session *auth_manager.Session, domainId, id int64, patch *model.FileBackendProfilePath) (*model.FileBackendProfile, model.AppError) {
	var err model.AppError
	permission := session.GetPermission(model.PERMISSION_SCOPE_BACKEND_PROFILE)
	if !permission.CanRead() {
		return nil, c.app.MakePermissionError(session, permission, auth_manager.PERMISSION_ACCESS_READ)
	}

	if !permission.CanUpdate() {
		return nil, c.app.MakePermissionError(session, permission, auth_manager.PERMISSION_ACCESS_UPDATE)
	}

	if session.UseRBAC(auth_manager.PERMISSION_ACCESS_UPDATE, permission) {
		var perm bool
		if perm, err = c.app.FileBackendProfileCheckAccess(session.Domain(domainId), id, session.RoleIds, auth_manager.PERMISSION_ACCESS_UPDATE); err != nil {
			return nil, err
		} else if !perm {
			return nil, c.app.MakeResourcePermissionError(session, id, permission, auth_manager.PERMISSION_ACCESS_UPDATE)
		}
	}

	patch.UpdatedAt = model.GetMillis()
	patch.UpdatedBy = &model.Lookup{
		Id: int(session.UserId),
	}

	return c.app.PatchFileBackendProfile(session.Domain(domainId), id, patch)
}

func (c *Controller) DeleteBackendProfile(session *auth_manager.Session, domainId, id int64) (*model.FileBackendProfile, model.AppError) {
	var err model.AppError
	permission := session.GetPermission(model.PERMISSION_SCOPE_BACKEND_PROFILE)
	if !permission.CanRead() {
		return nil, c.app.MakePermissionError(session, permission, auth_manager.PERMISSION_ACCESS_READ)
	}

	if !permission.CanDelete() {
		return nil, c.app.MakePermissionError(session, permission, auth_manager.PERMISSION_ACCESS_DELETE)
	}

	if session.UseRBAC(auth_manager.PERMISSION_ACCESS_DELETE, permission) {
		var perm bool
		if perm, err = c.app.FileBackendProfileCheckAccess(session.Domain(domainId), id, session.RoleIds, auth_manager.PERMISSION_ACCESS_DELETE); err != nil {
			return nil, err
		} else if !perm {
			return nil, c.app.MakeResourcePermissionError(session, id, permission, auth_manager.PERMISSION_ACCESS_DELETE)
		}
	}

	return c.app.DeleteFileBackendProfiles(session.Domain(domainId), id)
}
