package grpc_api

import (
	"bytes"
	"context"
	"encoding/json"
	"time"

	"github.com/golang/protobuf/jsonpb"

	"github.com/webitel/storage/model"
	"google.golang.org/protobuf/types/known/structpb"

	gogrpc "buf.build/gen/go/webitel/storage/grpc/go/_gogrpc"
	storage "buf.build/gen/go/webitel/storage/protocolbuffers/go"

	"github.com/webitel/storage/controller"
)

type cognitiveProfile struct {
	ctrl *controller.Controller
	gogrpc.UnsafeCognitiveProfileServiceServer
}

func NewCognitiveProfileApi(c *controller.Controller) *cognitiveProfile {
	return &cognitiveProfile{ctrl: c}
}

func (api *cognitiveProfile) CreateCognitiveProfile(ctx context.Context, in *storage.CreateCognitiveProfileRequest) (*storage.CognitiveProfile, error) {
	session, err := api.ctrl.GetSessionFromCtx(ctx)
	if err != nil {
		return nil, err
	}

	profile := &model.CognitiveProfile{
		Provider:    in.Provider.String(),
		Properties:  MarshalJsonpbToMap(in.GetProperties()),
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

	if len(in.Service) != 0 {
		rec.Service = make([]string, 0, len(in.Service))
		for _, v := range in.Service {
			rec.Service = append(rec.Service, v.String())
		}
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
		Properties:  MarshalJsonpbToMap(in.GetProperties()),
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
			p := MarshalJsonpbToMap(in.GetProperties())
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
	// nullify password
	src.Properties.Remove(model.CognitiveProfileKeyField)
	return &storage.CognitiveProfile{
		Id:          src.Id,
		CreatedAt:   getTimestamp(src.CreatedAt),
		CreatedBy:   GetProtoLookup(src.CreatedBy),
		UpdatedAt:   getTimestamp(src.UpdatedAt),
		UpdatedBy:   GetProtoLookup(src.UpdatedBy),
		Provider:    getProvider(src.Provider),
		Properties:  UnmarshalJsonpb([]byte(src.Properties.ToJson())),
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
	case storage.ProviderType_Google.String():
		return storage.ProviderType_Google
	case storage.ProviderType_ElevenLabs.String():
		return storage.ProviderType_ElevenLabs
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

var (
	jsonpbCodec = struct {
		jsonpb.Unmarshaler
		jsonpb.Marshaler
	}{
		Unmarshaler: jsonpb.Unmarshaler{

			// Whether to allow messages to contain unknown fields, as opposed to
			// failing to unmarshal.
			AllowUnknownFields: false, // bool

			// A custom URL resolver to use when unmarshaling Any messages from JSON.
			// If unset, the default resolution strategy is to extract the
			// fully-qualified type name from the type URL and pass that to
			// proto.MessageType(string).
			AnyResolver: nil,
		},
		Marshaler: jsonpb.Marshaler{

			// Whether to render enum values as integers, as opposed to string values.
			EnumsAsInts: false, // bool

			// Whether to render fields with zero values.
			EmitDefaults: true, // bool

			// A string to indent each level by. The presence of this field will
			// also cause a space to appear between the field separator and
			// value, and for newlines to be appear between fields and array
			// elements.
			Indent: "", // string

			// Whether to use the original (.proto) name for fields.
			OrigName: true, // bool

			// A custom URL resolver to use when marshaling Any messages to JSON.
			// If unset, the default resolution strategy is to extract the
			// fully-qualified type name from the type URL and pass that to
			// proto.MessageType(string).
			AnyResolver: nil,
		},
	}
)

func UnmarshalJsonpb(data []byte) *structpb.Value {
	var pb structpb.Value
	rd := bytes.NewReader(data)
	err := jsonpbCodec.Unmarshal(rd, &pb)
	if err != nil {
		return nil
	}
	return &pb
}

func MarshalJsonpbToMap(src *structpb.Value) model.StringInterface {
	var buf bytes.Buffer
	err := jsonpbCodec.Marshal(&buf, src)
	if err != nil {
		return nil
	}

	res := make(model.StringInterface)

	json.Unmarshal(buf.Bytes(), &res)
	return res
}
