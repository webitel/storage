package grpc_api

import (
	"context"
	"time"

	"github.com/webitel/storage/model"

	"github.com/webitel/protos/storage"

	"github.com/webitel/storage/controller"
)

type cognitiveProfile struct {
	ctrl *controller.Controller
}

func NewCognitiveProfileApi(api *controller.Controller) *cognitiveProfile {
	return &cognitiveProfile{api}
}

func (api *cognitiveProfile) CreateCognitiveProfile(ctx context.Context, in *storage.CreateCognitiveProfileRequest) (*storage.CognitiveProfile, error) {
	session, err := api.ctrl.GetSessionFromCtx(ctx)
	if err != nil {
		return nil, err
	}

	profile := &model.CognitiveProfile{
		Provider:    in.Provider.String(),
		Properties:  toStorageBackendProperties(in.GetProperties()),
		Enabled:     in.Enabled,
		Name:        in.GetName(),
		Description: in.GetDescription(),
		Service:     in.Service.String(),
		Default:     in.Default,
	}

	profile, err = api.ctrl.CreateCognitiveProfile(session, profile)
	if err != nil {
		return nil, err
	}

	return toGrpcCognitiveProfile(profile), nil
}

func (api *cognitiveProfile) SearchCognitiveProfile(ctx context.Context, in *storage.SearchCognitiveProfileRequest) (*storage.ListCognitiveProfile, error) {
	session, err := api.ctrl.GetSessionFromCtx(ctx)
	if err != nil {
		return nil, err
	}

	var list []*model.CognitiveProfile
	var endOfData bool

	rec := &model.SearchCognitiveProfile{
		ListRequest: model.ListRequest{
			Q:       in.GetQ(),
			Page:    int(in.GetPage()),
			PerPage: int(in.GetSize()),
			Fields:  in.Fields,
			Sort:    in.Sort,
		},
		Ids:     in.Id,
		Enabled: in.Enabled,
	}

	list, endOfData, err = api.ctrl.SearchCognitiveProfile(session, session.Domain(0), rec)

	if err != nil {
		return nil, err
	}

	items := make([]*storage.CognitiveProfile, 0, len(list))
	for _, v := range list {
		items = append(items, toGrpcCognitiveProfile(v))
	}
	return &storage.ListCognitiveProfile{
		Next:  !endOfData,
		Items: items,
	}, nil
}

func (api *cognitiveProfile) ReadCognitiveProfile(ctx context.Context, in *storage.ReadCognitiveProfileRequest) (*storage.CognitiveProfile, error) {
	session, err := api.ctrl.GetSessionFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	var profile *model.CognitiveProfile

	profile, err = api.ctrl.GetCognitiveProfile(session, in.GetId(), 0)
	if err != nil {
		return nil, err
	}

	return toGrpcCognitiveProfile(profile), nil
}

func (api *cognitiveProfile) UpdateCognitiveProfile(ctx context.Context, in *storage.UpdateCognitiveProfileRequest) (*storage.CognitiveProfile, error) {
	session, err := api.ctrl.GetSessionFromCtx(ctx)
	if err != nil {
		return nil, err
	}

	profile := &model.CognitiveProfile{
		Id:          in.Id,
		DomainId:    session.Domain(0),
		Provider:    in.Provider.String(),
		Properties:  toStorageBackendProperties(in.GetProperties()),
		Enabled:     in.Enabled,
		Name:        in.GetName(),
		Description: in.GetDescription(),
		Service:     in.Service.String(),
		Default:     in.Default,
	}

	profile, err = api.ctrl.UpdateCognitiveProfile(session, profile)
	if err != nil {
		return nil, err
	}

	return toGrpcCognitiveProfile(profile), nil
}

func (api *cognitiveProfile) PatchCognitiveProfile(ctx context.Context, in *storage.PatchCognitiveProfileRequest) (*storage.CognitiveProfile, error) {
	session, err := api.ctrl.GetSessionFromCtx(ctx)
	if err != nil {
		return nil, err
	}

	var profile *model.CognitiveProfile
	patch := &model.CognitiveProfilePath{}

	for _, v := range in.Fields {
		switch v {
		case "provider":
			patch.Provider = model.NewString(in.Provider.String())
		case "properties":
			p := toStorageBackendProperties(in.GetProperties())
			patch.Properties = &p

		case "enabled":
			patch.Enabled = &in.Enabled
		case "name":
			patch.Name = &in.Name
		case "description":
			patch.Description = &in.Description
		case "service":
			patch.Service = model.NewString(in.Service.String())
		case "default":
			patch.Default = &in.Default
		}
	}

	profile, err = api.ctrl.PatchCognitiveProfile(session, 0, in.GetId(), patch)
	if err != nil {
		return nil, err
	}

	return toGrpcCognitiveProfile(profile), nil
}

func (api *cognitiveProfile) DeleteCognitiveProfile(ctx context.Context, in *storage.DeleteCognitiveProfileRequest) (*storage.CognitiveProfile, error) {
	session, err := api.ctrl.GetSessionFromCtx(ctx)
	if err != nil {
		return nil, err
	}

	var profile *model.CognitiveProfile
	profile, err = api.ctrl.DeleteCognitiveProfile(session, 0, in.GetId())
	if err != nil {
		return nil, err
	}

	return toGrpcCognitiveProfile(profile), nil
}

func toGrpcCognitiveProfile(src *model.CognitiveProfile) *storage.CognitiveProfile {
	return &storage.CognitiveProfile{
		Id:          src.Id,
		CreatedAt:   getTimestamp(src.CreatedAt),
		CreatedBy:   GetProtoLookup(src.CreatedBy),
		UpdatedAt:   getTimestamp(src.UpdatedAt),
		UpdatedBy:   GetProtoLookup(src.UpdatedBy),
		Provider:    getProvider(src.Provider),
		Properties:  toFrpcBackendProperties(src.Properties), //FIXME allow proto json
		Enabled:     src.Enabled,
		Name:        src.Name,
		Description: src.Description,
		Service:     getService(src.Service),
		Default:     src.Default,
	}
}

func getProvider(p string) storage.ProviderType {
	switch p {
	case storage.ProviderType_Microsoft.String():
		return storage.ProviderType_Microsoft
	default:
		return storage.ProviderType_DefaultProvider
	}
}
func getService(s string) storage.ServiceType {
	switch s {
	case storage.ServiceType_STT.String():
		return storage.ServiceType_STT
	case storage.ServiceType_TTS.String():
		return storage.ServiceType_TTS
	default:
		return storage.ServiceType_DefaultService
	}
}

func getTimestamp(t *time.Time) int64 {
	if t != nil {
		return t.UnixNano() / 1000
	}

	return 0
}
