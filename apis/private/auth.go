package private

import (
	"errors"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/webitel/wlog"

	"github.com/webitel/storage/model"
	"github.com/webitel/storage/pkg/sysurl"
	"github.com/webitel/storage/utils"
)

func (api *API) sysAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mode := api.App.Config().ServiceSettings.SysAuthMode
		if mode != model.SysAuthModeOff {
			route := sysRoute(r)
			if err := api.App.SysVerifier().Verify(route, r); err != nil {
				api.logSysAuth(route, r, err)
				if mode == model.SysAuthModeEnforce {
					writeSysAuthError(w, err)
					return
				}
			}
		}
		sysurl.Strip(r)
		next.ServeHTTP(w, r)
	})
}

func sysRoute(r *http.Request) sysurl.Route {
	if current := mux.CurrentRoute(r); current != nil {
		return sysurl.Route(current.GetName())
	}
	return ""
}

func (api *API) logSysAuth(route sysurl.Route, r *http.Request, err error) {
	log := api.App.Log.With(
		wlog.String("route", string(route)),
		wlog.String("path", r.URL.Path),
		wlog.String("method", r.Method),
		wlog.String("ip_addr", utils.GetIpAddress(r)),
		wlog.Err(err),
	)
	if errors.Is(err, sysurl.ErrMissingSignature) {
		log.Debug("unsigned /sys request")
		return
	}
	log.Warn("invalid /sys request signature")
}

func writeSysAuthError(w http.ResponseWriter, err error) {
	appErr := model.NewForbiddenError("api.sys_auth.invalid_signature.app_error", "invalid signature")
	if errors.Is(err, sysurl.ErrMissingSignature) {
		appErr = model.NewUnauthorizedError("api.sys_auth.missing_signature.app_error", "signature required")
	}
	appErr.SetRequestId(model.NewId())
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(appErr.GetStatusCode())
	w.Write([]byte(appErr.ToJson()))
}
