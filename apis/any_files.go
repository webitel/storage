package apis

import (
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/webitel/storage/model"
	"github.com/webitel/storage/utils"
)

func (api *API) InitAnyFile() {
	api.PublicRoutes.AnyFiles.Handle("/{id}/stream", api.ApiHandler(streamAnyFile)).Methods("GET")
	api.PublicRoutes.AnyFiles.Handle("/{id}/download", api.ApiHandler(downloadAnyFile)).Methods("GET")
	api.PublicRoutes.AnyFiles.Handle("/stream", api.ApiHandler(streamAnyFileByQuery)).Methods("GET")
	api.PublicRoutes.AnyFiles.Handle("/download", api.ApiHandler(downloadAnyFileByQuery)).Methods("GET")
}

func streamAnyFile(c *Context, w http.ResponseWriter, r *http.Request) {
	c.RequireId()
	c.RequireDomain()
	c.RequireExpire()
	c.RequireSignature()

	if c.Err != nil {
		return
	}

	if c.Params.Expires < model.GetMillis() {
		c.SetSessionExpire()
		return
	}

	key := fmt.Sprintf("%s/%s/stream?domain_id=%s&expires=%d", model.AnyFileRouteName, c.Params.Id, c.Params.Domain, c.Params.Expires)

	if !c.App.ValidateSignature(key, c.Params.Signature) {
		c.SetSessionErrSignature()
		return
	}

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

	if file, backend, c.Err = c.App.GetFileWithProfile(int64(domainId), int64(id)); c.Err != nil {
		return
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

	if w.Header().Get("Content-Encoding") == "" {
		w.Header().Set("Content-Length", strconv.FormatInt(sendSize, 10))
	}

	w.Header().Set("Accept-Ranges", "bytes")
	w.Header().Set("Content-Type", file.MimeType)

	w.WriteHeader(code)
	io.CopyN(w, reader, sendSize)
}

func downloadAnyFile(c *Context, w http.ResponseWriter, _ *http.Request) {
	c.RequireId()
	c.RequireDomain()
	c.RequireExpire()
	c.RequireSignature()

	if c.Err != nil {
		return
	}

	if c.Params.Expires < model.GetMillis() {
		c.SetSessionExpire()
		return
	}

	key := fmt.Sprintf("%s/%s/download?domain_id=%s&expires=%d", model.AnyFileRouteName, c.Params.Id, c.Params.Domain, c.Params.Expires)

	if !c.App.ValidateSignature(key, c.Params.Signature) {
		c.SetSessionErrSignature()
		return
	}

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

	if file, backend, c.Err = c.App.GetFileWithProfile(int64(domainId), int64(id)); c.Err != nil {
		return
	}

	sendSize := file.Size
	code := http.StatusOK

	if reader, c.Err = backend.Reader(file, 0); c.Err != nil {
		return
	}

	defer reader.Close()

	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment;  filename=\"%s\"", model.EncodeURIComponent(file.Name)))
	w.Header().Set("Content-Type", file.MimeType)
	w.Header().Set("Content-Length", strconv.FormatInt(sendSize, 10))

	w.WriteHeader(code)
	io.Copy(w, reader)
}

func streamAnyFileByQuery(c *Context, w http.ResponseWriter, r *http.Request) {
	c.RequireDomain()
	c.RequireExpire()
	c.RequireSignature()

	if c.Err != nil {
		return
	}

	if c.Params.Expires < model.GetMillis() {
		c.SetSessionExpire()
		return
	}

	q := r.URL.Query()
	uuid := q.Get("uuid")
	if uuid == "" {
		c.SetInvalidUrlParam("uuid")
		return
	}

	key := fmt.Sprintf("%s/stream?domain_id=%s&uuid=%s&expires=%d", model.AnyFileRouteName, c.Params.Domain, uuid, c.Params.Expires)

	if !c.App.ValidateSignature(key, c.Params.Signature) {
		c.SetSessionErrSignature()
		return
	}

	var file *model.File
	var backend utils.FileBackend
	var domainId int
	var ranges []HttpRange
	var offset int64 = 0
	var reader io.ReadCloser

	domainId, _ = strconv.Atoi(c.Params.Domain)

	if file, backend, c.Err = c.App.GetFileByUuidWithProfile(int64(domainId), uuid); c.Err != nil {
		return
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

	if w.Header().Get("Content-Encoding") == "" {
		w.Header().Set("Content-Length", strconv.FormatInt(sendSize, 10))
	}

	w.Header().Set("Accept-Ranges", "bytes")
	w.Header().Set("Content-Type", file.MimeType)

	w.WriteHeader(code)
	io.CopyN(w, reader, sendSize)
}

func downloadAnyFileByQuery(c *Context, w http.ResponseWriter, r *http.Request) {
	c.RequireDomain()
	c.RequireExpire()
	c.RequireSignature()

	if c.Err != nil {
		return
	}

	if c.Params.Expires < model.GetMillis() {
		c.SetSessionExpire()
		return
	}

	q := r.URL.Query()
	uuid := q.Get("uuid")
	if uuid == "" {
		c.SetInvalidUrlParam("uuid")
		return
	}

	key := fmt.Sprintf("%s/download?domain_id=%s&uuid=%s&expires=%d", model.AnyFileRouteName, c.Params.Domain, uuid, c.Params.Expires)

	if !c.App.ValidateSignature(key, c.Params.Signature) {
		c.SetSessionErrSignature()
		return
	}

	var file utils.File
	var backend utils.FileBackend
	var domainId int
	var reader io.ReadCloser
	source := q.Get("source")

	domainId, _ = strconv.Atoi(c.Params.Domain)

	switch source {
	case "media":
		backend = c.App.MediaFileStore
		mediaId, _ := strconv.Atoi(uuid)
		file, c.Err = c.App.GetMediaFile(int64(domainId), mediaId)
	case "file":
		fileId, _ := strconv.Atoi(uuid)
		file, backend, c.Err = c.App.GetFileWithProfile(int64(domainId), int64(fileId))
	default:
		file, backend, c.Err = c.App.GetFileByUuidWithProfile(int64(domainId), uuid)
	}

	if c.Err != nil {
		return
	}

	if file == nil {
		// TODO not found
		return
	}

	sendSize := file.GetSize()
	code := http.StatusOK

	if reader, c.Err = backend.Reader(file, 0); c.Err != nil {
		return
	}

	defer reader.Close()

	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment;  filename=\"%s\"", model.EncodeURIComponent(file.GetStoreName())))
	w.Header().Set("Content-Type", file.GetMimeType())
	w.Header().Set("Content-Length", strconv.FormatInt(sendSize, 10))

	w.WriteHeader(code)
	io.Copy(w, reader)
}
