package private

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/webitel/storage/app"
	"github.com/webitel/storage/model"
	"github.com/webitel/storage/web"
)

type RoutesInternal struct {
	Root     *mux.Router // ''
	ApiRoot  *mux.Router // 'sys'
	Files    *mux.Router // '/records'
	Media    *mux.Router // '/media'
	TTS      *mux.Router // '/tts'
	Redirect *mux.Router // for freeswitch redirection

}

type API struct {
	App    *app.App
	Routes *RoutesInternal
}

func Init(a *app.App, root *mux.Router) *API {
	api := &API{
		App:    a,
		Routes: &RoutesInternal{},
	}

	api.Routes.Root = root
	api.Routes.ApiRoot = root.PathPrefix(model.API_INTERNAL_URL_SUFFIX_V1).Subrouter()
	api.Routes.Files = api.Routes.ApiRoot.PathPrefix("/recordings").Subrouter()
	api.Routes.Media = api.Routes.ApiRoot.PathPrefix("/media").Subrouter()
	api.Routes.TTS = api.Routes.ApiRoot.PathPrefix("/tts").Subrouter()
	api.Routes.Redirect = api.Routes.ApiRoot.PathPrefix("/redirect").Subrouter()

	api.InitFile()
	api.InitMedia()
	api.InitTTS()
	api.InitRedirect()

	return api
}

func (api *API) Handle404(w http.ResponseWriter, r *http.Request) {
	web.Handle404(api.App, w, r)
}

//	func getTest(c *Context, w http.ResponseWriter, r *http.Request)  {
//		res := <-c.App.Store.FileBackendProfile().Get(1, "10.10.10.144")
//		if res.Err != nil {
//			c.Err = res.Err
//			return
//		}
//
//		w.WriteHeader(http.StatusOK)
//		w.Write([]byte(res.Data.(*model.FileBackendProfile).ToJson()))
//	}
var ReturnStatusOK = web.ReturnStatusOK
