package controller

import (
	"context"
	"strings"

	"github.com/webitel/engine/pkg/wbt/auth_manager"

	"github.com/webitel/storage/model"
)

func fileDeleteObjclass(f *model.File) string {
	if f.Channel != nil {
		switch *f.Channel {
		case model.UploadFileChannelScreenRecording:
			return model.PermissionScreenRecordings
		case model.UploadFileChannelCall:
			if isVideocallMime(f.MimeType) {
				return model.PermissionVideocallFiles
			}
		}
	}

	return model.PERMISSION_SCOPE_RECORD_FILE
}

func isVideocallMime(mime string) bool {
	return strings.HasPrefix(mime, model.ImageMimePrefix) || strings.HasPrefix(mime, model.PdfMimePrefix)
}

// ensureFilesDeletable enforces the delete permission of the object class that governs each file
func (c *Controller) ensureFilesDeletable(session *auth_manager.Session, files []*model.File) model.AppError {
	checked := make(map[string]struct{}, 3)

	for _, f := range files {
		obj := fileDeleteObjclass(f)
		if _, ok := checked[obj]; ok {
			continue
		}
		checked[obj] = struct{}{}

		permission := session.GetPermission(obj)
		if obj == model.PERMISSION_SCOPE_RECORD_FILE && !permission.CanRead() {
			return c.app.MakePermissionError(session, permission, auth_manager.PERMISSION_ACCESS_READ)
		}

		if !permission.CanDelete() {
			return c.app.MakePermissionError(session, permission, auth_manager.PERMISSION_ACCESS_DELETE)
		}
	}

	return nil
}

func (c *Controller) DeleteFiles(ctx context.Context, session *auth_manager.Session, ids []int64) model.AppError {
	if len(ids) == 0 {
		return model.NewBadRequestError("files.delete", "id is empty")
	}

	files, _, err := c.app.SearchFiles(ctx, session.Domain(0), &model.SearchFile{
		ListRequest: model.ListRequest{
			Fields:  []string{"id", "mime_type", "channel"},
			PerPage: len(ids),
		},
		Ids: ids,
	})
	if err != nil {
		return err
	}

	if err := c.ensureFilesDeletable(session, files); err != nil {
		return err
	}

	deleteIds := make([]int64, 0, len(files))
	for _, f := range files {
		deleteIds = append(deleteIds, f.Id)
	}
	if len(deleteIds) == 0 {
		return nil
	}

	return c.app.RemoveFiles(session.Domain(0), deleteIds)
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
	// if !permission.CanRead() {
	return nil, false, c.app.MakePermissionError(session, permission, auth_manager.PERMISSION_ACCESS_READ)
	//}

	if err := search.IsValid(); err != nil {
		return nil, false, err
	}

	return c.app.SearchFiles(ctx, session.Domain(0), search)
}

const (
	ScreenrecordingChannelCall = "call"
)

func screenRecordingsObjclass(screenrecordingChannel string) string {
	if screenrecordingChannel == ScreenrecordingChannelCall {
		return model.PermissionVideocallFiles
	}
	return model.PermissionScreenRecordings
}

func (c *Controller) SearchScreenRecordings(ctx context.Context, session *auth_manager.Session, search *model.SearchFile, screenrecordingChannel string) ([]*model.File, bool, model.AppError) {
	permission := session.GetPermission(screenRecordingsObjclass(screenrecordingChannel))
	if !permission.CanRead() {
		return nil, false, c.app.MakePermissionError(session, permission, auth_manager.PERMISSION_ACCESS_READ)
	}

	return c.app.SearchScreenRecordings(ctx, session.Domain(0), search, screenrecordingChannel)
}

const (
	DeleteFileChannelScreenRecording = "screenrecording"
)

func (c *Controller) DeleteScreenRecordings(ctx context.Context, session *auth_manager.Session, userId int64, ids []int64) model.AppError {
	permission := session.GetPermission(model.PermissionScreenRecordings)
	if !permission.CanDelete() {
		return c.app.MakePermissionError(session, permission, auth_manager.PERMISSION_ACCESS_DELETE)
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
		Channels:   []string{DeleteFileChannelScreenRecording},
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
	permission := session.GetPermission(model.PermissionScreenRecordings)
	if !permission.CanDelete() {
		return c.app.MakePermissionError(session, permission, auth_manager.PERMISSION_ACCESS_DELETE)
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
		Channels: []string{DeleteFileChannelScreenRecording},
	}

	if len(ids) == 0 {
		return nil
	}

	return c.app.RemoveFilesByChannels(ctx, session.Domain(0), ids, search.Channels)
}

func (c *Controller) SearchCallFiles(ctx context.Context, session *auth_manager.Session, callId string, search *model.SearchFile) ([]*model.File, bool, model.AppError) {
	if isVideocallFileSearch(search) {
		permission := session.GetPermission(model.PermissionVideocallFiles)
		if !permission.CanRead() {
			return nil, false, c.app.MakePermissionError(session, permission, auth_manager.PERMISSION_ACCESS_READ)
		}
	} else if !session.HasAction(auth_manager.PermissionRecordFile) {
		permission := session.GetPermission(model.PERMISSION_SCOPE_RECORD_FILE)
		if !permission.CanRead() {
			return nil, false, model.NewForbiddenError("call.recordings.access.forbidden", "Not allow")
		}

		// TODO RBAC ?
	}

	search.CallId = &callId
	return c.app.SearchFiles(ctx, session.Domain(0), search)
}

func (c *Controller) DeleteVideocallFiles(ctx context.Context, session *auth_manager.Session, callId string, ids []int64) model.AppError {
	permission := session.GetPermission(model.PermissionVideocallFiles)
	if !permission.CanDelete() {
		return c.app.MakePermissionError(session, permission, auth_manager.PERMISSION_ACCESS_DELETE)
	}

	if len(ids) == 0 {
		return model.NewBadRequestError("files.delete.videocall", "id is empty")
	}

	search := &model.SearchFile{
		ListRequest: model.ListRequest{
			Fields:  []string{"id", "mime_type", "channel"},
			Page:    0,
			PerPage: 1000, // TODO
		},
		Ids:      ids,
		CallId:   &callId,
		Channels: []string{model.UploadFileChannelCall},
	}

	res, _, err := c.app.SearchFiles(ctx, session.Domain(0), search)
	if err != nil {
		return err
	}

	videocallIds := make([]int64, 0, len(res))
	for _, f := range res {
		if f.Channel != nil && *f.Channel == model.UploadFileChannelCall &&
			(strings.HasPrefix(f.MimeType, model.ImageMimePrefix) || strings.HasPrefix(f.MimeType, "application/pdf")) {
			videocallIds = append(videocallIds, f.Id)
		}
	}

	if len(videocallIds) == 0 {
		return nil
	}

	return c.app.RemoveFilesByChannels(ctx, session.Domain(0), videocallIds, []string{model.UploadFileChannelCall})
}

func isVideocallFileSearch(search *model.SearchFile) bool {
	if search.MimeType == nil {
		return false
	}
	isVideocallMime := strings.HasPrefix(*search.MimeType, model.ImageMimePrefix) ||
		strings.HasPrefix(*search.MimeType, "application/pdf")
	if !isVideocallMime {
		return false
	}
	for _, ch := range search.Channels {
		if ch == model.UploadFileChannelCall {
			return true
		}
	}
	return false
}
