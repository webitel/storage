package tts

import (
	"io"
	"net/http"
	"net/url"
	"strings"
)

const (
	ttsWebitelResource = "/synthesize/audio"
)

var (
	wbtTTSEndpoint = "https://f500-176-39-35-14.ngrok-free.app"
)

func SetWbtTTSEndpoint(endpoint string) {
	wbtTTSEndpoint = endpoint
}

func Webitel(req TTSParams) (io.ReadCloser, *string, error) {
	req.Text = strings.TrimSpace(req.Text)
	l := len(req.Text)

	if req.Text[l-1:l] != "." {
		req.Text = req.Text + "."
	}

	data := url.Values{}
	data.Set("text", req.Text)
	data.Set("speaker_id", "0")

	u, _ := url.ParseRequestURI(wbtTTSEndpoint)
	u.Path = ttsWebitelResource
	urlStr := u.String()

	client := &http.Client{}

	r, err := http.NewRequest(http.MethodPost, urlStr, strings.NewReader(strings.Replace(data.Encode(), "+", "%20", -1))) // URL-encoded payload
	if err != nil {
		return nil, nil, err
	}
	r.Header.Add("Accept", "text/plain")
	r.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	result, err := client.Do(r)
	if err != nil {
		return nil, nil, err
	}

	contentType := result.Header.Get("Content-Type")

	if contentType == "" {
		contentType = "audio/wav"
	}

	return result.Body, &contentType, nil
}
