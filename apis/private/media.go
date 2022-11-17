package private

import (
	"io"
	"net/http"
	"strconv"

	"github.com/webitel/storage/apis/helper"
	"github.com/webitel/storage/model"
)

func (api *API) InitMedia() {
	api.Routes.Media.Handle("/{id}/stream", api.ApiHandler(streamMedia)).Methods("GET")
}

//  /sys/media/:id/stream?domain_id=
func streamMedia(c *Context, w http.ResponseWriter, r *http.Request) {
	c.RequireId()

	if c.Err != nil {
		return
	}

	var file *model.MediaFile
	var id, domainId int
	var err error
	var ranges []helper.HttpRange
	var offset int64 = 0
	var reader io.ReadCloser

	if id, err = strconv.Atoi(c.Params.Id); err != nil {
		c.SetInvalidUrlParam("id")
		return
	}

	domainId, _ = strconv.Atoi(c.Params.Domain)

	if file, c.Err = c.App.GetMediaFile(int64(domainId), id); c.Err != nil {
		return
	}

	if ranges, c.Err = helper.ParseRange(r.Header.Get("Range"), file.Size); c.Err != nil {
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

	if w.Header().Get("Content-Encoding") == "" {
		w.Header().Set("Content-Length", strconv.FormatInt(sendSize, 10))
	}

	w.Header().Set("Accept-Ranges", "bytes")
	w.Header().Set("Content-Type", file.MimeType)

	w.WriteHeader(code)
	io.CopyN(w, reader, sendSize)
}
