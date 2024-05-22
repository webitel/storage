package grpc_api

import (
	"context"
	"errors"
	"fmt"
	"github.com/webitel/storage/app"
	"io"
	"net/http"
	"net/url"

	"github.com/webitel/wlog"

	gogrpc "buf.build/gen/go/webitel/storage/grpc/go/_gogrpc"
	storage "buf.build/gen/go/webitel/storage/protocolbuffers/go"
	engine "github.com/webitel/engine/model"

	"github.com/webitel/storage/controller"
	"github.com/webitel/storage/model"
)

type file struct {
	ctrl       *controller.Controller
	curl       *http.Client
	publicHost string
	gogrpc.UnsafeFileServiceServer
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

func (api *file) UploadFile(in gogrpc.FileService_UploadFileServer) error {
	var chunk *storage.UploadFileRequest_Chunk

	res, gErr := in.Recv()
	if gErr != nil {
		wlog.Error(gErr.Error())
		return gErr
	}

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
	fileRequest.ViewName = &metadata.Metadata.Name

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

		if gErr != nil && gErr != io.EOF {
			wlog.Error(gErr.Error())
			writer.CloseWithError(gErr)
		} else {
			writer.Close()
		}

	}(pipeWriter)

	var err engine.AppError
	var publicUrl string

	if metadata.Metadata.ProfileId != 0 {
		fileRequest.Name = fmt.Sprintf("%s_%s", model.NewId()[0:7], fileRequest.Name)
		err = api.ctrl.UploadFileStreamToProfile(pipeReader, int(metadata.Metadata.ProfileId), &fileRequest)
	} else {
		err = api.ctrl.UploadFileStream(pipeReader, &fileRequest)
	}

	if err != nil {
		return err
	}

	if publicUrl, err = api.ctrl.GeneratePreSignetResourceSignature(model.AnyFileRouteName, "download", fileRequest.Id, fileRequest.DomainId); err != nil {
		return err
	}

	result := &storage.UploadFileResponse{
		FileId:  fileRequest.Id,
		Size:    fileRequest.Size,
		Code:    storage.UploadStatusCode_Ok,
		FileUrl: publicUrl,
		Server:  api.publicHost,
	}

	if fileRequest.SHA256Sum != nil {
		result.Sha256Sum = *fileRequest.SHA256Sum
	}

	return in.SendAndClose(result)
}

func (api *file) GenerateFileLink(ctx context.Context, in *storage.GenerateFileLinkRequest) (*storage.GenerateFileLinkResponse, error) {
	uri, err := api.ctrl.GeneratePreSignedResourceSignatureBulk(in.GetFileId(), in.GetDomainId(), model.AnyFileRouteName, in.GetAction(), in.GetSource(), in.GetQuery())
	if err != nil {
		return nil, err
	}
	return &storage.GenerateFileLinkResponse{Url: uri}, nil
}

func (api *file) DownloadFile(in *storage.DownloadFileRequest, stream gogrpc.FileService_DownloadFileServer) error {
	var sFile io.ReadCloser
	var err error
	var buf []byte
	var bufferSize int64 = 4 * 1024

	f, backend, appErr := api.ctrl.InsecureGetFileWithProfile(in.DomainId, in.Id)
	if appErr != nil {
		return appErr
	}

	sFile, appErr = backend.Reader(f, in.Offset)
	if appErr != nil {
		return appErr
	}

	defer sFile.Close()

	if in.Metadata {
		d := &storage.StreamFile_Metadata_{
			Metadata: &storage.StreamFile_Metadata{
				Id:       f.Id,
				Name:     f.Name,
				MimeType: f.MimeType,
				Uuid:     f.Uuid,
				Size:     f.Size,
			},
		}
		if f.SHA256Sum != nil {
			d.Metadata.Sha256Sum = *f.SHA256Sum
		}
		err = stream.Send(&storage.StreamFile{
			Data: d,
		})

		if err != nil {
			return err
		}
	}

	if in.BufferSize > 0 {
		bufferSize = in.BufferSize
	}

	buf = make([]byte, bufferSize)

	var n int
	for {
		n, err = sFile.Read(buf)
		if err != nil && err != io.EOF {
			break
		}
		if n == 0 {
			break
		}
		err = stream.Send(&storage.StreamFile{
			Data: &storage.StreamFile_Chunk{
				Chunk: buf[:n],
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

	result := &storage.UploadFileUrlResponse{
		Id:   fileRequest.Id,
		Code: storage.UploadStatusCode_Ok,
		Url:  publicUrl,
		Size: fileRequest.Size,
		Mime: fileRequest.MimeType,
		// TODO ADD Server
	}

	if fileRequest.SHA256Sum != nil {
		result.Sha256Sum = *fileRequest.SHA256Sum
	}

	return result, nil
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

func (api *file) SafeUploadFile(in gogrpc.FileService_SafeUploadFileServer) error {
	var su *app.SafeUpload
	res, gErr := in.Recv()
	if gErr != nil {
		wlog.Error(gErr.Error())
		return gErr
	}

	switch r := res.Data.(type) {
	case *storage.SafeUploadFileRequest_UploadId:
		su, gErr = app.RecoverySafeUploadProcess(r.UploadId)
		if gErr != nil {
			return gErr
		}
		break
	case *storage.SafeUploadFileRequest_Metadata_:
		var fileRequest model.JobUploadFile
		fileRequest.DomainId = r.Metadata.DomainId
		fileRequest.Name = r.Metadata.Name

		fileRequest.MimeType = r.Metadata.MimeType
		fileRequest.Uuid = r.Metadata.Uuid
		fileRequest.ViewName = &r.Metadata.Name
		var pid *int
		if r.Metadata.ProfileId > 0 {
			pid = model.NewInt(int(r.Metadata.ProfileId))
		}
		su = api.ctrl.App().NewSafeUpload(pid, &fileRequest)
		break
	default:
		gErr = errors.New("bad request")
		return gErr
	}

	gErr = in.Send(&storage.SafeUploadFileResponse{
		Data: &storage.SafeUploadFileResponse_Part_{
			Part: &storage.SafeUploadFileResponse_Part{
				UploadId: su.Id(),
				Size:     int64(su.Size()),
			},
		},
	})

	if gErr != nil {
		su.SetError(gErr)
		return gErr
	}

	var chunk *storage.SafeUploadFileRequest_Chunk
	var ok bool
	for {
		res, gErr = in.Recv()
		if gErr != nil {
			break
		}

		if chunk, ok = res.Data.(*storage.SafeUploadFileRequest_Chunk); !ok {
			gErr = errors.New("streaming data error: bad SafeUploadFileRequest_Chunk")
			break
		}

		if len(chunk.Chunk) == 0 {
			break
		}

		su.Write(chunk.Chunk)
	}

	if gErr != nil && gErr != io.EOF {
		su.Sleep()
		wlog.Error(gErr.Error())
		return gErr
	} else {
		su.CloseWrite()
	}

	<-su.WaitUploaded()
	fileRequest := su.File()
	var err engine.AppError
	var publicUrl string
	if publicUrl, err = api.ctrl.GeneratePreSignetResourceSignature(model.AnyFileRouteName, "download", fileRequest.Id, fileRequest.DomainId); err != nil {
		return err
	}

	metadata := &storage.SafeUploadFileResponse_Metadata{
		FileId:   fileRequest.Id,
		FileUrl:  publicUrl,
		Size:     fileRequest.Size,
		Code:     storage.UploadStatusCode_Ok,
		Server:   api.publicHost,
		Name:     fileRequest.GetViewName(),
		Uuid:     fileRequest.Uuid,
		MimeType: fileRequest.GetMimeType(),
	}

	if fileRequest.SHA256Sum != nil {
		metadata.Sha256Sum = *fileRequest.SHA256Sum
	}

	return in.Send(&storage.SafeUploadFileResponse{
		Data: &storage.SafeUploadFileResponse_Metadata_{
			Metadata: metadata,
		}})
}
