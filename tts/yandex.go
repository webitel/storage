package tts

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
)

func Yandex(params TTSParams) (io.ReadCloser, *string, *int, error) {
	api := fmt.Sprintf("https://tts.api.cloud.yandex.net/speech/v1/tts:synthesize?lang=%s", url.QueryEscape(params.Language))

	if params.Voice != "" {
		api += "&voice=" + params.Voice
	}

	if params.TextType == "ssml" {
		api += "&ssml=" + url.QueryEscape(params.Text)
	} else {
		api += "&text=" + url.QueryEscape(params.Text)
	}

	request, err := http.NewRequest("POST", api, nil)
	request.Header.Add("Authorization", fmt.Sprintf("Api-Key %s", params.Token))

	result, err := http.DefaultClient.Do(request)
	if err != nil {
		return nil, nil, nil, err
	}

	return result.Body, nil, nil, nil
}
