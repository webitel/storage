package controller

import (
	"io"

	"github.com/webitel/engine/auth_manager"
	engine "github.com/webitel/engine/model"
	"github.com/webitel/storage/model"
	"github.com/webitel/storage/utils"
)

func (c *Controller) GetFileWithProfile(session *auth_manager.Session, domainId, id int64) (*model.File, utils.FileBackend, engine.AppError) {
	permission := session.GetPermission(model.PERMISSION_SCOPE_RECORD_FILE)
	if !permission.CanRead() {
		//FIXME
		//return nil, nil, c.app.MakePermissionError(session, permission, auth_manager.PERMISSION_ACCESS_READ)
	}

	return c.app.GetFileWithProfile(session.Domain(domainId), id)
}

func (c *Controller) UploadFileStream(src io.ReadCloser, file *model.JobUploadFile) engine.AppError {
	return c.app.SyncUpload(src, file)
}

func (c *Controller) UploadFileStreamToProfile(src io.ReadCloser, profileId int, file *model.JobUploadFile) engine.AppError {
	return c.app.SyncUploadToProfile(src, profileId, file)
}

func (c *Controller) GeneratePreSignetResourceSignature(resource, action string, id int64, domainId int64) (string, engine.AppError) {
	return c.app.GeneratePreSignedResourceSignature(resource, action, id, domainId)
}
func (c *Controller) GeneratePreSignedResourceSignatureBulk(id int64, domainId int64, resource string, action string, source string, queryParams map[string]string) (string, engine.AppError) {
	return c.app.GeneratePreSignedResourceSignatureBulk(id, domainId, resource, action, source, queryParams)
}

func (c *Controller) InsecureGetFileWithProfile(domainId, id int64) (*model.File, utils.FileBackend, engine.AppError) {
	return c.app.GetFileWithProfile(domainId, id)
}
