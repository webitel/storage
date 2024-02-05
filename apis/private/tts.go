package private

import (
	"io"
	"net/http"
	"strconv"

	. "github.com/webitel/storage/apis/helper"
	"github.com/webitel/storage/app"
)

func (api *API) InitTTS() {
	api.Routes.TTS.Handle("/{id}", api.ApiHandler(ttsByProvider)).Methods("GET")
	api.Routes.TTS.Handle("/", api.ApiHandler(ttsByProfile)).Methods("GET")
}

func ttsByProfile(c *Context, w http.ResponseWriter, r *http.Request) {
	params := TtsParamsFromRequest(r)

	out, t, size, err := c.App.TTS(app.TtsProfile, params)
	if err != nil {
		c.Err = err
		return
	}

	defer out.Close()

	if t != nil {
		w.Header().Set("Content-Type", *t)
	}
	if size != nil {
		w.Header().Set("Content-Length", strconv.Itoa(*size))
	}
	io.Copy(w, out)
}

func ttsByProvider(c *Context, w http.ResponseWriter, r *http.Request) {
	params := TtsParamsFromRequest(r)

	out, t, size, err := c.App.TTS(c.Params.Id, params)
	if err != nil {
		c.Err = err
		return
	}

	defer out.Close()

	if t != nil {
		w.Header().Set("Content-Type", *t)
	}
	if size != nil {
		w.Header().Set("Content-Length", strconv.Itoa(*size))
	}
	io.Copy(w, out)
}
