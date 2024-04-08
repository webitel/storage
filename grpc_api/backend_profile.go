package grpc_api

import (
	"context"
	"fmt"

	gogrpc "buf.build/gen/go/webitel/storage/grpc/go/_gogrpc"
	storage "buf.build/gen/go/webitel/storage/protocolbuffers/go"
	"github.com/webitel/storage/controller"
	"github.com/webitel/storage/model"
)

type backendProfiles struct {
	ctrl *controller.Controller
	gogrpc.UnsafeBackendProfileServiceServer
}

func NewBackendProfileApi(c *controller.Controller) *backendProfiles {
	return &backendProfiles{ctrl: c}
}

func (api *backendProfiles) CreateBackendProfile(ctx context.Context, in *storage.CreateBackendProfileRequest) (*storage.BackendProfile, error) {
	session, err := api.ctrl.GetSessionFromCtx(ctx)
	if err != nil {
		return nil, err
	}

	profile := &model.FileBackendProfile{
		Name:        in.GetName(),
		Description: in.GetDescription(),
		ExpireDay:   int(in.GetExpireDays()),
		Priority:    int(in.GetPriority()),
		Disabled:    in.GetDisabled(),
		MaxSizeMb:   int(in.GetMaxSize()),
		Properties:  toStorageBackendProperties(in.GetProperties()),
		Type:        model.StorageBackendTypeFromString(in.GetType()),
	}

	profile, err = api.ctrl.CreateBackendProfile(session, profile)
	if err != nil {
		return nil, err
	}

	return toGrpcProfile(profile), nil
}

func (api *backendProfiles) SearchBackendProfile(ctx context.Context, in *storage.SearchBackendProfileRequest) (*storage.ListBackendProfile, error) {
	session, err := api.ctrl.GetSessionFromCtx(ctx)
	if err != nil {
		return nil, err
	}

	var list []*model.FileBackendProfile
	var endOfData bool

	rec := &model.SearchFileBackendProfile{
		ListRequest: model.ListRequest{
			Q:       in.GetQ(),
			Page:    int(in.GetPage()),
			PerPage: int(in.GetSize()),
			Fields:  in.Fields,
			Sort:    in.Sort,
		},
		Ids: in.Id,
	}

	list, endOfData, err = api.ctrl.SearchBackendProfile(session, session.Domain(0), rec)

	if err != nil {
		return nil, err
	}

	items := make([]*storage.BackendProfile, 0, len(list))
	for _, v := range list {
		items = append(items, toGrpcProfile(v))
	}
	return &storage.ListBackendProfile{
		Next:  !endOfData,
		Items: items,
	}, nil
}

func (api *backendProfiles) ReadBackendProfile(ctx context.Context, in *storage.ReadBackendProfileRequest) (*storage.BackendProfile, error) {
	session, err := api.ctrl.GetSessionFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	var profile *model.FileBackendProfile

	profile, err = api.ctrl.GetBackendProfile(session, in.GetId(), in.GetDomainId())
	if err != nil {
		return nil, err
	}

	return toGrpcProfile(profile), nil
}

func (api *backendProfiles) UpdateBackendProfile(ctx context.Context, in *storage.UpdateBackendProfileRequest) (*storage.BackendProfile, error) {
	session, err := api.ctrl.GetSessionFromCtx(ctx)
	if err != nil {
		return nil, err
	}

	profile := &model.FileBackendProfile{
		DomainRecord: model.DomainRecord{
			Id:       in.GetId(),
			DomainId: session.Domain(in.GetDomainId()),
		},
		Name:        in.GetName(),
		Description: in.GetDescription(),
		ExpireDay:   int(in.GetExpireDays()),
		Priority:    int(in.GetPriority()),
		Disabled:    in.GetDisabled(),
		MaxSizeMb:   int(in.GetMaxSize()),
		Properties:  toStorageBackendProperties(in.GetProperties()),
	}

	profile, err = api.ctrl.UpdateBackendProfile(session, profile)
	if err != nil {
		return nil, err
	}

	return toGrpcProfile(profile), nil
}

func (api *backendProfiles) PatchBackendProfile(ctx context.Context, in *storage.PatchBackendProfileRequest) (*storage.BackendProfile, error) {
	session, err := api.ctrl.GetSessionFromCtx(ctx)
	if err != nil {
		return nil, err
	}

	var profile *model.FileBackendProfile
	patch := &model.FileBackendProfilePath{}

	for _, v := range in.Fields {
		switch v {
		case "name":
			patch.Name = model.NewString(in.GetName())
		case "description":
			patch.Description = model.NewString(in.GetDescription())
		case "max_size":
			patch.MaxSizeMb = model.NewInt(int(in.GetMaxSize()))
		case "priority":
			patch.Priority = model.NewInt(int(in.GetPriority()))
		case "disabled":
			patch.Disabled = model.NewBool(in.GetDisabled())
		}
	}

	profile, err = api.ctrl.PatchBackendProfile(session, in.GetDomainId(), in.GetId(), patch)
	if err != nil {
		return nil, err
	}

	return toGrpcProfile(profile), nil
}

func (api *backendProfiles) DeleteBackendProfile(ctx context.Context, in *storage.DeleteBackendProfileRequest) (*storage.BackendProfile, error) {
	session, err := api.ctrl.GetSessionFromCtx(ctx)
	if err != nil {
		return nil, err
	}

	var profile *model.FileBackendProfile
	profile, err = api.ctrl.DeleteBackendProfile(session, in.GetDomainId(), in.GetId())
	if err != nil {
		return nil, err
	}

	return toGrpcProfile(profile), nil
}

func toGrpcProfile(src *model.FileBackendProfile) *storage.BackendProfile {
	// nullify password (task: WTEL-4344)
	src.Properties.Remove(model.BackendProfileAccessKeyField)
	return &storage.BackendProfile{
		Id:          src.Id,
		CreatedAt:   src.CreatedAt,
		CreatedBy:   GetProtoLookup(src.CreatedBy),
		UpdatedAt:   src.UpdatedAt,
		UpdatedBy:   GetProtoLookup(src.UpdatedBy),
		DataSize:    int64(src.DataSize),
		DataCount:   src.DataCount,
		Name:        src.Name,
		ExpireDays:  int32(src.ExpireDay),
		MaxSize:     int64(src.MaxSizeMb),
		Priority:    int32(src.Priority),
		Type:        src.Type.String(),
		Properties:  toGrpcBackendProperties(src.Properties), //FIXME allow proto json
		Description: src.Description,
		Disabled:    src.Disabled,
	}
}

// FIXME
func toStorageBackendProperties(src map[string]string) model.StringInterface {
	out := make(map[string]interface{})
	for k, v := range src {
		out[k] = v
	}
	return out
}

func toGrpcBackendProperties(src model.StringInterface) map[string]string {
	out := make(map[string]string)
	for k, v := range src {
		out[k] = fmt.Sprintf("%v", v)
	}
	return out
}
