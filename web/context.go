package web

import (
	"net/http"

	"github.com/webitel/engine/pkg/wbt/auth_manager"
	"github.com/webitel/storage/app"
	"github.com/webitel/storage/controller"
	"github.com/webitel/storage/model"
	"github.com/webitel/wlog"
)

type Context struct {
	App           *app.App
	Log           *wlog.Logger
	Session       auth_manager.Session
	Err           model.AppError
	Params        *Params
	Ctrl          *controller.Controller
	RequestId     string
	IpAddress     string
	Path          string
	siteURLHeader string
}

func (c *Context) LogError(err model.AppError) {
	// Filter out 404s, endless reconnects and browser compatibility errors
	if err.GetStatusCode() == http.StatusNotFound {
		c.LogDebug(err)
	} else {
		c.Log.Error(
			err.Error(),
			wlog.Int("http_code", err.GetStatusCode()),
			wlog.String("err_details", err.GetDetailedError()),
		)
	}
}

func (c *Context) LogInfo(err model.AppError) {
	// Filter out 401s
	if err.GetStatusCode() == http.StatusUnauthorized {
		c.LogDebug(err)
	} else {
		c.Log.Info(
			err.Error(),
			wlog.Int("http_code", err.GetStatusCode()),
			wlog.String("err_details", err.GetDetailedError()),
		)
	}
}

func (c *Context) LogDebug(err model.AppError) {
	c.Log.Debug(
		err.Error(),
		wlog.Int("http_code", err.GetStatusCode()),
		wlog.String("err_details", err.GetDetailedError()),
	)
}

func (c *Context) SessionRequired() {
	if c.Session.UserId == 0 {
		c.Err = model.NewInternalError("api.context.session_expired.app_error", "UserRequired")
		return
	}
}

func (c *Context) SetInvalidParam(parameter string) {
	c.Err = NewInvalidParamError(parameter)
}

func (c *Context) SetInvalidUrlParam(parameter string) {
	c.Err = NewInvalidUrlParamError(parameter)
}

func NewInvalidParamError(parameter string) model.AppError {
	err := model.NewBadRequestError("api.context.invalid_body_param.app_error", "").SetTranslationParams(map[string]interface{}{"Name": parameter})
	return err
}

func NewInvalidUrlParamError(parameter string) model.AppError {
	err := model.NewBadRequestError("api.context.invalid_url_param.app_error", "").SetTranslationParams(map[string]interface{}{"Name": parameter})
	return err
}

func (c *Context) RequireId() *Context {
	if c.Err != nil {
		return c
	}

	if len(c.Params.Id) == 0 {
		c.SetInvalidUrlParam("id")
	}
	return c
}

func (c *Context) RequireDomain() *Context {
	if c.Err != nil {
		return c
	}

	if len(c.Params.Domain) == 0 {
		c.SetInvalidUrlParam("domain_id")
	}

	return c
}

func (c *Context) RequireExpire() *Context {
	if c.Err != nil {
		return c
	}

	if c.Params.Expires < 1 {
		c.SetInvalidUrlParam("expires")
	}
	return c
}

func (c *Context) SetSessionExpire() {
	c.Err = model.NewInternalError("api.context.session_expired.app_error", "")
}

func (c *Context) SetSessionErrSignature() {
	c.Err = model.NewInternalError("api.context.session_signature.app_error", "")
}

func (c *Context) RequireSignature() *Context {
	if c.Err != nil {
		return c
	}

	if len(c.Params.Signature) == 0 {
		c.SetInvalidUrlParam("signature")
	}
	return c
}
