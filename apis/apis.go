package apis

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/webitel/storage/app"
	"github.com/webitel/storage/controller"
	"github.com/webitel/storage/model"
	"github.com/webitel/storage/web"
)

type RoutesPublic struct {
	Root    *mux.Router // ''
	ApiRoot *mux.Router // 'api/v2'

	CallRecordingsFiles *mux.Router // '/call files'
	MediaFiles          *mux.Router // '/media
	AnyFiles            *mux.Router // for chat
	Files               *mux.Router // for chat
	Jobs                *mux.Router
	Tts                 *mux.Router
}

type API struct {
	App          *app.App
	PublicRoutes *RoutesPublic
	ctrl         *controller.Controller
}

func Init(a *app.App, root *mux.Router) *API {
	api := &API{
		App:          a,
		PublicRoutes: &RoutesPublic{},
		ctrl:         controller.NewController(a),
	}
	api.PublicRoutes.Root = root
	api.PublicRoutes.ApiRoot = root.PathPrefix(model.API_URL_SUFFIX).Subrouter()

	api.PublicRoutes.MediaFiles = api.PublicRoutes.ApiRoot.PathPrefix("/media").Subrouter()
	api.PublicRoutes.CallRecordingsFiles = api.PublicRoutes.ApiRoot.PathPrefix("/recordings").Subrouter()
	api.PublicRoutes.Files = api.PublicRoutes.ApiRoot.PathPrefix("/file").Subrouter()
	api.PublicRoutes.Jobs = api.PublicRoutes.ApiRoot.PathPrefix("/jobs").Subrouter()
	api.PublicRoutes.Tts = api.PublicRoutes.ApiRoot.PathPrefix("/tts").Subrouter()

	api.PublicRoutes.AnyFiles = api.PublicRoutes.ApiRoot.PathPrefix(model.AnyFileRouteName).Subrouter()

	api.InitMediaFile()
	api.InitCallRecordingsFiles()
	api.InitAnyFile()
	api.InitFile()
	api.InitJobs()
	api.InitTts()

	return api
}

func (api *API) Handle404(w http.ResponseWriter, r *http.Request) {
	web.Handle404(api.App, w, r)
}

var ReturnStatusOK = web.ReturnStatusOK
