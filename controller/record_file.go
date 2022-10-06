package controller

import (
	"io"

	"github.com/webitel/engine/auth_manager"
	"github.com/webitel/storage/model"
	"github.com/webitel/storage/utils"
)

func (c *Controller) GetFileWithProfile(session *auth_manager.Session, domainId, id int64) (*model.File, utils.FileBackend, *model.AppError) {
	permission := session.GetPermission(model.PERMISSION_SCOPE_RECORD_FILE)
	if !permission.CanRead() {
		//FIXME
		//return nil, nil, c.app.MakePermissionError(session, permission, auth_manager.PERMISSION_ACCESS_READ)
	}

	return c.app.GetFileWithProfile(session.Domain(domainId), id)
}

func (c *Controller) UploadFileStream(src io.ReadCloser, file *model.JobUploadFile) *model.AppError {
	return c.app.SyncUpload(src, file)
}

func (c *Controller) GeneratePreSignetResourceSignature(resource, action string, id int64, domainId int64) (string, *model.AppError) {
	return c.app.GeneratePreSignetResourceSignature(resource, action, id, domainId)
}

func (c *Controller) InsecureGetFileWithProfile(domainId, id int64) (*model.File, utils.FileBackend, *model.AppError) {
	return c.app.GetFileWithProfile(domainId, id)
}
