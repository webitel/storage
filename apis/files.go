package apis

import (
	"encoding/json"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/webitel/storage/utils"

	"github.com/webitel/storage/model"
)

type fileResponse struct {
	Id        int64  `json:"id"`
	Name      string `json:"name"`
	Size      int64  `json:"size"`
	MimeType  string `json:"mime"`
	SharedUrl string `json:"shared"`
}

func (api *API) InitFile() {
	api.PublicRoutes.Files.Handle("/{id}/stream", api.ApiSessionRequired(streamFile)).Methods("GET")
	api.PublicRoutes.Files.Handle("/{id}/download", api.ApiSessionRequired(downloadFile)).Methods("GET")
	api.PublicRoutes.Files.Handle("/{id}/upload", api.ApiSessionRequired(uploadAnyFile)).Methods("POST")
	api.PublicRoutes.Files.Handle("/{id}/transcript", api.ApiSessionRequired(transcriptFile)).Methods("GET")
}

func transcriptFile(c *Context, w http.ResponseWriter, r *http.Request) {
	var id int
	var err error

	c.RequireId()

	if c.Err != nil {
		return
	}

	if id, err = strconv.Atoi(c.Params.Id); err != nil {
		c.SetInvalidUrlParam("id")
		return
	}

	//TODO
	var tr *model.FileTranscript
	if tr, c.Err = c.App.TranscriptFile(int64(id), model.TranscriptOptions{}); c.Err != nil {

		return
	}

	data, _ := json.Marshal(tr)
	w.Write(data)
}

func uploadAnyFile(c *Context, w http.ResponseWriter, r *http.Request) {
	c.RequireId()

	if c.Err != nil {
		return
	}

	defer r.Body.Close()

	files := make([]*fileResponse, 0)

	mediaType, params, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil {
		panic(err)
	}

	generateThumbnail := r.URL.Query().Get("thumbnail") == "true"
	channel := r.URL.Query().Get("channel")
	if channel == "" {
		channel = "unknown"
	}

	if strings.HasPrefix(mediaType, "multipart/form-data") {
		writer := multipart.NewReader(r.Body, params["boundary"])
		var part *multipart.Part

		for {
			part, err = writer.NextPart()
			if err == io.EOF {
				break
			}

			if err != nil {
				break //fixme
			}

			file := &model.JobUploadFile{}
			name := part.FileName()
			file.ViewName = &name
			file.Name = model.NewId() + "_" + name
			file.MimeType = part.Header.Get("Content-Type")
			file.DomainId = c.Session.DomainId
			file.Uuid = c.Params.Id
			file.GenerateThumbnail = generateThumbnail
			file.Channel = &channel
			file.CustomProperties = CustomPropertiesFromQuery(r.URL.Query()) // TODO
			file.UploadedBy = &model.Lookup{Id: int(c.Session.UserId)}

			var reader io.ReadCloser
			reader, c.Err = c.App.FilePolicyForUpload(c.Session.DomainId, &file.BaseFile, part)
			if c.Err != nil {
				return
			}

			// TODO PERMISSION
			if c.Err = c.App.SyncUpload(reader, file); c.Err != nil {
				if c.Err.GetId() == utils.ErrMaxLimitId {
					c.Err.SetDetailedError(utils.BytesSize(float64(c.App.MaxUploadFileSize())))
				}
				reader.Close()
				return
			}

			reader.Close()
			sig, _ := c.App.GeneratePreSignedResourceSignature(model.AnyFileRouteName, "download", file.Id, file.DomainId)

			files = append(files, &fileResponse{
				Id:        file.Id,
				Name:      file.GetViewName(),
				Size:      file.Size,
				MimeType:  file.MimeType,
				SharedUrl: sig,
			})
			part.Close()
		}
	} else {
		file := &model.JobUploadFile{}
		name := r.URL.Query().Get("name")
		file.ViewName = &name
		file.Name = model.NewId() + "_" + name
		file.MimeType = r.Header.Get("Content-Type")
		file.DomainId = c.Session.DomainId
		file.Uuid = c.Params.Id
		file.GenerateThumbnail = generateThumbnail
		file.Channel = &channel
		file.CustomProperties = CustomPropertiesFromQuery(r.URL.Query())
		file.UploadedBy = &model.Lookup{Id: int(c.Session.UserId)}

		var reader io.ReadCloser
		reader, c.Err = c.App.FilePolicyForUpload(c.Session.DomainId, &file.BaseFile, r.Body)
		if c.Err != nil {
			return
		}
		defer reader.Close()

		// TODO PERMISSION
		if c.Err = c.App.SyncUpload(reader, file); c.Err != nil {
			if c.Err.GetId() == utils.ErrMaxLimitId {
				c.Err.SetDetailedError(utils.BytesSize(float64(c.App.MaxUploadFileSize())))
			}
			return
		}

		sig, _ := c.App.GeneratePreSignedResourceSignature(model.AnyFileRouteName, "download", file.Id, file.DomainId)
		files = append(files, &fileResponse{
			Id:        file.Id,
			Name:      file.GetViewName(),
			Size:      file.Size,
			MimeType:  file.MimeType,
			SharedUrl: sig,
		})
	}

	if c.Err != nil {
		return
	}

	data, _ := json.Marshal(files)
	w.Write(data)

	// todo app generate public download
}

func CustomPropertiesFromQuery(q url.Values) *model.CustomFileProperties {
	var c model.CustomFileProperties
	set := func(v *int, s string) {
		if s != "" {
			*v, _ = strconv.Atoi(s)
		}
	}

	set(&c.StartTime, q.Get("start_time"))
	set(&c.EndTime, q.Get("end_time"))
	set(&c.Width, q.Get("width"))
	set(&c.Height, q.Get("height"))

	if c.Width != 0 || c.Height != 0 || c.StartTime != 0 || c.EndTime != 0 {
		return &c
	}

	return nil
}
