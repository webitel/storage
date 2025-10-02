package grpc_api

import (
	"context"
	"errors"
	"fmt"
	"github.com/webitel/storage/app"
	"io"
	"net/http"
	"net/url"

	"github.com/webitel/storage/controller"
	"github.com/webitel/storage/gen/storage"
	"github.com/webitel/storage/model"
	"github.com/webitel/wlog"
)

var (
	ErrCancel = errors.New("cancel")
)

type file struct {
	ctrl       *controller.Controller
	curl       *http.Client
	publicHost string
	storage.UnsafeFileServiceServer
}

func NewFileApi(proxy string, ph string, api *controller.Controller) *file {
	c := &file{
		ctrl:       api,
		publicHost: ph,
	}
	if proxy != "" {
		proxyUrl, err := url.Parse(proxy)
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

	metadata, ok := res.Data.(*storage.UploadFileRequest_Metadata_)
	if !ok {
		gErr = errors.New("bad metadata")
		return gErr
	}

	var fileRequest model.JobUploadFile
	fileRequest.DomainId = metadata.Metadata.DomainId
	fileRequest.MimeType = metadata.Metadata.MimeType
	fileRequest.Uuid = metadata.Metadata.Uuid

	fileRequest.ViewName = &metadata.Metadata.Name
	fileRequest.Channel = model.NewString(channelType(metadata.Metadata.Channel))
	fileRequest.GenerateThumbnail = metadata.Metadata.GetGenerateThumbnail()
	if metadata.Metadata.UploadedBy > 0 {
		fileRequest.UploadedBy = &model.Lookup{Id: int(metadata.Metadata.UploadedBy)}
	}
	if metadata.Metadata.CreatedAt > 0 {
		fileRequest.CreatedAt = metadata.Metadata.CreatedAt
	}

	// TODO DEV-5174
	if metadata.Metadata.Channel == storage.UploadFileChannel_MailChannel {
		fileRequest.Name = model.NewId()[:6] + "_" + metadata.Metadata.Name
	} else {
		fileRequest.Name = metadata.Metadata.Name
	}

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

	var err model.AppError
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
		Malware: getMalware(fileRequest.Malware),
	}

	if fileRequest.SHA256Sum != nil {
		result.Sha256Sum = *fileRequest.SHA256Sum
	}

	return in.SendAndClose(result)
}

func getMalware(in *model.MalwareScan) *storage.FileMalwareScan {
	if in == nil {
		return nil
	}

	malware := &storage.FileMalwareScan{
		Found:       false,
		Status:      in.Status,
		Description: "",
	}
	if in.Desc != nil {
		malware.Description = *in.Desc
		malware.Found = true
	}

	return malware
}

func (api *file) GenerateFileLink(ctx context.Context, in *storage.GenerateFileLinkRequest) (*storage.GenerateFileLinkResponse, error) {
	uri, err := api.ctrl.GeneratePreSignedResourceSignatureBulk(in.GetFileId(), in.GetDomainId(), model.AnyFileRouteName, in.GetAction(), in.GetSource(), in.GetQuery())
	if err != nil {
		return nil, err
	}

	response := &storage.GenerateFileLinkResponse{
		Url:     uri,
		BaseUrl: api.publicHost,
	}

	if in.Metadata {
		var f model.BaseFile
		switch in.Source {
		case "file":
			f, err = api.ctrl.App().Store.File().Metadata(in.GetDomainId(), in.GetFileId())
		default:
			f, err = api.ctrl.App().Store.MediaFile().Metadata(in.GetDomainId(), in.GetFileId())
		}

		if err != nil {
			return nil, err
		}

		response.Metadata = &storage.GenerateFileLinkResponse_Metadata{
			Id:       in.GetFileId(),
			Name:     f.GetViewName(),
			MimeType: f.GetMimeType(),
			Size:     f.GetSize(),
		}
	}

	return response, nil
}

func (api *file) BulkGenerateFileLink(ctx context.Context, in *storage.BulkGenerateFileLinkRequest) (*storage.BulkGenerateFileLinkResponse, error) {
	l := len(in.Files)
	items := make([]*storage.GenerateFileLinkResponse, l, l)
	var uri string
	var err error

	for k, v := range in.GetFiles() {
		uri, err = api.ctrl.GeneratePreSignetResourceSignature(model.AnyFileRouteName, v.GetAction(), v.GetFileId(), v.GetDomainId())
		if err == nil {
			items[k] = &storage.GenerateFileLinkResponse{
				Url:     uri,
				BaseUrl: api.publicHost,
			}
		}
	}

	return &storage.BulkGenerateFileLinkResponse{
		Links: items,
	}, nil
}

func (api *file) DownloadFile(in *storage.DownloadFileRequest, stream storage.FileService_DownloadFileServer) error {
	var sFile io.ReadCloser
	var err error
	var buf []byte
	var bufferSize int64 = 4 * 1024

	f, backend, appErr := api.ctrl.InsecureGetFileWithProfile(in.DomainId, in.Id)
	if appErr != nil {
		return appErr
	}

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
		if f.Thumbnail != nil {
			d.Metadata.Thumbnail = &storage.Thumbnail{
				MimeType: f.Thumbnail.MimeType,
				Size:     f.Thumbnail.Size,
				Scale:    f.Thumbnail.Scale,
			}
		}
		err = stream.Send(&storage.StreamFile{
			Data: d,
		})

		if err != nil {
			return err
		}
	}

	if in.FetchThumbnail && f.Thumbnail != nil {
		f.BaseFile = f.Thumbnail.BaseFile
	}

	sFile, appErr = backend.Reader(f, in.Offset)
	if appErr != nil {
		return appErr
	}

	defer sFile.Close()

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

	if err != nil && err != io.EOF {
		wlog.Error(fmt.Sprintf("DownloadFile \"%s\" error: %s", f.Name, err.Error()))
	}

	return nil
}

func (api *file) UploadFileUrl(ctx context.Context, in *storage.UploadFileUrlRequest) (*storage.UploadFileUrlResponse, error) {
	var err model.AppError
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
	fileRequest.Channel = model.NewString(channelType(in.Channel))
	fileRequest.GenerateThumbnail = in.GetGenerateThumbnail()
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
		Id:      fileRequest.Id,
		Code:    storage.UploadStatusCode_Ok,
		Url:     publicUrl,
		Size:    fileRequest.Size,
		Mime:    fileRequest.MimeType,
		Server:  api.publicHost,
		Malware: getMalware(fileRequest.Malware),
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

func (api *file) RestoreFiles(ctx context.Context, in *storage.RestoreFilesRequest) (*storage.RestoreFilesResponse, error) {
	session, err := api.ctrl.GetSessionFromCtx(ctx)
	if err != nil {
		return nil, err
	}

	err = api.ctrl.RestoreFiles(ctx, session, in.Id)
	if err != nil {
		return nil, err
	}

	return &storage.RestoreFilesResponse{}, nil
}

func (api *file) DeleteQuarantineFiles(ctx context.Context, in *storage.DeleteQuarantineFilesRequest) (*storage.DeleteFilesResponse, error) {
	session, err := api.ctrl.GetSessionFromCtx(ctx)
	if err != nil {
		return nil, err
	}

	err = api.ctrl.DeleteQuarantineFiles(session, in.Id)
	if err != nil {
		return nil, err
	}

	return &storage.DeleteFilesResponse{}, nil
}

func (api *file) SafeUploadFile(in storage.FileService_SafeUploadFileServer) error {
	var su *app.SafeUpload
	ctx := in.Context()
	res, gErr := in.Recv()
	if gErr != nil {
		wlog.Error(gErr.Error())
		return gErr
	}

	switch r := res.Data.(type) {
	case *storage.SafeUploadFileRequest_UploadId:
		su, gErr = app.RecoverySafeUploadProcess(ctx, r.UploadId)
		if gErr != nil {
			return gErr
		}
		break
	case *storage.SafeUploadFileRequest_Metadata_:
		var fileRequest model.JobUploadFile
		fileRequest.DomainId = r.Metadata.DomainId
		fileRequest.Name = model.NewId()[0:6] + "_" + r.Metadata.Name

		fileRequest.MimeType = r.Metadata.MimeType
		fileRequest.Uuid = r.Metadata.Uuid
		fileRequest.ViewName = &r.Metadata.Name
		fileRequest.Channel = model.NewString(channelType(r.Metadata.Channel))
		fileRequest.GenerateThumbnail = r.Metadata.GetGenerateThumbnail()
		var pid *int
		if r.Metadata.ProfileId > 0 {
			pid = model.NewInt(int(r.Metadata.ProfileId))
		}
		su, gErr = api.ctrl.App().NewSafeUpload(pid, &fileRequest)
		if gErr != nil {
			return gErr
		}
		su.SetProgress(r.Metadata.Progress)
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
	for {
		res, gErr = in.Recv()
		if gErr != nil {
			break
		}

		switch res.Data.(type) {
		case *storage.SafeUploadFileRequest_Chunk:
			chunk = res.Data.(*storage.SafeUploadFileRequest_Chunk)
		case *storage.SafeUploadFileRequest_Cancel:
			su.SetError(ErrCancel)
			return in.Send(&storage.SafeUploadFileResponse{})
		default:
			gErr = errors.New("streaming data error: bad SafeUploadFileRequest_Chunk")
			break
		}

		if len(chunk.Chunk) == 0 {
			break
		}

		gErr = su.Write(chunk.Chunk)
		if gErr != nil {
			break
		}

		if su.Progress {
			in.Send(&storage.SafeUploadFileResponse{
				Data: &storage.SafeUploadFileResponse_Progress_{
					Progress: &storage.SafeUploadFileResponse_Progress{
						Uploaded: int64(su.Size()),
					},
				},
			})
		}
	}

	policyErr := model.IsFilePolicyError(gErr)
	if gErr != nil && gErr != io.EOF && !policyErr {
		su.Sleep()
		wlog.Error(gErr.Error())
		return gErr
	} else {
		su.CloseWrite()
		if policyErr {
			return gErr
		}
	}

	<-su.WaitUploaded()
	fileRequest := su.File()
	var err model.AppError
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
		Malware:  getMalware(fileRequest.Malware),
	}

	if fileRequest.SHA256Sum != nil {
		metadata.Sha256Sum = *fileRequest.SHA256Sum
	}

	return in.Send(&storage.SafeUploadFileResponse{
		Data: &storage.SafeUploadFileResponse_Metadata_{
			Metadata: metadata,
		}})
}

func (api *file) SearchFiles(ctx context.Context, in *storage.SearchFilesRequest) (*storage.ListFile, error) {
	session, err := api.ctrl.GetSessionFromCtx(ctx)
	if err != nil {
		return nil, err
	}

	var list []*model.File
	var endOfData bool

	search := &model.SearchFile{
		ListRequest: model.ListRequest{
			Q:       in.GetQ(),
			Page:    int(in.GetPage()),
			PerPage: int(in.GetSize()),
			Fields:  in.Fields,
			Sort:    in.Sort,
		},
		Ids:            in.Id,
		UploadedAt:     nil,
		UploadedBy:     in.UploadedBy,
		Channels:       channelsType(in.Channel),
		RetentionUntil: nil,
	}

	if in.UploadedAt != nil {
		search.UploadedAt = &model.FilterBetween{
			From: in.GetUploadedAt().GetFrom(),
			To:   in.GetUploadedAt().GetTo(),
		}
	}

	if in.RetentionUntil != nil {
		search.RetentionUntil = &model.FilterBetween{
			From: in.GetRetentionUntil().GetFrom(),
			To:   in.GetRetentionUntil().GetTo(),
		}
	}

	list, endOfData, err = api.ctrl.SearchFile(ctx, session, search)

	if err != nil {
		return nil, err
	}

	items := make([]*storage.File, 0, len(list))
	for _, v := range list {
		items = append(items, toGrpcFile(v))
	}
	return &storage.ListFile{
		Next:  !endOfData,
		Items: items,
	}, nil
}

func (api *file) SearchScreenRecordings(ctx context.Context, in *storage.SearchScreenRecordingsRequest) (*storage.ListFile, error) {
	session, err := api.ctrl.GetSessionFromCtx(ctx)
	if err != nil {
		return nil, err
	}

	search := &model.SearchFile{
		ListRequest: model.ListRequest{
			Q:       in.GetQ(),
			Page:    int(in.GetPage()),
			PerPage: int(in.GetSize()),
			Fields:  in.Fields,
			Sort:    in.Sort,
		},
		Ids:        in.Id,
		Removed:    model.NewBool(false),
		UploadedBy: []int64{in.GetUserId()},
		Channels:   []string{channelType(in.GetChannel())},
	}

	switch in.GetChannel() {
	case storage.UploadFileChannel_ScreenSharingChannel:
		search.Channels = []string{model.UploadFileChannelScreenShare}
	case storage.UploadFileChannel_ScreenshotChannel:
		search.Channels = []string{model.UploadFileChannelScreenshot}
	default:
		return nil, model.NewBadRequestError("grpc.screen_file", "bad channel")
	}

	if in.UploadedAt != nil {
		search.UploadedAt = &model.FilterBetween{
			From: in.GetUploadedAt().GetFrom(),
			To:   in.GetUploadedAt().GetTo(),
		}
	}

	if in.RetentionUntil != nil {
		search.RetentionUntil = &model.FilterBetween{
			From: in.GetRetentionUntil().GetFrom(),
			To:   in.GetRetentionUntil().GetTo(),
		}
	}

	output, next, err := api.ctrl.SearchScreenRecordings(ctx, session, search)
	if err != nil {
		return nil, err
	}

	items := make([]*storage.File, 0, len(output))
	for _, v := range output {
		items = append(items, toGrpcFile(v))
	}

	return &storage.ListFile{
		Next:  !next,
		Items: items,
	}, nil
}

func (api *file) SearchScreenRecordingsByAgent(ctx context.Context, in *storage.SearchScreenRecordingsByAgentRequest) (*storage.ListFile, error) {
	session, err := api.ctrl.GetSessionFromCtx(ctx)
	if err != nil {
		return nil, err
	}

	search := &model.SearchFile{
		ListRequest: model.ListRequest{
			Q:       in.GetQ(),
			Page:    int(in.GetPage()),
			PerPage: int(in.GetSize()),
			Fields:  in.Fields,
			Sort:    in.Sort,
		},
		Ids:      in.Id,
		Removed:  model.NewBool(false),
		Channels: []string{channelType(in.GetChannel())},
		AgentIds: []int{int(in.GetAgentId())},
	}

	switch in.GetChannel() {
	case storage.UploadFileChannel_ScreenSharingChannel:
		search.Channels = []string{model.UploadFileChannelScreenShare}
	case storage.UploadFileChannel_ScreenshotChannel:
		search.Channels = []string{model.UploadFileChannelScreenshot}
	default:
		return nil, model.NewBadRequestError("grpc.screen_file", "bad channel")
	}

	if in.UploadedAt != nil {
		search.UploadedAt = &model.FilterBetween{
			From: in.GetUploadedAt().GetFrom(),
			To:   in.GetUploadedAt().GetTo(),
		}
	}

	if in.RetentionUntil != nil {
		search.RetentionUntil = &model.FilterBetween{
			From: in.GetRetentionUntil().GetFrom(),
			To:   in.GetRetentionUntil().GetTo(),
		}
	}

	output, next, err := api.ctrl.SearchScreenRecordings(ctx, session, search)
	if err != nil {
		return nil, err
	}

	items := make([]*storage.File, 0, len(output))
	for _, v := range output {
		items = append(items, toGrpcFile(v))
	}

	return &storage.ListFile{
		Next:  !next,
		Items: items,
	}, nil
}

func (api *file) DeleteScreenRecordings(ctx context.Context, in *storage.DeleteScreenRecordingsRequest) (*storage.DeleteFilesResponse, error) {
	session, err := api.ctrl.GetSessionFromCtx(ctx)
	if err != nil {
		return nil, err
	}

	err = api.ctrl.DeleteScreenRecordings(ctx, session, in.UserId, in.Id)
	if err != nil {
		return nil, err
	}

	return &storage.DeleteFilesResponse{}, nil

}

func (api *file) DeleteScreenRecordingsByAgent(ctx context.Context, in *storage.DeleteScreenRecordingsByAgentRequest) (*storage.DeleteFilesResponse, error) {
	session, err := api.ctrl.GetSessionFromCtx(ctx)
	if err != nil {
		return nil, err
	}

	err = api.ctrl.DeleteScreenRecordingsByAgent(ctx, session, int(in.AgentId), in.Id)
	if err != nil {
		return nil, err
	}

	return &storage.DeleteFilesResponse{}, nil
}

func channelsType(channels []storage.UploadFileChannel) []string {
	l := make([]string, 0, len(channels))
	for _, v := range channels {
		l = append(l, channelType(v))
	}
	return l
}

func toGrpcFile(src *model.File) *storage.File {
	f := &storage.File{
		Id:             src.Id,
		UploadedAt:     model.TimeToInt64(src.UploadedAt),
		UploadedBy:     GetProtoLookup(src.UploadedBy),
		Name:           src.Name,
		MimeType:       src.MimeType,
		ReferenceId:    src.Uuid,
		Size:           src.Size,
		Thumbnail:      nil,
		RetentionUntil: model.TimeToInt64(src.RetentionUntil),
		Uuid:           src.Uuid,
	}

	if src.ViewName != nil {
		f.ViewName = *src.ViewName
	}

	if src.SHA256Sum != nil {
		f.Sha256Sum = *src.SHA256Sum
	}

	if src.BaseFile.Channel != nil {
		f.Channel = channelTypeGrpc(*src.BaseFile.Channel)
	}

	if src.Thumbnail != nil {
		f.Thumbnail = &storage.Thumbnail{
			MimeType: src.Thumbnail.MimeType,
			Size:     src.Thumbnail.Size,
			Scale:    src.Thumbnail.Scale,
		}
	}

	return f
}

func channelType(channel storage.UploadFileChannel) string {
	switch channel {
	case storage.UploadFileChannel_CallChannel:
		return model.UploadFileChannelCall
	case storage.UploadFileChannel_MailChannel:
		return model.UploadFileChannelMail
	case storage.UploadFileChannel_MediaChannel:
		return model.UploadFileChannelMedia
	case storage.UploadFileChannel_LogChannel:
		return model.UploadFileChannelLog
	case storage.UploadFileChannel_ScreenSharingChannel:
		return model.UploadFileChannelScreenShare
	case storage.UploadFileChannel_ScreenshotChannel:
		return model.UploadFileChannelScreenshot

	default:
		return model.UploadFileChannelChat

	}
}

func channelTypeGrpc(channel string) storage.UploadFileChannel {
	switch channel {
	case model.UploadFileChannelCall:
		return storage.UploadFileChannel_CallChannel
	case model.UploadFileChannelMail:
		return storage.UploadFileChannel_MailChannel
	case model.UploadFileChannelMedia:
		return storage.UploadFileChannel_MediaChannel
	case model.UploadFileChannelLog:
		return storage.UploadFileChannel_LogChannel
	case model.UploadFileChannelScreenshot:
		return storage.UploadFileChannel_ScreenshotChannel
	case model.UploadFileChannelScreenShare:
		return storage.UploadFileChannel_ScreenSharingChannel
	default:
		return storage.UploadFileChannel_UnknownChannel

	}
}
