package controller

import (
	"io"

	"github.com/webitel/engine/auth_manager"
	"github.com/webitel/storage/model"
)

func (c *Controller) CreateMediaFile(session *auth_manager.Session, src io.Reader, mediaFile *model.MediaFile) (*model.MediaFile, *model.AppError) {
	//var err *model.AppError
	permission := session.GetPermission(model.PERMISSION_SCOPE_MEDIA_FILE)
	if !permission.CanCreate() {
		return nil, c.app.MakePermissionError(session, permission, auth_manager.PERMISSION_ACCESS_CREATE)
	}

	mediaFile.DomainRecord = model.DomainRecord{
		DomainId:  session.Domain(mediaFile.DomainId),
		CreatedAt: model.GetMillis(),
		CreatedBy: &model.Lookup{
			Id: int(session.UserId),
		},
		UpdatedAt: model.GetMillis(),
		UpdatedBy: &model.Lookup{
			Id: int(session.UserId),
		},
	}

	if err := mediaFile.IsValid(); err != nil {
		return nil, err
	}

	return c.app.SaveMediaFile(src, mediaFile)
}

func (c *Controller) SearchMediaFile(session *auth_manager.Session, domainId int64, search *model.SearchMediaFile) ([]*model.MediaFile, bool, *model.AppError) {
	permission := session.GetPermission(model.PERMISSION_SCOPE_MEDIA_FILE)
	if !permission.CanRead() {
		return nil, false, c.app.MakePermissionError(session, permission, auth_manager.PERMISSION_ACCESS_READ)
	}

	list, next, err := c.app.GetMediaFilePage(session.Domain(domainId), search)
	return list, next, err
}

func (c *Controller) GetMediaFile(session *auth_manager.Session, domainId int64, id int) (*model.MediaFile, *model.AppError) {
	permission := session.GetPermission(model.PERMISSION_SCOPE_MEDIA_FILE)
	if !permission.CanRead() {
		return nil, c.app.MakePermissionError(session, permission, auth_manager.PERMISSION_ACCESS_READ)
	}

	return c.app.GetMediaFile(session.Domain(domainId), id)
}

func (c *Controller) DeleteMediaFile(session *auth_manager.Session, domainId int64, id int) (*model.MediaFile, *model.AppError) {
	permission := session.GetPermission(model.PERMISSION_SCOPE_MEDIA_FILE)
	if !permission.CanRead() {
		return nil, c.app.MakePermissionError(session, permission, auth_manager.PERMISSION_ACCESS_READ)
	}

	if !permission.CanDelete() {
		return nil, c.app.MakePermissionError(session, permission, auth_manager.PERMISSION_ACCESS_DELETE)
	}

	return c.app.DeleteMediaFile(session.Domain(domainId), id)
}
