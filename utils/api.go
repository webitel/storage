package utils

import (
	"net/http"
	"strings"

	engine "github.com/webitel/engine/model"
	"github.com/webitel/storage/model"
)

func CheckOrigin(r *http.Request, allowedOrigins string) bool {
	origin := r.Header.Get("Origin")
	if allowedOrigins == "*" {
		return true
	}
	for _, allowed := range strings.Split(allowedOrigins, " ") {
		if allowed == origin {
			return true
		}
	}
	return false
}

func OriginChecker(allowedOrigins string) func(*http.Request) bool {
	return func(r *http.Request) bool {
		return CheckOrigin(r, allowedOrigins)
	}
}

func RenderWebAppError(config *model.Config, w http.ResponseWriter, r *http.Request, err engine.AppError) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(err.GetStatusCode())
	w.Write([]byte(err.ToJson()))
}
