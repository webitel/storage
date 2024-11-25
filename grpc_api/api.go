package grpc_api

import (
	gogrpc "buf.build/gen/go/webitel/storage/grpc/go/_gogrpc"
	"github.com/webitel/storage/app"
	"github.com/webitel/storage/controller"
	"google.golang.org/grpc"
)

type API struct {
	app              *app.App
	ctrl             *controller.Controller
	backendProfiles  *backendProfiles
	cognitiveProfile *cognitiveProfile
	media            *media
	file             *file
	fileTranscript   *fileTranscript
	importTemplate   *importTemplate
	filePolicies     *filePolicies
}

func Init(a *app.App, server *grpc.Server) {
	api := &API{
		app: a,
	}

	ctrl := controller.NewController(a)
	api.backendProfiles = NewBackendProfileApi(ctrl)
	api.cognitiveProfile = NewCognitiveProfileApi(ctrl)
	api.media = NewMediaApi(ctrl, a)
	api.file = NewFileApi(a.Config().ProxyUploadUrl, a.Config().ServiceSettings.PublicHost, ctrl)
	api.fileTranscript = NewFileTranscriptApi(ctrl)
	api.importTemplate = NewImportTemplateApi(ctrl)
	api.filePolicies = NewFilePoliciesApi(ctrl)

	gogrpc.RegisterBackendProfileServiceServer(server, api.backendProfiles)
	gogrpc.RegisterMediaFileServiceServer(server, api.media)
	gogrpc.RegisterFileServiceServer(server, api.file)
	gogrpc.RegisterCognitiveProfileServiceServer(server, api.cognitiveProfile)
	gogrpc.RegisterFileTranscriptServiceServer(server, api.fileTranscript)
	gogrpc.RegisterImportTemplateServiceServer(server, api.importTemplate)
	gogrpc.RegisterFilePoliciesServiceServer(server, api.filePolicies)
}
