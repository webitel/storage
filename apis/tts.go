package apis

import (
	"io"
	"net/http"
	"strconv"

	"github.com/webitel/storage/apis/helper"
	"github.com/webitel/storage/app"
)

func (api *API) InitTts() {
	api.PublicRoutes.Tts.Handle("/stream", api.ApiSessionRequired(tts)).Methods("GET")
}

func tts(c *Context, w http.ResponseWriter, r *http.Request) {
	params := helper.TtsParamsFromRequest(r)

	if params.DomainId == 0 {
		params.DomainId = int(c.Session.DomainId)
	}
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
