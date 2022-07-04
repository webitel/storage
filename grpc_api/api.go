package grpc_api

import (
	"github.com/webitel/protos/storage"
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
}

func Init(a *app.App, server *grpc.Server) {
	api := &API{
		app: a,
	}

	ctrl := controller.NewController(a)
	api.backendProfiles = NewBackendProfileApi(ctrl)
	api.cognitiveProfile = NewCognitiveProfileApi(ctrl)
	api.media = NewMediaApi(ctrl)
	api.file = NewFileApi(a.Config().ProxyUploadUrl, ctrl)
	api.fileTranscript = NewFileTranscriptApi(ctrl)
	api.importTemplate = NewImportTemplateApi(ctrl)

	storage.RegisterBackendProfileServiceServer(server, api.backendProfiles)
	storage.RegisterMediaFileServiceServer(server, api.media)
	storage.RegisterFileServiceServer(server, api.file)
	storage.RegisterCognitiveProfileServiceServer(server, api.cognitiveProfile)
	storage.RegisterFileTranscriptServiceServer(server, api.fileTranscript)
	storage.RegisterImportTemplateServiceServer(server, api.importTemplate)
}
