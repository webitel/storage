package tts

import (
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

	url := fmt.Sprintf("https://api.elevenlabs.io/v1/text-to-speech/%s/stream", voiceId)
	payload := strings.NewReader(fmt.Sprintf("{\n  \"model_id\": \"eleven_multilingual_v2\",\n  \"text\": \"%s\",\n  \"voice_settings\": {\n    \"similarity_boost\": 0.75,\n    \"stability\": 0.5,\n    \"style\": 0.5,\n    \"use_speaker_boost\": true\n  }\n}", params.Text))

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
