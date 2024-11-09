package controller

import (
	"context"
	"github.com/webitel/engine/auth_manager"
	engine "github.com/webitel/engine/model"
	"github.com/webitel/storage/model"
)

func (c *Controller) DeleteFiles(session *auth_manager.Session, ids []int64) engine.AppError {
	permission := session.GetPermission(model.PERMISSION_SCOPE_RECORD_FILE)
	if !permission.CanRead() {
		return c.app.MakePermissionError(session, permission, auth_manager.PERMISSION_ACCESS_READ)
	}

	if !permission.CanDelete() {
		return c.app.MakePermissionError(session, permission, auth_manager.PERMISSION_ACCESS_DELETE)
	}

	return c.app.RemoveFiles(session.Domain(0), ids)
}

// SearchFile TODO PERMISSION (OBAC or RBAC)
func (c *Controller) SearchFile(ctx context.Context, session *auth_manager.Session, search *model.SearchFile) ([]*model.File, bool, engine.AppError) {
	permission := session.GetPermission(model.PermissionScopeFiles)
	//if !permission.CanRead() {
	return nil, false, c.app.MakePermissionError(session, permission, auth_manager.PERMISSION_ACCESS_READ)
	//}

	if err := search.IsValid(); err != nil {
		return nil, false, err
	}

	return c.app.SearchFiles(ctx, session.Domain(0), search)
}
