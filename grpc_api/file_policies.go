package grpc_api

import (
	"context"
	"github.com/webitel/storage/controller"
	storage "github.com/webitel/storage/gen/storage"
	"github.com/webitel/storage/model"
	"unicode"
)

type filePolicies struct {
	ctrl *controller.Controller
	storage.UnsafeFilePoliciesServiceServer
}

var uploadFileChannelName = map[storage.UploadFileChannel]string{
	0: model.UploadFileChannelUnknown,
	1: model.UploadFileChannelChat,
	2: model.UploadFileChannelMail,
	3: model.UploadFileChannelCall,
	4: model.UploadFileChannelLog,
	5: model.UploadFileChannelMedia,
	6: model.UploadFileChannelKnowledgebase,
	7: model.UploadFileChannelCases,
}

func NewFilePoliciesApi(c *controller.Controller) *filePolicies {
	return &filePolicies{ctrl: c}
}

func (api *filePolicies) CreateFilePolicy(ctx context.Context, in *storage.CreateFilePolicyRequest) (*storage.FilePolicy, error) {
	session, err := api.ctrl.GetSessionFromCtx(ctx)
	if err != nil {
		return nil, err
	}

	policy := &model.FilePolicy{
		Enabled:       in.Enabled,
		Name:          in.GetName(),
		Description:   in.GetDescription(),
		SpeedUpload:   in.SpeedUpload,
		SpeedDownload: in.SpeedDownload,
		MimeTypes:     in.MimeTypes,
		Channels:      fileChannelsFromProto(in.Channels),
		RetentionDays: in.RetentionDays,
		MaxUploadSize: in.MaxUploadSize,
	}

	policy, err = api.ctrl.CreateFilePolicy(ctx, session, policy)
	if err != nil {
		return nil, err
	}

	return toGrpcFilePolicy(policy), nil
}

func (api *filePolicies) SearchFilePolicies(ctx context.Context, in *storage.SearchFilePoliciesRequest) (*storage.ListFilePolicies, error) {
	session, err := api.ctrl.GetSessionFromCtx(ctx)
	if err != nil {
		return nil, err
	}

	var list []*model.FilePolicy
	var endOfData bool

	rec := &model.SearchFilePolicy{
		ListRequest: model.ListRequest{
			Q:       in.GetQ(),
			Page:    int(in.GetPage()),
			PerPage: int(in.GetSize()),
			Fields:  in.Fields,
			Sort:    in.Sort,
		},
		Ids: in.Id,
	}

	list, endOfData, err = api.ctrl.SearchFilePolicies(ctx, session, rec)

	if err != nil {
		return nil, err
	}

	items := make([]*storage.FilePolicy, 0, len(list))
	for _, v := range list {
		items = append(items, toGrpcFilePolicy(v))
	}
	return &storage.ListFilePolicies{
		Next:  !endOfData,
		Items: items,
	}, nil
}

func (api *filePolicies) ReadFilePolicy(ctx context.Context, in *storage.ReadFilePolicyRequest) (*storage.FilePolicy, error) {
	session, err := api.ctrl.GetSessionFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	var policy *model.FilePolicy

	policy, err = api.ctrl.GetFilePolicy(ctx, session, in.GetId())
	if err != nil {
		return nil, err
	}

	return toGrpcFilePolicy(policy), nil
}

func (api *filePolicies) UpdateFilePolicy(ctx context.Context, in *storage.UpdateFilePolicyRequest) (*storage.FilePolicy, error) {
	session, err := api.ctrl.GetSessionFromCtx(ctx)
	if err != nil {
		return nil, err
	}

	policy := &model.FilePolicy{
		Id: in.Id,

		Enabled:       in.Enabled,
		Name:          in.GetName(),
		Description:   in.GetDescription(),
		SpeedUpload:   in.SpeedUpload,
		SpeedDownload: in.SpeedDownload,
		MimeTypes:     in.MimeTypes,
		Channels:      fileChannelsFromProto(in.Channels),
		RetentionDays: in.RetentionDays,
		MaxUploadSize: in.MaxUploadSize,
	}

	policy, err = api.ctrl.UpdateFilePolicy(ctx, session, in.Id, policy)
	if err != nil {
		return nil, err
	}

	return toGrpcFilePolicy(policy), nil
}

func (api *filePolicies) PatchFilePolicy(ctx context.Context, in *storage.PatchFilePolicyRequest) (*storage.FilePolicy, error) {
	session, err := api.ctrl.GetSessionFromCtx(ctx)
	if err != nil {
		return nil, err
	}

	var policy *model.FilePolicy
	patch := &model.FilePolicyPath{}

	for _, v := range in.Fields {
		switch v {

		case "enabled":
			patch.Enabled = &in.Enabled
		case "name":
			patch.Name = &in.Name
		case "description":
			patch.Description = &in.Description
		case "mime_types":
			patch.MimeTypes = in.MimeTypes
		case "channels":
			patch.Channels = fileChannelsFromProto(in.Channels)
		case "speed_download":
			patch.SpeedDownload = &in.SpeedDownload
		case "speed_upload":
			patch.SpeedUpload = &in.SpeedUpload
		case "retention_days":
			patch.RetentionDays = &in.RetentionDays
		case "max_upload_size":
			patch.MaxUploadSize = &in.MaxUploadSize
		}
	}

	policy, err = api.ctrl.PatchFilePolicy(ctx, session, in.GetId(), patch)
	if err != nil {
		return nil, err
	}

	return toGrpcFilePolicy(policy), nil
}

func (api *filePolicies) DeleteFilePolicy(ctx context.Context, in *storage.DeleteFilePolicyRequest) (*storage.FilePolicy, error) {
	session, err := api.ctrl.GetSessionFromCtx(ctx)
	if err != nil {
		return nil, err
	}

	var policy *model.FilePolicy
	policy, err = api.ctrl.DeleteFilePolicy(ctx, session, in.GetId())
	if err != nil {
		return nil, err
	}

	return toGrpcFilePolicy(policy), nil
}

func (api *filePolicies) MovePositionFilePolicy(ctx context.Context, in *storage.MovePositionFilePolicyRequest) (*storage.MovePositionFilePolicyResponse, error) {
	session, err := api.ctrl.GetSessionFromCtx(ctx)
	if err != nil {
		return nil, err
	}

	err = api.ctrl.ChangePositionFilePolicy(ctx, session, in.FromId, in.ToId)
	if err != nil {
		return nil, err
	}

	return &storage.MovePositionFilePolicyResponse{
		Success: true,
	}, nil

}

func (api *filePolicies) FilePolicyApply(ctx context.Context, in *storage.FilePolicyApplyRequest) (*storage.FilePolicyApplyResponse, error) {
	var count int64
	session, err := api.ctrl.GetSessionFromCtx(ctx)
	if err != nil {
		return nil, err
	}

	count, err = api.ctrl.ApplyFilePolicy(ctx, session, in.GetId())
	if err != nil {
		return nil, err
	}

	return &storage.FilePolicyApplyResponse{Count: count}, nil
}

func toGrpcFilePolicy(src *model.FilePolicy) *storage.FilePolicy {
	return &storage.FilePolicy{
		Id:            src.Id,
		CreatedAt:     getTimestamp(src.CreatedAt),
		CreatedBy:     GetProtoLookup(src.CreatedBy),
		UpdatedAt:     getTimestamp(src.UpdatedAt),
		UpdatedBy:     GetProtoLookup(src.UpdatedBy),
		Enabled:       src.Enabled,
		Name:          src.Name,
		Description:   src.Description,
		SpeedDownload: src.SpeedDownload,
		SpeedUpload:   src.SpeedUpload,
		MimeTypes:     src.MimeTypes,
		Channels:      fileChannelsProto(src.Channels),
		RetentionDays: src.RetentionDays,
		MaxUploadSize: src.MaxUploadSize,
		Position:      src.Position, //
	}

}

func fileChannelsFromProto(in []storage.UploadFileChannel) []string {
	var res []string
	var channel string
	var ok bool

	for _, v := range in {
		channel, ok = uploadFileChannelName[v]
		if !ok {
			channel = model.UploadFileChannelUnknown
		}

		res = append(res, channel)
	}

	return res
}

func fileChannelsProto(in []string) []storage.UploadFileChannel {
	var res []storage.UploadFileChannel
	var channel int32
	var ok bool
	for _, v := range in {
		channel, ok = storage.UploadFileChannel_value[capitalizeFirstLetterUnicode(v, true)+"Channel"]
		if !ok {
			channel = int32(storage.UploadFileChannel_UnknownChannel)
		}
		res = append(res, storage.UploadFileChannel(channel))
	}

	return res
}

func capitalizeFirstLetterUnicode(s string, upper bool) string {
	if len(s) == 0 {
		return s
	}
	runes := []rune(s)
	if upper {
		runes[0] = unicode.ToUpper(runes[0])
	} else {
		runes[0] = unicode.ToLower(runes[0])
	}
	return string(runes)
}
