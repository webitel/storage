package grpc_api

import (
	"github.com/webitel/storage/app"
	"github.com/webitel/storage/controller"
	"github.com/webitel/storage/gen/storage"
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

	storage.RegisterBackendProfileServiceServer(server, api.backendProfiles)
	storage.RegisterMediaFileServiceServer(server, api.media)
	storage.RegisterFileServiceServer(server, api.file)
	storage.RegisterCognitiveProfileServiceServer(server, api.cognitiveProfile)
	storage.RegisterFileTranscriptServiceServer(server, api.fileTranscript)
	storage.RegisterImportTemplateServiceServer(server, api.importTemplate)
	storage.RegisterFilePoliciesServiceServer(server, api.filePolicies)
}
