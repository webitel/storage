package grpc_api

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"

	"github.com/webitel/wlog"

	engine "github.com/webitel/engine/model"
	"github.com/webitel/protos/storage"
	"github.com/webitel/storage/controller"
	"github.com/webitel/storage/model"
)

type file struct {
	ctrl       *controller.Controller
	curl       *http.Client
	publicHost string
	storage.UnsafeFileServiceServer
}

func NewFileApi(proxy *string, ph string, api *controller.Controller) *file {
	c := &file{
		ctrl:       api,
		publicHost: ph,
	}
	if proxy != nil {
		proxyUrl, err := url.Parse(*proxy)
		if err != nil {
			panic(err.Error())
		}

		c.curl = &http.Client{Transport: &http.Transport{Proxy: http.ProxyURL(proxyUrl)}}
	} else {
		c.curl = http.DefaultClient
	}

	return c
}

func (api *file) UploadFile(in storage.FileService_UploadFileServer) error {
	var chunk *storage.UploadFileRequest_Chunk

	res, gErr := in.Recv()
	if gErr != nil {
		wlog.Error(gErr.Error())
		return gErr
	}

	defer func() {
		if gErr != nil && gErr != io.EOF {
			wlog.Error(gErr.Error())
		}
	}()

	metadata, ok := res.Data.(*storage.UploadFileRequest_Metadata_)
	if !ok {
		gErr = errors.New("bad metadata")
		return gErr
	}

	var fileRequest model.JobUploadFile
	fileRequest.DomainId = metadata.Metadata.DomainId
	fileRequest.Name = metadata.Metadata.Name

	fileRequest.MimeType = metadata.Metadata.MimeType
	fileRequest.Uuid = metadata.Metadata.Uuid

	pipeReader, pipeWriter := io.Pipe()

	go func(writer *io.PipeWriter) {
		for {
			res, gErr = in.Recv()
			if gErr != nil {
				break
			}

			if chunk, ok = res.Data.(*storage.UploadFileRequest_Chunk); !ok {
				gErr = errors.New("streaming data error: bad UploadFileRequest_Chunk")
				break
			}

			if len(chunk.Chunk) == 0 {
				break
			}

			writer.Write(chunk.Chunk)
		}

		writer.Close()

	}(pipeWriter)

	var err engine.AppError
	var publicUrl string
	if err = api.ctrl.UploadFileStream(pipeReader, &fileRequest); err != nil {
		return err
	}

	if publicUrl, err = api.ctrl.GeneratePreSignetResourceSignature(model.AnyFileRouteName, "download", fileRequest.Id, fileRequest.DomainId); err != nil {
		return err
	}

	return in.SendAndClose(&storage.UploadFileResponse{
		FileId:  fileRequest.Id,
		Size:    fileRequest.Size,
		Code:    storage.UploadStatusCode_Ok,
		FileUrl: publicUrl,
		Server:  api.publicHost,
	})
}

func (api *file) DownloadFile(in *storage.DownloadFileRequest, stream storage.FileService_DownloadFileServer) error {
	var sFile io.ReadCloser
	var err error
	f, backend, appErr := api.ctrl.InsecureGetFileWithProfile(in.DomainId, in.Id)
	if appErr != nil {
		return appErr
	}

	sFile, appErr = backend.Reader(f, 0)
	if appErr != nil {
		return appErr
	}

	defer sFile.Close()

	if in.Metadata {
		err = stream.Send(&storage.StreamFile{
			Data: &storage.StreamFile_Metadata_{
				Metadata: &storage.StreamFile_Metadata{
					Id:       f.Id,
					Name:     f.Name,
					MimeType: f.MimeType,
					Uuid:     f.Uuid,
					Size:     f.Size,
				},
			},
		})

		if err != nil {
			return err
		}
	}

	buf := make([]byte, 4*1024)
	var n int
	for {
		n, err = sFile.Read(buf)
		buf = buf[:n]
		if err != nil {
			break
		}
		err = stream.Send(&storage.StreamFile{
			Data: &storage.StreamFile_Chunk{
				Chunk: buf,
			},
		})
		if err != nil {
			break
		}
	}

	return nil
}

func (api *file) UploadFileUrl(ctx context.Context, in *storage.UploadFileUrlRequest) (*storage.UploadFileUrlResponse, error) {
	var err engine.AppError
	var publicUrl string

	if in.Url == "" || in.DomainId == 0 || in.Name == "" {
		return nil, errors.New("bad request")
	}

	res, httpErr := api.curl.Get(in.GetUrl())
	if httpErr != nil {
		return nil, httpErr
	}

	defer res.Body.Close()

	var fileRequest model.JobUploadFile
	fileRequest.DomainId = in.GetDomainId()
	fileRequest.Name = model.NewId() + "_" + in.GetName()
	fileRequest.ViewName = model.NewString(in.GetName())
	fileRequest.MimeType = res.Header.Get("Content-Type")
	fileRequest.Uuid = in.GetUuid()
	fileRequest.Size = res.ContentLength
	if fileRequest.Uuid == "" {
		fileRequest.Uuid = model.NewId() // bad request ?
	}

	if fileRequest.MimeType == "application/octet-stream" && in.Mime != "" {
		fileRequest.MimeType = in.Mime
	}

	if err = api.ctrl.UploadFileStream(res.Body, &fileRequest); err != nil {
		return nil, err
	}

	if publicUrl, err = api.ctrl.GeneratePreSignetResourceSignature(model.AnyFileRouteName, "download", fileRequest.Id, fileRequest.DomainId); err != nil {
		return nil, err
	}

	return &storage.UploadFileUrlResponse{
		Id:   fileRequest.Id,
		Code: storage.UploadStatusCode_Ok,
		Url:  publicUrl,
		Size: fileRequest.Size,
		Mime: fileRequest.MimeType,
		// TODO ADD Server
	}, nil
}

func (api *file) DeleteFiles(ctx context.Context, in *storage.DeleteFilesRequest) (*storage.DeleteFilesResponse, error) {
	session, err := api.ctrl.GetSessionFromCtx(ctx)
	if err != nil {
		return nil, err
	}

	err = api.ctrl.DeleteFiles(session, in.Id)
	if err != nil {
		return nil, err
	}

	return &storage.DeleteFilesResponse{}, nil
}
