package controller

import (
	"time"

	"github.com/webitel/engine/auth_manager"
	engine "github.com/webitel/engine/model"
	"github.com/webitel/storage/model"
)

func (c *Controller) CreateImportTemplate(session *auth_manager.Session, template *model.ImportTemplate) (*model.ImportTemplate, engine.AppError) {
	var err engine.AppError
	permission := session.GetPermission(model.PermissionScopeImportTemplate)
	if !permission.CanCreate() {
		return nil, c.app.MakePermissionError(session, permission, auth_manager.PERMISSION_ACCESS_CREATE)
	}

	if err = template.IsValid(); err != nil {
		return nil, err
	}

	t := time.Now()
	template.CreatedAt = &t
	template.CreatedBy = &model.Lookup{
		Id: int(session.UserId),
	}
	template.UpdatedAt = template.CreatedAt
	template.UpdatedBy = template.CreatedBy

	return c.app.CreateImportTemplate(session.Domain(0), template)
}

func (c *Controller) SearchImportTemplates(session *auth_manager.Session, search *model.SearchImportTemplate) ([]*model.ImportTemplate, bool, engine.AppError) {
	permission := session.GetPermission(model.PermissionScopeImportTemplate)
	if !permission.CanRead() {
		return nil, false, c.app.MakePermissionError(session, permission, auth_manager.PERMISSION_ACCESS_READ)
	}

	var list []*model.ImportTemplate
	var err engine.AppError
	var endOfList bool

	if session.UseRBAC(auth_manager.PERMISSION_ACCESS_READ, permission) {
		list, endOfList, err = c.app.SearchImportTemplatesByGroup(session.Domain(0), session.RoleIds, search)
	} else {
		list, endOfList, err = c.app.SearchImportTemplates(session.Domain(0), search)
	}

	return list, endOfList, err
}

func (c *Controller) GetImportTemplate(session *auth_manager.Session, id int32) (*model.ImportTemplate, engine.AppError) {
	var err engine.AppError
	permission := session.GetPermission(model.PermissionScopeImportTemplate)
	if !permission.CanRead() {
		return nil, c.app.MakePermissionError(session, permission, auth_manager.PERMISSION_ACCESS_READ)
	}

	if session.UseRBAC(auth_manager.PERMISSION_ACCESS_READ, permission) {
		var perm bool
		if perm, err = c.app.FileBackendProfileCheckAccess(session.Domain(0), int64(id), session.RoleIds, auth_manager.PERMISSION_ACCESS_READ); err != nil {
			return nil, err
		} else if !perm {
			return nil, c.app.MakeResourcePermissionError(session, int64(id), permission, auth_manager.PERMISSION_ACCESS_READ)
		}
	}

	return c.app.GetImportTemplate(session.Domain(0), id)
}

func (c *Controller) UpdateImportTemplate(session *auth_manager.Session, template *model.ImportTemplate) (*model.ImportTemplate, engine.AppError) {
	var err engine.AppError
	permission := session.GetPermission(model.PermissionScopeImportTemplate)
	if !permission.CanRead() {
		return nil, c.app.MakePermissionError(session, permission, auth_manager.PERMISSION_ACCESS_READ)
	}

	if !permission.CanUpdate() {
		return nil, c.app.MakePermissionError(session, permission, auth_manager.PERMISSION_ACCESS_UPDATE)
	}

	if session.UseRBAC(auth_manager.PERMISSION_ACCESS_UPDATE, permission) {
		var perm bool
		if perm, err = c.app.ImportTemplateCheckAccess(session.Domain(0), template.Id, session.RoleIds, auth_manager.PERMISSION_ACCESS_UPDATE); err != nil {
			return nil, err
		} else if !perm {
			return nil, c.app.MakeResourcePermissionError(session, int64(template.Id), permission, auth_manager.PERMISSION_ACCESS_UPDATE)
		}
	}

	if err = template.IsValid(); err != nil {
		return nil, err
	}

	t := time.Now()
	template.UpdatedAt = &t
	template.UpdatedBy = &model.Lookup{
		Id: int(session.UserId),
	}

	return c.app.UpdateImportTemplate(session.Domain(0), template)
}

func (c *Controller) PatchImportTemplate(session *auth_manager.Session, id int32, patch *model.ImportTemplatePatch) (*model.ImportTemplate, engine.AppError) {
	var err engine.AppError
	permission := session.GetPermission(model.PermissionScopeImportTemplate)
	if !permission.CanRead() {
		return nil, c.app.MakePermissionError(session, permission, auth_manager.PERMISSION_ACCESS_READ)
	}

	if !permission.CanUpdate() {
		return nil, c.app.MakePermissionError(session, permission, auth_manager.PERMISSION_ACCESS_UPDATE)
	}

	if session.UseRBAC(auth_manager.PERMISSION_ACCESS_UPDATE, permission) {
		var perm bool
		if perm, err = c.app.ImportTemplateCheckAccess(session.Domain(0), id, session.RoleIds, auth_manager.PERMISSION_ACCESS_UPDATE); err != nil {
			return nil, err
		} else if !perm {
			return nil, c.app.MakeResourcePermissionError(session, int64(id), permission, auth_manager.PERMISSION_ACCESS_UPDATE)
		}
	}

	patch.UpdatedAt = time.Now()
	patch.UpdatedBy = model.Lookup{
		Id: int(session.UserId),
	}

	return c.app.PatchImportTemplate(session.Domain(0), id, patch)
}

func (c *Controller) DeleteImportTemplate(session *auth_manager.Session, id int32) (*model.ImportTemplate, engine.AppError) {
	var err engine.AppError
	permission := session.GetPermission(model.PermissionScopeImportTemplate)
	if !permission.CanRead() {
		return nil, c.app.MakePermissionError(session, permission, auth_manager.PERMISSION_ACCESS_READ)
	}

	if !permission.CanDelete() {
		return nil, c.app.MakePermissionError(session, permission, auth_manager.PERMISSION_ACCESS_DELETE)
	}

	if session.UseRBAC(auth_manager.PERMISSION_ACCESS_DELETE, permission) {
		var perm bool
		if perm, err = c.app.ImportTemplateCheckAccess(session.Domain(0), id, session.RoleIds, auth_manager.PERMISSION_ACCESS_DELETE); err != nil {
			return nil, err
		} else if !perm {
			return nil, c.app.MakeResourcePermissionError(session, int64(id), permission, auth_manager.PERMISSION_ACCESS_DELETE)
		}
	}
	return c.app.DeleteImportTemplate(session.Domain(0), id)
}
