package app

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/webitel/storage/model"

	tts2 "github.com/webitel/storage/tts"
)

const (
	TtsProfile   = ""
	TtsPoly      = "Poly"
	TtsMicrosoft = "Microsoft"
	TtsGoogle    = "Google"
	TtsYandex    = "Yandex"
)

type ttsFunction func(tts2.TTSParams) (io.ReadCloser, *string, error)

var (
	ttsEngine = map[string]ttsFunction{
		TtsPoly:      tts2.Poly,
		TtsMicrosoft: tts2.Microsoft,
		TtsGoogle:    tts2.Google,
		TtsYandex:    tts2.Yandex,
	}
)

func (a *App) TTS(provider string, params tts2.TTSParams) (out io.ReadCloser, t *string, err *model.AppError) {
	var ttsErr error
	provider = strings.ToLower(provider)

	if params.ProfileId > 0 && params.Key == "" {
		var ttsProfile *model.TtsProfile
		ttsProfile, err = a.Store.CognitiveProfile().SearchTtsProfile(int64(params.DomainId), params.ProfileId)
		if err != nil {

			return
		}

		if !ttsProfile.Enabled {
			err = model.NewAppError("TTS", "tts.profile.disabled", nil, "Profile is disabled", http.StatusBadRequest)

			return
		}

		provider = ttsProfile.Provider

		json.Unmarshal(ttsProfile.Properties, &params)
	}

	if fn, ok := ttsEngine[provider]; ok {
		out, t, ttsErr = fn(params)
		if ttsErr != nil {
			err = model.NewAppError("TTS", "tts.app_error", nil, ttsErr.Error(), http.StatusInternalServerError)
		}
	} else {
		return nil, nil, model.NewAppError("TTS", "tts.valid.not_found", nil, "Not found provider", http.StatusNotFound)
	}

	return
}
