package grpc_api

import (
	"context"

	"google.golang.org/protobuf/types/known/structpb"

	"github.com/webitel/storage/gen/storage"
	"github.com/webitel/storage/model"

	"github.com/webitel/storage/controller"
)

type importTemplate struct {
	ctrl *controller.Controller
	storage.UnsafeImportTemplateServiceServer
}

func NewImportTemplateApi(c *controller.Controller) *importTemplate {
	return &importTemplate{ctrl: c}
}

func (api *importTemplate) CreateImportTemplate(ctx context.Context, in *storage.CreateImportTemplateRequest) (*storage.ImportTemplate, error) {
	session, err := api.ctrl.GetSessionFromCtx(ctx)
	if err != nil {
		return nil, err
	}

	template := &model.ImportTemplate{
		Name:        in.GetName(),
		Description: in.GetDescription(),
		SourceType:  in.GetSourceType().String(),
		SourceId:    in.GetSourceId(),
		Parameters:  in.GetParameters().AsMap(),
	}

	if in.GetSource() != nil {
		template.SourceId = in.GetSource().GetId()
	}

	template, err = api.ctrl.CreateImportTemplate(session, template)
	if err != nil {
		return nil, err
	}

	return toGrpcImportTemplate(template), nil
}

func (api *importTemplate) SearchImportTemplate(ctx context.Context, in *storage.SearchImportTemplateRequest) (*storage.ListImportTemplate, error) {
	session, err := api.ctrl.GetSessionFromCtx(ctx)
	if err != nil {
		return nil, err
	}

	var list []*model.ImportTemplate
	var endOfData bool

	search := &model.SearchImportTemplate{
		ListRequest: model.ListRequest{
			Q:       in.GetQ(),
			Page:    int(in.GetPage()),
			PerPage: int(in.GetSize()),
			Fields:  in.Fields,
			Sort:    in.Sort,
		},
		Ids: in.Id,
	}

	list, endOfData, err = api.ctrl.SearchImportTemplates(session, search)

	if err != nil {
		return nil, err
	}

	items := make([]*storage.ImportTemplate, 0, len(list))
	for _, v := range list {
		items = append(items, toGrpcImportTemplate(v))
	}
	return &storage.ListImportTemplate{
		Next:  !endOfData,
		Items: items,
	}, nil
}

func (api *importTemplate) ReadImportTemplate(ctx context.Context, in *storage.ReadImportTemplateRequest) (*storage.ImportTemplate, error) {
	session, err := api.ctrl.GetSessionFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	var template *model.ImportTemplate

	template, err = api.ctrl.GetImportTemplate(session, in.GetId())
	if err != nil {
		return nil, err
	}

	return toGrpcImportTemplate(template), nil
}

func (api *importTemplate) UpdateImportTemplate(ctx context.Context, in *storage.UpdateImportTemplateRequest) (*storage.ImportTemplate, error) {
	session, err := api.ctrl.GetSessionFromCtx(ctx)
	if err != nil {
		return nil, err
	}

	template := &model.ImportTemplate{
		Id:          in.GetId(),
		Name:        in.GetName(),
		Description: in.GetDescription(),
		Parameters:  in.GetParameters().AsMap(),
	}

	if in.Source != nil {
		template.Source = &model.Lookup{
			Id: int(in.GetSource().GetId()),
		}
	}

	template, err = api.ctrl.UpdateImportTemplate(session, template)
	if err != nil {
		return nil, err
	}

	return toGrpcImportTemplate(template), nil
}

func (api *importTemplate) PatchImportTemplate(ctx context.Context, in *storage.PatchImportTemplateRequest) (*storage.ImportTemplate, error) {
	session, err := api.ctrl.GetSessionFromCtx(ctx)
	if err != nil {
		return nil, err
	}

	var template *model.ImportTemplate
	patch := &model.ImportTemplatePatch{}

	for _, v := range in.Fields {
		switch v {
		case "name":
			patch.Name = &in.Name
		case "description":
			patch.Description = &in.Description
		case "parameters":
			patch.Parameters = in.GetParameters().AsMap()
		}
	}

	template, err = api.ctrl.PatchImportTemplate(session, in.GetId(), patch)
	if err != nil {
		return nil, err
	}

	return toGrpcImportTemplate(template), nil
}

func (api *importTemplate) DeleteImportTemplate(ctx context.Context, in *storage.DeleteImportTemplateRequest) (*storage.ImportTemplate, error) {
	session, err := api.ctrl.GetSessionFromCtx(ctx)
	if err != nil {
		return nil, err
	}

	var template *model.ImportTemplate
	template, err = api.ctrl.DeleteImportTemplate(session, in.GetId())
	if err != nil {
		return nil, err
	}

	return toGrpcImportTemplate(template), nil
}

func toGrpcImportTemplate(src *model.ImportTemplate) *storage.ImportTemplate {
	t := storage.ImportTemplate{
		Id:          src.Id,
		Name:        src.Name,
		Description: src.Description,
		SourceType:  storage.ImportSourceType_Dialer, // todo fixme
		SourceId:    src.SourceId,
		Source:      GetProtoLookup(src.Source),
	}

	t.Parameters, _ = structpb.NewStruct(src.Parameters)

	return &t
}
