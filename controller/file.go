package controller

import (
	"context"
	"github.com/webitel/engine/pkg/wbt/auth_manager"
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

func (c *Controller) DeleteQuarantineFiles(session *auth_manager.Session, ids []int64) model.AppError {
	permission := session.GetPermission(model.PERMISSION_SCOPE_RECORD_FILE)
	if !permission.CanRead() {
		return c.app.MakePermissionError(session, permission, auth_manager.PERMISSION_ACCESS_READ)
	}

	if !permission.CanDelete() {
		return c.app.MakePermissionError(session, permission, auth_manager.PERMISSION_ACCESS_DELETE)
	}

	return c.app.RemoveQuarantineFiles(session.Domain(0), ids)
}

func (c *Controller) RestoreFiles(ctx context.Context, session *auth_manager.Session, ids []int64) model.AppError {
	permission := session.GetPermission(model.PERMISSION_SCOPE_RECORD_FILE)
	if !permission.CanRead() {
		return c.app.MakePermissionError(session, permission, auth_manager.PERMISSION_ACCESS_READ)
	}

	if !permission.CanUpdate() {
		return c.app.MakePermissionError(session, permission, auth_manager.PERMISSION_ACCESS_UPDATE)
	}

	return c.app.RestoreFiles(ctx, session.Domain(0), ids, session.UserId)
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

var errNoActionSearchScreenRecordings = model.NewForbiddenError("files.search.screen", "You don't have access to control_agent_screen action")

const (
	PermissionControlAgentScreen = "control_agent_screen"
)

func (c *Controller) SearchScreenRecordings(ctx context.Context, session *auth_manager.Session, search *model.SearchFile) ([]*model.File, bool, model.AppError) {
	if !session.HasAction(PermissionControlAgentScreen) {
		return nil, false, errNoActionSearchScreenRecordings
	}
	return c.app.SearchFiles(ctx, session.Domain(0), search)
}

func (c *Controller) DeleteScreenRecordings(ctx context.Context, session *auth_manager.Session, userId int64, ids []int64) model.AppError {
	if !session.HasAction(PermissionControlAgentScreen) {
		return errNoActionSearchScreenRecordings
	}

	if len(ids) == 0 {
		return model.NewBadRequestError("files.delete.screen", "id is empty")
	}

	search := &model.SearchFile{
		ListRequest: model.ListRequest{
			Fields:  []string{"id"},
			Page:    0,
			PerPage: 1000, // TODO
		},
		Ids:        ids,
		UploadedBy: []int64{userId},
		Channels:   []string{model.UploadFileChannelScreenshot, model.UploadFileChannelScreenShare},
	}

	res, _, err := c.app.SearchFiles(ctx, session.Domain(0), search)
	if err != nil {
		return err
	}

	ids = make([]int64, 0, len(res))
	for _, v := range res {
		ids = append(ids, v.Id)
	}

	if len(ids) == 0 {
		return nil
	}

	return c.app.RemoveFilesByChannels(ctx, session.Domain(0), ids, search.Channels)
}

func (c *Controller) DeleteScreenRecordingsByAgent(ctx context.Context, session *auth_manager.Session, agentId int, ids []int64) model.AppError {
	if !session.HasAction(PermissionControlAgentScreen) {
		return errNoActionSearchScreenRecordings
	}

	if len(ids) == 0 {
		return model.NewBadRequestError("files.delete.screen", "id is empty")
	}

	search := &model.SearchFile{
		ListRequest: model.ListRequest{
			Fields:  []string{"id"},
			Page:    0,
			PerPage: 1000, // TODO
		},
		Ids:      ids,
		AgentIds: []int{agentId},
		Channels: []string{model.UploadFileChannelScreenshot, model.UploadFileChannelScreenShare},
	}

	res, _, err := c.app.SearchFiles(ctx, session.Domain(0), search)
	if err != nil {
		return err
	}

	ids = make([]int64, 0, len(res))
	for _, v := range res {
		ids = append(ids, v.Id)
	}

	if len(ids) == 0 {
		return nil
	}

	return c.app.RemoveFilesByChannels(ctx, session.Domain(0), ids, search.Channels)
}
