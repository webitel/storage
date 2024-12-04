package tts

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
)

type ElevenLabsVoiceSettings struct {
	Stability       float64 `json:"stability"`
	SimilarityBoost float64 `json:"similarity_boost"`
	Style           float64 `json:"style"`
	UseSpeakerBoost bool    `json:"use_speaker_boost,omitempty"`
}

type ElevenLabsRequest struct {
	ModelId       string                  `json:"model_id"`
	Text          string                  `json:"text"`
	VoiceSettings ElevenLabsVoiceSettings `json:"voice_settings"`
}

func ElevenLabs(params TTSParams) (io.ReadCloser, *string, *int, error) {
	token := string(fixKey(params.Key))
	voiceId := ""

	switch strings.ToUpper(params.Voice) {
	case "MALE":
		voiceId = "J7snWfBtGKxBcPiNUoia"
	case "FEMALE":
		voiceId = "yoKoOZMjmn2OuceuTuQO"
	default:
		if params.Voice == "" {
			voiceId = "J7snWfBtGKxBcPiNUoia"
		} else {
			voiceId = params.Voice
		}
	}

	req := ElevenLabsRequest{
		ModelId: "eleven_multilingual_v2",
		Text:    params.Text,
	}

	if params.VoiceSettings != nil {
		if params.VoiceSettings.Has("similarity_boost") {
			req.VoiceSettings.SimilarityBoost, _ = strconv.ParseFloat(params.VoiceSettings.Get("similarity_boost"), 32)
		}
		if params.VoiceSettings.Has("stability") {
			req.VoiceSettings.Stability, _ = strconv.ParseFloat(params.VoiceSettings.Get("stability"), 32)
		}
		if params.VoiceSettings.Has("style") {
			req.VoiceSettings.Style, _ = strconv.ParseFloat(params.VoiceSettings.Get("style"), 32)
		}
		if params.VoiceSettings.Has("use_speaker_boost") {
			req.VoiceSettings.UseSpeakerBoost = params.VoiceSettings.Get("use_speaker_boost") == "true"
		}
		if params.VoiceSettings.Has("model") {
			req.ModelId = params.VoiceSettings.Get("model")
		}
	}

	jsonData, err := json.Marshal(&req)
	if err != nil {
		return nil, nil, nil, err
	}

	url := fmt.Sprintf("https://api.elevenlabs.io/v1/text-to-speech/%s/stream?output_format=mp3_22050_32", voiceId)
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
