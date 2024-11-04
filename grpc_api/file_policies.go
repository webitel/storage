package grpc_api

import (
	gogrpc "buf.build/gen/go/webitel/storage/grpc/go/_gogrpc"
	storage "buf.build/gen/go/webitel/storage/protocolbuffers/go"
	"context"
	"errors"
	"github.com/webitel/storage/controller"
	"github.com/webitel/storage/model"
)

type filePolicies struct {
	ctrl *controller.Controller
	gogrpc.UnsafeFilePoliciesServiceServer
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
		Channels:      in.Channels,
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
		Channels:      in.Channels,
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
			patch.Channels = in.Channels
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
	return nil, errors.New("TODO")
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
		Channels:      src.Channels,
		RetentionDays: src.RetentionDays,
		MaxUploadSize: src.MaxUploadSize,
		Position:      src.Position, //
	}
}
