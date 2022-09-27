package apis

import (
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/webitel/storage/model"
	"github.com/webitel/storage/utils"
)

func (api *API) InitCallRecordingsFiles() {
	api.PublicRoutes.CallRecordingsFiles.Handle("/{id}/stream", api.ApiSessionRequired(streamRecordFile)).Methods("GET")
	api.PublicRoutes.CallRecordingsFiles.Handle("/{id}/download", api.ApiSessionRequired(downloadRecordFile)).Methods("GET")
}

func streamRecordFile(c *Context, w http.ResponseWriter, r *http.Request) {
	c.RequireId()

	if c.Err != nil {
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

	if file, backend, c.Err = c.Ctrl.GetFileWithProfile(&c.Session, int64(domainId), int64(id)); c.Err != nil {
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

func downloadRecordFile(c *Context, w http.ResponseWriter, r *http.Request) {
	c.RequireId()

	if c.Err != nil {
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

	if file, backend, c.Err = c.Ctrl.GetFileWithProfile(&c.Session, int64(domainId), int64(id)); c.Err != nil {
		return
	}

	sendSize := file.Size
	code := http.StatusOK

	if reader, c.Err = backend.Reader(file, 0); c.Err != nil {
		return
	}

	defer reader.Close()

	var name = file.Name
	if c.Params.Name != "" {
		name = c.Params.Name
	}

	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment;  filename=\"%s\"", model.EncodeURIComponent(name)))
	w.Header().Set("Content-Type", file.MimeType)
	w.Header().Set("Content-Length", strconv.FormatInt(sendSize, 10))

	w.WriteHeader(code)
	io.Copy(w, reader)
}
