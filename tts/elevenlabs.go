package tts

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
)

func Elevenlabs(params TTSParams) (io.ReadCloser, *string, *int, error) {
	token := string(fixKey(params.Key))
	voiceId := ""

	params.Voice = strings.ToUpper(params.Voice)
	switch params.Voice {
	case "MALE":
		voiceId = "J7snWfBtGKxBcPiNUoia"
	case "FEMALE":
		voiceId = "yoKoOZMjmn2OuceuTuQO"
	default:
		voiceId = "J7snWfBtGKxBcPiNUoia"
	}

	req := struct {
		ModelId string `json:"model_id"`
		Text    string `json:"text"`
	}{
		"eleven_multilingual_v2",
		params.Text,
	}

	jsonData, err := json.Marshal(&req)
	if err != nil {
		return nil, nil, nil, err
	}

	url := fmt.Sprintf("https://api.elevenlabs.io/v1/text-to-speech/%s/stream", voiceId)
	payload := strings.NewReader(string(jsonData))

	request, err := http.NewRequest("POST", url, payload)
	if err != nil {
		return nil, nil, nil, err
	}

	request.Header.Add("Content-Type", "application/json")
	request.Header.Add("xi-api-key", token)

	res, err := http.DefaultClient.Do(request)
	if err != nil {
		return nil, nil, nil, err
	}

	ct := "audio/mp3"

	if res.StatusCode != http.StatusOK {
		defer res.Body.Close()
		body, _ := ioutil.ReadAll(res.Body)

		return nil, nil, nil, errors.New("Bad response, error: " + string(body))
	}

	return res.Body, &ct, nil, nil
}
