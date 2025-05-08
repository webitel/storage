package web

import (
	"net/http"

	"github.com/webitel/storage/app"
	"github.com/webitel/storage/model"
)

func Handle404(a *app.App, w http.ResponseWriter, r *http.Request) {
	err := model.NewNotFoundError("api.context.404.app_error", "")

	w.WriteHeader(err.GetStatusCode())
	err.SetDetailedError("There doesn't appear to be an api call for the url='" + r.URL.Path + "'.  Typo? are you missing a team_id or user_id as part of the url?")
	w.Write([]byte(err.ToJson()))
}

func ReturnStatusOK(w http.ResponseWriter) {
	m := make(map[string]string)
	m[model.STATUS] = model.STATUS_OK
	w.Write([]byte(model.MapToJson(m)))
}
