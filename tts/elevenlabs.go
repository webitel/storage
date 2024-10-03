package tts

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"

	engine "github.com/webitel/engine/model"
	"github.com/webitel/storage/model"
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

type Voice struct {
	VoiceID  string `json:"voice_id"`
	Name     string `json:"name"`
	Category string `json:"category"`
}

type Response struct {
	Voices []Voice `json:"voices"`
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

func ElevenLabsVoice(domainId int64, req *model.SearchCognitiveProfileVoice) ([]*model.CognitiveProfileVoice, engine.AppError) {
	token := req.Key
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}
	var url string
	if req.Q != "" {
		url = fmt.Sprintf("https://api.elevenlabs.io/v1/shared-voices?search=%s", req.Q)
	} else {
		url = "https://api.elevenlabs.io/v1/voices"
	}

	resp, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, engine.NewCustomCodeError("store.cognitive_profile_store.search_voice.app_error", err.Error(), http.StatusInternalServerError)
	}

	resp.Header.Add("xi-api-key", token)

	res, err := client.Do(resp)
	if err != nil {
		return nil, engine.NewCustomCodeError("store.cognitive_profile_store.search_voice.app_error", err.Error(), http.StatusInternalServerError)
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, engine.NewCustomCodeError("store.cognitive_profile_store.search_voice.app_error", err.Error(), http.StatusInternalServerError)
	}

	var response Response
	err = json.Unmarshal(body, &response)
	if err != nil {
		return nil, engine.NewCustomCodeError("store.cognitive_profile_store.search_voice.app_error", err.Error(), http.StatusInternalServerError)
	}

	var filteredVoices []*model.CognitiveProfileVoice
	for _, voice := range response.Voices {
		if req.Q != "" || voice.Category == "generated" {
			filteredVoices = append(filteredVoices, &model.CognitiveProfileVoice{
				Id:   voice.VoiceID,
				Name: voice.Name,
			})
		}
	}

	return filteredVoices, nil
}
