package controller

import (
	"context"
	"github.com/webitel/engine/pkg/wbt/auth_manager"
	"github.com/webitel/storage/model"
)

func (c *Controller) TranscriptFiles(session *auth_manager.Session, ops *model.TranscriptOptions) ([]*model.FileTranscriptJob, model.AppError) {
	permission := session.GetPermission(model.PERMISSION_SCOPE_RECORD_FILE)
	if !permission.CanRead() {
		return nil, c.app.MakePermissionError(session, permission, auth_manager.PERMISSION_ACCESS_READ)
	}

	if !permission.CanUpdate() {
		return nil, c.app.MakePermissionError(session, permission, auth_manager.PERMISSION_ACCESS_UPDATE)
	}

	return c.app.CreateTranscriptFilesJob(session.Domain(0), ops)
}

func (c *Controller) TranscriptFilesSafe(fileId int64, ops *model.TranscriptOptions) (*model.FileTranscript, model.AppError) {
	return c.app.TranscriptFile(fileId, *ops)
}

func (c *Controller) GetProfileWithoutAuth(domainId int64, profileId int64) (*model.CognitiveProfile, model.AppError) {
	return c.app.Store.CognitiveProfile().Get(profileId, domainId)
}

func (c *Controller) TranscriptFilePhrases(session *auth_manager.Session, id int64, search *model.ListRequest) ([]*model.TranscriptPhrase, bool, model.AppError) {
	permission := session.GetPermission(model.PERMISSION_SCOPE_RECORD_FILE)
	if !permission.CanRead() {
		return nil, true, c.app.MakePermissionError(session, permission, auth_manager.PERMISSION_ACCESS_READ)
	}

	return c.app.TranscriptFilePhrases(session.Domain(0), id, search)
}

func (c *Controller) DeleteTranscript(session *auth_manager.Session, ids []int64, uuid []string) ([]int64, model.AppError) {
	permission := session.GetPermission(model.PERMISSION_SCOPE_RECORD_FILE)
	if !permission.CanRead() {
		return nil, c.app.MakePermissionError(session, permission, auth_manager.PERMISSION_ACCESS_READ)
	}
	if !permission.CanUpdate() {
		return nil, c.app.MakePermissionError(session, permission, auth_manager.PERMISSION_ACCESS_UPDATE)
	}

	return c.app.RemoveTranscript(session.Domain(0), ids, uuid)
}

func (c *Controller) PutTranscript(ctx context.Context, session *auth_manager.Session, uuid string, tr model.FileTranscript) (int64, model.AppError) {
	permission := session.GetPermission(model.PERMISSION_SCOPE_RECORD_FILE)
	if !permission.CanRead() {
		return 0, c.app.MakePermissionError(session, permission, auth_manager.PERMISSION_ACCESS_READ)
	}
	if !permission.CanUpdate() {
		return 0, c.app.MakePermissionError(session, permission, auth_manager.PERMISSION_ACCESS_UPDATE)
	}

	return c.app.PutTranscript(ctx, session.Domain(0), uuid, tr)
}
