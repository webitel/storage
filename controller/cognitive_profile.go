package controller

import (
	"time"

	"github.com/webitel/engine/pkg/wbt/auth_manager"
	"github.com/webitel/storage/model"
)

func (c *Controller) CreateCognitiveProfile(session *auth_manager.Session, profile *model.CognitiveProfile) (*model.CognitiveProfile, model.AppError) {
	var err model.AppError
	permission := session.GetPermission(model.PermissionScopeCognitiveProfile)
	if !permission.CanCreate() {
		return nil, c.app.MakePermissionError(session, permission, auth_manager.PERMISSION_ACCESS_CREATE)
	}

	profile.DomainId = session.Domain(profile.DomainId)
	t := time.Now()
	profile.CreatedAt = &t
	profile.CreatedBy = &model.Lookup{
		Id: int(session.UserId),
	}
	profile.UpdatedAt = profile.CreatedAt
	profile.UpdatedBy = profile.CreatedBy

	if err = profile.IsValid(); err != nil {
		return nil, err
	}

	return c.app.CreateCognitiveProfile(profile)
}

func (c *Controller) SearchCognitiveProfile(session *auth_manager.Session, domainId int64, search *model.SearchCognitiveProfile) ([]*model.CognitiveProfile, bool, model.AppError) {
	permission := session.GetPermission(model.PermissionScopeCognitiveProfile)
	if !permission.CanRead() {
		return nil, false, c.app.MakePermissionError(session, permission, auth_manager.PERMISSION_ACCESS_READ)
	}

	var list []*model.CognitiveProfile
	var err model.AppError
	var endOfList bool

	if session.UseRBAC(auth_manager.PERMISSION_ACCESS_READ, permission) {
		list, endOfList, err = c.app.SearchCognitiveProfilesByGroups(session.Domain(domainId), session.RoleIds, search)
	} else {
		list, endOfList, err = c.app.SearchCognitiveProfiles(session.Domain(domainId), search)
	}

	return list, endOfList, err
}

func (c *Controller) GetCognitiveProfile(session *auth_manager.Session, id int64, domainId int64) (*model.CognitiveProfile, model.AppError) {
	var err model.AppError
	permission := session.GetPermission(model.PermissionScopeCognitiveProfile)
	if !permission.CanRead() {
		return nil, c.app.MakePermissionError(session, permission, auth_manager.PERMISSION_ACCESS_READ)
	}

	if session.UseRBAC(auth_manager.PERMISSION_ACCESS_READ, permission) {
		var perm bool
		if perm, err = c.app.CognitiveProfileCheckAccess(session.Domain(domainId), id, session.RoleIds, auth_manager.PERMISSION_ACCESS_READ); err != nil {
			return nil, err
		} else if !perm {
			return nil, c.app.MakeResourcePermissionError(session, id, permission, auth_manager.PERMISSION_ACCESS_READ)
		}
	}

	return c.app.GetCognitiveProfile(id, session.Domain(domainId))
}

func (c *Controller) UpdateCognitiveProfile(session *auth_manager.Session, profile *model.CognitiveProfile) (*model.CognitiveProfile, model.AppError) {
	var err model.AppError
	permission := session.GetPermission(model.PermissionScopeCognitiveProfile)
	if !permission.CanRead() {
		return nil, c.app.MakePermissionError(session, permission, auth_manager.PERMISSION_ACCESS_READ)
	}

	if !permission.CanUpdate() {
		return nil, c.app.MakePermissionError(session, permission, auth_manager.PERMISSION_ACCESS_UPDATE)
	}

	if session.UseRBAC(auth_manager.PERMISSION_ACCESS_UPDATE, permission) {
		var perm bool
		if perm, err = c.app.CognitiveProfileCheckAccess(session.Domain(profile.DomainId), profile.Id, session.RoleIds, auth_manager.PERMISSION_ACCESS_READ); err != nil {
			return nil, err
		} else if !perm {
			return nil, c.app.MakeResourcePermissionError(session, profile.Id, permission, auth_manager.PERMISSION_ACCESS_READ)
		}
	}
	t := time.Now()
	profile.UpdatedAt = &t
	profile.UpdatedBy = &model.Lookup{
		Id: int(session.UserId),
	}

	if err = profile.IsValid(); err != nil {
		return nil, err
	}

	return c.app.UpdateCognitiveProfile(profile)
}

func (c *Controller) PatchCognitiveProfile(session *auth_manager.Session, domainId, id int64, patch *model.CognitiveProfilePath) (*model.CognitiveProfile, model.AppError) {
	var err model.AppError
	permission := session.GetPermission(model.PermissionScopeCognitiveProfile)
	if !permission.CanRead() {
		return nil, c.app.MakePermissionError(session, permission, auth_manager.PERMISSION_ACCESS_READ)
	}

	if !permission.CanUpdate() {
		return nil, c.app.MakePermissionError(session, permission, auth_manager.PERMISSION_ACCESS_UPDATE)
	}

	if session.UseRBAC(auth_manager.PERMISSION_ACCESS_UPDATE, permission) {
		var perm bool
		if perm, err = c.app.CognitiveProfileCheckAccess(session.Domain(domainId), id, session.RoleIds, auth_manager.PERMISSION_ACCESS_READ); err != nil {
			return nil, err
		} else if !perm {
			return nil, c.app.MakeResourcePermissionError(session, id, permission, auth_manager.PERMISSION_ACCESS_READ)
		}
	}

	patch.UpdatedAt = time.Now()
	patch.UpdatedBy = model.Lookup{
		Id: int(session.UserId),
	}

	return c.app.PatchCognitiveProfile(session.Domain(domainId), id, patch)
}

func (c *Controller) DeleteCognitiveProfile(session *auth_manager.Session, domainId, id int64) (*model.CognitiveProfile, model.AppError) {
	var err model.AppError
	permission := session.GetPermission(model.PermissionScopeCognitiveProfile)
	if !permission.CanRead() {
		return nil, c.app.MakePermissionError(session, permission, auth_manager.PERMISSION_ACCESS_READ)
	}

	if !permission.CanDelete() {
		return nil, c.app.MakePermissionError(session, permission, auth_manager.PERMISSION_ACCESS_DELETE)
	}

	if session.UseRBAC(auth_manager.PERMISSION_ACCESS_DELETE, permission) {
		var perm bool
		if perm, err = c.app.CognitiveProfileCheckAccess(session.Domain(domainId), id, session.RoleIds, auth_manager.PERMISSION_ACCESS_DELETE); err != nil {
			return nil, err
		} else if !perm {
			return nil, c.app.MakeResourcePermissionError(session, id, permission, auth_manager.PERMISSION_ACCESS_DELETE)
		}
	}

	return c.app.DeleteCognitiveProfile(session.Domain(domainId), id)
}
