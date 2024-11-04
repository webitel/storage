package apis

import (
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"

	"github.com/webitel/storage/model"
)

func (api *API) InitMediaFile() {
	api.PublicRoutes.MediaFiles.Handle("", api.ApiSessionRequired(saveMediaFile)).Methods("POST")
	api.PublicRoutes.MediaFiles.Handle("/{id}/stream", api.ApiSessionRequired(streamMediaFile)).Methods("GET")
	api.PublicRoutes.MediaFiles.Handle("/{id}/download", api.ApiSessionRequired(downloadMediaFile)).Methods("GET")
}

func streamMediaFile(c *Context, w http.ResponseWriter, r *http.Request) {
	c.RequireId()

	if c.Err != nil {
		return
	}

	var file *model.MediaFile
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

	if file, c.Err = c.Ctrl.GetMediaFile(&c.Session, int64(domainId), id); c.Err != nil {
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

	if reader, c.Err = c.App.MediaFileStore.Reader(file, offset); c.Err != nil {
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

func downloadMediaFile(c *Context, w http.ResponseWriter, r *http.Request) {
	c.RequireId()

	if c.Err != nil {
		return
	}

	var file *model.MediaFile
	var id, domainId int
	var err error
	var reader io.ReadCloser

	if id, err = strconv.Atoi(c.Params.Id); err != nil {
		c.SetInvalidUrlParam("id")
		return
	}

	domainId, _ = strconv.Atoi(c.Params.Domain)

	if file, c.Err = c.Ctrl.GetMediaFile(&c.Session, int64(domainId), id); c.Err != nil {
		return
	}

	sendSize := file.Size
	code := http.StatusOK

	if reader, c.Err = c.App.MediaFileStore.Reader(file, 0); c.Err != nil {
		return
	}

	defer reader.Close()

	reader, c.Err = c.App.FilePolicyForDownload(file.DomainId, &file.BaseFile, reader)
	if c.Err != nil {
		return
	}

	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment;  filename=\"%s\"", model.EncodeURIComponent(file.Name)))
	w.Header().Set("Content-Type", file.MimeType)
	w.Header().Set("Content-Length", strconv.FormatInt(sendSize, 10))

	w.WriteHeader(code)
	io.Copy(w, reader)
}

func saveMediaFile(c *Context, w http.ResponseWriter, r *http.Request) {

	defer r.Body.Close()

	files := make([]*model.MediaFile, 0)

	mediaType, params, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil {
		panic(err)
	}

	if strings.HasPrefix(mediaType, "multipart/form-data") {
		writer := multipart.NewReader(r.Body, params["boundary"])

		for {
			part, err := writer.NextPart()
			if err == io.EOF {
				break
			}

			if err != nil {
				panic(err)
			}

			file := &model.MediaFile{}
			file.Properties = model.StringInterface{}
			file.Name = part.FileName()
			file.MimeType = part.Header.Get("Content-Type")

			if file, c.Err = c.Ctrl.CreateMediaFile(&c.Session, part, file); c.Err != nil {
				break
			}
			files = append(files, file)
			part.Close()
		}
	} else {
		file := &model.MediaFile{}
		file.Properties = model.StringInterface{}
		file.Name = r.URL.Query().Get("name")
		file.MimeType = r.Header.Get("Content-Type")

		if file, c.Err = c.Ctrl.CreateMediaFile(&c.Session, r.Body, file); c.Err == nil {
			files = append(files, file)
		}
	}

	if c.Err != nil {
		return
	}

	response := &ListResponse{
		Items: files,
	}

	w.Write([]byte(response.ToJson()))
}
