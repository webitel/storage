package web

import (
	"net/http"

	"github.com/webitel/webitel-go-kit/infra/otel/correlation"
	"github.com/webitel/webitel-go-kit/pkg/logging"

	"github.com/webitel/storage/app"
	"github.com/webitel/storage/controller"
	"github.com/webitel/storage/model"
	"github.com/webitel/storage/utils"
)

const httpLogComponent logging.Component = "storage.http"

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
	c := &Context{}
	c.App = h.App
	c.Ctrl = h.Ctrl
	c.Params = ParamsFromRequest(r)
	c.RequestId = model.NewId()
	c.IpAddress = utils.GetIpAddress(r)
	c.Path = r.URL.Path
	c.Ctx = r.Context()
	c.Log = c.App.Logger().WithComponent(httpLogComponent).With(
		logging.Str("path", c.Path),
		logging.Str("request_id", c.RequestId),
		logging.Str("ip_addr", c.IpAddress),
		logging.Str("method", r.Method),
	)

	c.Log.DebugCtx(c.Ctx, "request")

	w.Header().Set(model.HEADER_REQUEST_ID, c.RequestId)
	w.Header().Set("Content-Type", "application/json")
	if r.Method == "GET" {
		w.Header().Set("Expires", "0")
	}

	//TODO
	token, _ := app.ParseAuthTokenFromRequest(r)
	if len(token) != 0 && h.RequireSession {
		session, err := c.App.GetSession(c.Ctx, token)
		if err != nil {
			c.Log.InfoCtx(c.Ctx, "invalid session", logging.Err(err))
			if err.GetStatusCode() == http.StatusInternalServerError {
				c.Err = err
			} else {
				c.Err = model.NewInternalError("api.context.session_expired.app_error", "token="+token)
			}
		} else {
			c.Session = *session
			c.Ctx = correlation.WithIdentity(c.Ctx, session.DomainId, session.UserId)
			r = r.WithContext(c.Ctx)
		}
	}

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
