package controller

import (
	"context"
	"github.com/webitel/engine/pkg/wbt/auth_manager"
	"github.com/webitel/storage/app"
	"github.com/webitel/storage/model"
)

func (c *Controller) DeleteFiles(session *auth_manager.Session, ids []int64) model.AppError {
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
func (c *Controller) SearchFile(ctx context.Context, session *auth_manager.Session, search *model.SearchFile) ([]*model.File, bool, model.AppError) {
	permission := session.GetPermission(model.PermissionScopeFiles)
	//if !permission.CanRead() {
	return nil, false, c.app.MakePermissionError(session, permission, auth_manager.PERMISSION_ACCESS_READ)
	//}

	if err := search.IsValid(); err != nil {
		return nil, false, err
	}

	return c.app.SearchFiles(ctx, session.Domain(0), search)
}

func (c *Controller) UploadP2PVideo(ctx context.Context, session *auth_manager.Session, file *model.JobUploadFile, offerSdp string, ice []app.ICEServer) (*app.RtcUploadVideoSession, error) {
	permission := session.GetPermission(model.PERMISSION_SCOPE_RECORD_FILE)
	if !permission.CanCreate() {
		return nil, c.app.MakePermissionError(session, permission, auth_manager.PERMISSION_ACCESS_CREATE)
	}

	return c.app.UploadP2PVideo(offerSdp, file, ice)
}

func (c *Controller) RenegotiateP2P(ctx context.Context, session *auth_manager.Session, id string, offerSdp string) (*app.RtcUploadVideoSession, error) {
	permission := session.GetPermission(model.PERMISSION_SCOPE_RECORD_FILE)
	if !permission.CanCreate() {
		return nil, c.app.MakePermissionError(session, permission, auth_manager.PERMISSION_ACCESS_CREATE)
	}

	return c.app.RenegotiateP2P(id, offerSdp)
}

func (c *Controller) CloseP2P(ctx context.Context, session *auth_manager.Session, id string) error {
	permission := session.GetPermission(model.PERMISSION_SCOPE_RECORD_FILE)
	if !permission.CanCreate() {
		return c.app.MakePermissionError(session, permission, auth_manager.PERMISSION_ACCESS_CREATE)
	}

	return c.app.CloseP2P(id)
}
