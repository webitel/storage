package web

import (
	"fmt"
	"net/http"

	"github.com/webitel/storage/app"
	"github.com/webitel/storage/controller"
	"github.com/webitel/storage/model"
	"github.com/webitel/storage/utils"
	"github.com/webitel/wlog"
)

type Handler struct {
	App            *app.App
	Ctrl           *controller.Controller
	HandleFunc     func(*Context, http.ResponseWriter, *http.Request)
	RequireSession bool
	TrustRequester bool
	RequireMfa     bool
	IsStatic       bool
}

func (h Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	wlog.Debug(fmt.Sprintf("%v - %v", r.Method, r.URL.Path))

	c := &Context{}
	c.App = h.App
	c.Ctrl = h.Ctrl
	c.Params = ParamsFromRequest(r)
	c.RequestId = model.NewId()
	c.IpAddress = utils.GetIpAddress(r)
	c.Path = r.URL.Path
	c.Log = c.App.Log

	w.Header().Set(model.HEADER_REQUEST_ID, c.RequestId)
	w.Header().Set("Content-Type", "application/json")
	if r.Method == "GET" {
		w.Header().Set("Expires", "0")
	}

	//TODO
	token, _ := app.ParseAuthTokenFromRequest(r)
	if len(token) != 0 && h.RequireSession {
		session, err := c.App.GetSession(token)
		if err != nil {
			c.Log.Info("Invalid session", wlog.Err(err))
			if err.GetStatusCode() == http.StatusInternalServerError {
				c.Err = err
			} else {
				c.Err = model.NewInternalError("api.context.session_expired.app_error", "token="+token)
			}
		} else {
			c.Session = *session
		}
	}

	c.Log = c.App.Log.With(
		wlog.String("path", c.Path),
		wlog.String("request_id", c.RequestId),
		wlog.String("ip_addr", c.IpAddress),
		wlog.Int64("user_id", c.Session.UserId),
		wlog.String("method", r.Method),
	)

	if c.Err == nil && h.RequireSession {
		c.SessionRequired()
	}

	if c.Err == nil {
		h.HandleFunc(c, w, r)
	}

	// Handle errors that have occurred
	if c.Err != nil {
		c.Err.SetRequestId(c.RequestId)

		if c.Err.GetId() == "api.context.session_expired.app_error" {
			c.LogInfo(c.Err)
		} else {
			c.LogError(c.Err)
		}

		w.WriteHeader(c.Err.GetStatusCode())
		w.Write([]byte(c.Err.ToJson()))
	}
}
