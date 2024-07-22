package tts

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
)

type Voice struct {
	VoiceID  string `json:"voice_id"`
	Name     string `json:"name"`
	Category string `json:"category"`
}

type VoiceFormat struct {
	VoiceID  string `json:"voice_id"`
	Name     string `json:"name"`
}

type Response struct {
	Voices []Voice `json:"voices"`
}

func ElevenLabs(params TTSParams) (io.ReadCloser, *string, *int, error) {
	token := string(fixKey(params.Key))

	reqBody := struct {
		ModelId string `json:"model_id"`
		Text    string `json:"text"`
	}{
		"eleven_multilingual_v2",
		params.Text,
	}

	jsonData, err := json.Marshal(&reqBody)
	if err != nil {
		return nil, nil, nil, err
	}

	url := fmt.Sprintf("https://api.elevenlabs.io/v1/text-to-speech/%s/stream", params.Voice)
	payload := bytes.NewReader(jsonData)

	request, err := http.NewRequest("POST", url, payload)
	if err != nil {
		return nil, nil, nil, err
	}

	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("xi-api-key", token)

	res, err := http.DefaultClient.Do(request)
	if err != nil {
		return nil, nil, nil, err
	}

	if res.StatusCode != http.StatusOK {
		defer res.Body.Close()
		body, _ := ioutil.ReadAll(res.Body)
		return nil, nil, nil, errors.New("Bad response, error: " + string(body))
	}

	contentType := "audio/mp3"
	return res.Body, &contentType, nil, nil
}


func ElevenLabsVoice(params TTSVoiceParams) (*string, error) {
	token := string(fixKey(params.Key))
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}

	var url string
	if params.Q != "" {
		url = fmt.Sprintf("https://api.elevenlabs.io/v1/shared-voices?search=%s", params.Q)
	} else {
		url = "https://api.elevenlabs.io/v1/voices"
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Add("xi-api-key", token)

	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	var response Response
	err = json.Unmarshal(body, &response)
	if err != nil {
		return nil, err
	}

	var filteredVoices []VoiceFormat
	for _, voice := range response.Voices {
		if params.Q != "" || voice.Category == "generated" {
			filteredVoices = append(filteredVoices, VoiceFormat{
				VoiceID: voice.VoiceID,
				Name:    voice.Name,
			})
		}
	}

	filteredVoicesJSON, err := json.Marshal(filteredVoices)
	if err != nil {
		return nil, err
	}

	result := string(filteredVoicesJSON)
	return &result, nil
}
