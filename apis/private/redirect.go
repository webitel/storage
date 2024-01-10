package private

import (
	"bytes"
	"github.com/h2non/filetype"
	"io"
	"net/http"
)

func (api *API) InitRedirect() {
	api.Routes.Redirect.Handle("/playback", api.ApiHandler(redirectPlayback)).Methods("GET")
}

func redirectPlayback(ctx *Context, w http.ResponseWriter, r *http.Request) {
	var (
		url, method string
		headers     map[string]string
		req         *http.Request
		res         *http.Response
		err         error
		body        bytes.Buffer
	)
	params := r.URL.Query()
	headers = make(map[string]string)
	for key, value := range params {
		var firstValue string
		if len(value) != 0 {
			firstValue = value[0]
		} else {
			continue
		}
		switch key {
		case "url":
			url = firstValue
		case "method":
			method = firstValue
		case ".mp3", ".wav":
			continue
		default:
			headers[key] = firstValue
		}
	}

	switch method {
	case http.MethodGet, "":
		req, err = http.NewRequest(http.MethodGet, url, nil)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}
	default:
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	for headerKey, headerValue := range headers {
		req.Header.Set(headerKey, headerValue)
	}
	res, err = http.DefaultClient.Do(req)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}
	_, err = io.Copy(&body, res.Body)
	if err != nil {
		return
	}
	if !filetype.IsAudio(body.Bytes()) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("file is not audio"))
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write(body.Bytes())
	return
}
