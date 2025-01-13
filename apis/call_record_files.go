package apis

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/webitel/engine/auth_manager"
	engine "github.com/webitel/engine/model"
	"github.com/webitel/storage/model"
	"github.com/webitel/storage/utils"
	"github.com/webitel/storage/web"
)

var errNoPermissionRecordFile = engine.NewForbiddenError("call.recordings.access.forbidden", "Not allow")

func (api *API) InitCallRecordingsFiles() {
	api.PublicRoutes.CallRecordingsFiles.Handle("/{id}/stream", api.ApiSessionRequired(streamRecordFile)).Methods("GET")
	api.PublicRoutes.CallRecordingsFiles.Handle("/{id}/download", api.ApiSessionRequired(downloadRecordFile)).Methods("GET")
}

func streamRecordFile(c *Context, w http.ResponseWriter, r *http.Request) {
	isAccessible, err := checkCallRecordPermission(c, r)
	if err != nil {
		c.Err = err
		return
	}
	if !isAccessible {
		c.Err = errNoPermissionRecordFile
		return
	}
	streamFile(c, w, r)

}

func downloadRecordFile(c *Context, w http.ResponseWriter, r *http.Request) {
	isAccessible, err := checkCallRecordPermission(c, r)
	if err != nil {
		c.Err = err
		return
	}
	if !isAccessible {
		c.Err = errNoPermissionRecordFile
		return
	}
	downloadFile(c, w, r)

}

func streamFile(c *Context, w http.ResponseWriter, r *http.Request) {

	c.RequireId()

	if c.Err != nil {
		return
	}
	query := r.URL.Query()
	var file *model.File
	var backend utils.FileBackend
	var id, domainId int
	var err error
	var ranges []HttpRange
	var offset int64 = 0
	var reader io.ReadCloser

	if id, err = strconv.Atoi(c.Params.Id); err != nil {
		c.SetInvalidUrlParam("id")
		return
	}

	domainId, _ = strconv.Atoi(c.Params.Domain)

	if source := sourceFromRequest(r); source == "media" {
		streamMediaFile(c, w, r)
		return
	}

	if file, backend, c.Err = c.Ctrl.GetFileWithProfile(&c.Session, int64(domainId), int64(id)); c.Err != nil {
		return
	}

	// TODO DEV-4661
	if file.Channel != nil && *file.Channel == model.UploadFileChannelCall {
		if !allowTimeLimited(r.Context(), c, file.CreatedAt) {
			c.Err = errNoPermissionRecordFile
			return
		}
	}

	if file.Thumbnail != nil && query.Get("fetch_thumbnail") == "true" {
		file.BaseFile = file.Thumbnail.BaseFile
	}

	if ranges, c.Err = parseRange(r.Header.Get("Range"), file.Size); c.Err != nil {
		return
	}

	sendSize := file.Size
	code := http.StatusOK

	switch {
	case len(ranges) == 1:
		code = http.StatusPartialContent
		offset = ranges[0].Start
		sendSize = ranges[0].Length
		w.Header().Set("Content-Range", ranges[0].ContentRange(file.Size))
	default:

	}

	if reader, c.Err = backend.Reader(file, offset); c.Err != nil {
		return
	}

	defer reader.Close()

	reader, c.Err = c.App.FilePolicyForDownload(file.DomainId, &file.BaseFile, reader)
	if c.Err != nil {
		return
	}

	if w.Header().Get("Content-Encoding") == "" {
		w.Header().Set("Content-Length", strconv.FormatInt(sendSize, 10))
	}

	w.Header().Set("Accept-Ranges", "bytes")
	w.Header().Set("Content-Type", file.MimeType)

	w.WriteHeader(code)
	io.CopyN(w, reader, sendSize)
}

func downloadFile(c *Context, w http.ResponseWriter, r *http.Request) {
	c.RequireId()

	if c.Err != nil {
		return
	}

	query := r.URL.Query()
	var file *model.File
	var backend utils.FileBackend
	var id, domainId int
	var err error
	var reader io.ReadCloser

	if id, err = strconv.Atoi(c.Params.Id); err != nil {
		c.SetInvalidUrlParam("id")
		return
	}

	domainId, _ = strconv.Atoi(c.Params.Domain)

	if source := sourceFromRequest(r); source == "media" {
		downloadMediaFile(c, w, r)
		return
	}

	if file, backend, c.Err = c.Ctrl.GetFileWithProfile(&c.Session, int64(domainId), int64(id)); c.Err != nil {
		return
	}

	// TODO DEV-4661
	if file.Channel != nil && *file.Channel == model.UploadFileChannelCall {
		if !allowTimeLimited(r.Context(), c, file.CreatedAt) {
			c.Err = errNoPermissionRecordFile
			return
		}
	}

	if file.Thumbnail != nil && query.Get("fetch_thumbnail") == "true" {
		file.BaseFile = file.Thumbnail.BaseFile
	}

	sendSize := file.Size
	code := http.StatusOK

	if reader, c.Err = backend.Reader(file, 0); c.Err != nil {
		return
	}

	defer reader.Close()

	reader, c.Err = c.App.FilePolicyForDownload(file.DomainId, &file.BaseFile, reader)
	if c.Err != nil {
		return
	}

	var name = file.GetViewName()
	if c.Params.Name != "" {
		name = c.Params.Name
	}

	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment;  filename=\"%s\"", model.EncodeURIComponent(name)))
	w.Header().Set("Content-Type", file.MimeType)
	w.Header().Set("Content-Length", strconv.FormatInt(sendSize, 10))

	w.WriteHeader(code)
	io.Copy(w, reader)
}

func allowTimeLimited(ctx context.Context, c *Context, createdAt int64) bool {
	if c.Session.HasAction(auth_manager.PermissionRecordFile) {
		return true
	}

	if c.Session.HasAction(auth_manager.PermissionTimeLimitedRecordFile) {
		if showFilePeriodDay, _ := c.App.GetCachedSystemSetting(ctx, c.Session.Domain(0), engine.SysNamePeriodToPlaybackRecord); showFilePeriodDay.Int() != nil {
			t := time.Now().Add(-(time.Hour * 24 * time.Duration(*showFilePeriodDay.Int())))
			return time.Unix(0, createdAt*int64(time.Millisecond)).After(t)
		}
	}

	return false
}

func checkCallRecordPermission(c *Context, r *http.Request) (bool, engine.AppError) {
	if !c.Session.HasAction(auth_manager.PermissionRecordFile) {
		session := c.Session
		permission := session.GetPermission(model.PERMISSION_SCOPE_RECORD_FILE)
		if !permission.CanRead() {
			return false, errNoPermissionRecordFile
		}
		if session.UseRBAC(auth_manager.PERMISSION_ACCESS_READ, permission) {
			id, err := strconv.Atoi(c.Params.Id)
			if err != nil {
				return false, web.NewInvalidUrlParamError("id")
			}
			isAccessible, appErr := c.App.CheckCallRecordPermissions(r.Context(), id, session.UserId, session.DomainId, session.RoleIds)
			if appErr != nil {
				return false, appErr
			}
			return isAccessible, nil

		}
	}

	return true, nil

}

func sourceFromRequest(r *http.Request) string {
	q := r.URL.Query()
	return q.Get("source")
}
