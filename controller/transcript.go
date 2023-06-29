package controller

import (
	"github.com/webitel/engine/auth_manager"
	engine "github.com/webitel/engine/model"
	"github.com/webitel/storage/model"
)

func (c *Controller) TranscriptFiles(session *auth_manager.Session, ops *model.TranscriptOptions) ([]*model.FileTranscriptJob, engine.AppError) {
	permission := session.GetPermission(model.PERMISSION_SCOPE_RECORD_FILE)
	if !permission.CanRead() {
		return nil, c.app.MakePermissionError(session, permission, auth_manager.PERMISSION_ACCESS_READ)
	}

	if !permission.CanUpdate() {
		return nil, c.app.MakePermissionError(session, permission, auth_manager.PERMISSION_ACCESS_UPDATE)
	}

	return c.app.CreateTranscriptFilesJob(session.Domain(0), ops)
}

func (c *Controller) TranscriptFilePhrases(session *auth_manager.Session, id int64, search *model.ListRequest) ([]*model.TranscriptPhrase, bool, engine.AppError) {
	permission := session.GetPermission(model.PERMISSION_SCOPE_RECORD_FILE)
	if !permission.CanRead() {
		return nil, true, c.app.MakePermissionError(session, permission, auth_manager.PERMISSION_ACCESS_READ)
	}

	return c.app.TranscriptFilePhrases(session.Domain(0), id, search)
}

func (c *Controller) DeleteTranscript(session *auth_manager.Session, ids []int64, uuid []string) ([]int64, engine.AppError) {
	permission := session.GetPermission(model.PERMISSION_SCOPE_RECORD_FILE)
	if !permission.CanRead() {
		return nil, c.app.MakePermissionError(session, permission, auth_manager.PERMISSION_ACCESS_READ)
	}
	if !permission.CanUpdate() {
		return nil, c.app.MakePermissionError(session, permission, auth_manager.PERMISSION_ACCESS_UPDATE)
	}

	return c.app.RemoveTranscript(session.Domain(0), ids, uuid)
}
