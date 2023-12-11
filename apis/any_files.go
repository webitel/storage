package apis

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	// so we use this beta one

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

	// region VALIDATION
	validationString := createValidationKey(*r.URL)
	// dynamic parameters validation
	if !c.App.ValidateSignature(model.AnyFileRouteName+validationString, c.Params.Signature) {
		c.SetSessionErrSignature()
		return
	}
	// endregion

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

func downloadAnyFile(c *Context, w http.ResponseWriter, r *http.Request) {
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
	// region VALIDATION
	validationString := createValidationKey(*r.URL)
	// dynamic parameters validation
	if !c.App.ValidateSignature(model.AnyFileRouteName+validationString, c.Params.Signature) {
		c.SetSessionErrSignature()
		return
	}
	// endregion

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

	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment;  filename=\"%s\"", model.EncodeURIComponent(file.GetViewName())))
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

	//key := fmt.Sprintf("%s/stream?domain_id=%s&uuid=%s&expires=%d", model.AnyFileRouteName, c.Params.Domain, uuid, c.Params.Expires)

	// region VALIDATION
	validationString := createValidationKey(*r.URL)
	// dynamic parameters validation
	if !c.App.ValidateSignature(model.AnyFileRouteName+validationString, c.Params.Signature) {
		c.SetSessionErrSignature()
		return
	}
	// endregion

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

func createValidationKey(key url.URL) string {
	existingParams := key.Query()
	existingParams.Del("signature")
	key.RawQuery = existingParams.Encode()
	// dynamic parameters validation
	before, after, found := strings.Cut(key.String(), model.AnyFileRouteName)
	if !found {
		return before
	}
	return after
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

	//key := fmt.Sprintf("%s/download?domain_id=%s&uuid=%s&expires=%d", model.AnyFileRouteName, c.Params.Domain, uuid, c.Params.Expires)

	var file utils.File
	var backend utils.FileBackend
	var domainId int
	var reader io.ReadCloser
	source := q.Get("source")

	// region VALIDATION
	validationString := createValidationKey(*r.URL)
	// dynamic parameters validation
	if !c.App.ValidateSignature(model.AnyFileRouteName+validationString, c.Params.Signature) {
		c.SetSessionErrSignature()
		return
	}
	// endregion
	domainId, _ = strconv.Atoi(c.Params.Domain)

	switch source {
	case "media":
		if uuid == "" {
			c.SetInvalidUrlParam("uuid")
			return
		}
		backend = c.App.MediaFileStore
		mediaId, _ := strconv.Atoi(uuid)
		file, c.Err = c.App.GetMediaFile(int64(domainId), mediaId)
	case "file":
		if uuid == "" {
			c.SetInvalidUrlParam("uuid")
			return
		}
		fileId, _ := strconv.Atoi(uuid)
		file, backend, c.Err = c.App.GetFileWithProfile(int64(domainId), int64(fileId))
	case "tts":
		tts(c, w, r)
		return
	case "barcode":
		var width, height int
		text := q.Get("text")
		altText := q.Get("alttext")
		wT := q.Get("w")
		hT := q.Get("h")
		if text == "" {
			return
		}
		var buf *bytes.Buffer

		if wT != "" {
			width, _ = strconv.Atoi(wT)
		}
		if hT != "" {
			height, _ = strconv.Atoi(hT)
		}

		if width == 0 {
			width = 350
		}

		if height == 0 {
			height = 150
		}

		if c.Err, buf = c.App.Barcode(text, altText, width, height); c.Err != nil {
			return
		}

		w.Header().Set("Content-Type", "image/png")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", model.EncodeURIComponent("code.png")))
		io.Copy(w, buf)
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(buf.Bytes())))
		//w.WriteHeader(200)

		return

	default:
		if uuid == "" {
			c.SetInvalidUrlParam("uuid")
			return
		}
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
