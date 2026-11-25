package app

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/webitel/storage/model"
	"github.com/webitel/storage/tts"
)

const (
	TtsProfile    = ""
	TtsPoly       = "Polly"
	TtsMicrosoft  = "Microsoft"
	TtsGoogle     = "Google"
	TtsYandex     = "Yandex"
	TtsWebitel    = "Webitel"
	TtsElevenLabs = "ElevenLabs"
)

type ttsFunction func(tts.TTSParams) (io.ReadCloser, *string, *int, error)

var (
	ttsEngine = map[string]ttsFunction{
		strings.ToLower(TtsPoly):       tts.Poly,
		strings.ToLower(TtsMicrosoft):  tts.Microsoft,
		strings.ToLower(TtsGoogle):     tts.Google,
		strings.ToLower(TtsYandex):     tts.Yandex,
		strings.ToLower(TtsWebitel):    tts.Webitel,
		strings.ToLower(TtsElevenLabs): tts.ElevenLabs,
	}
)

func (a *App) TTS(provider string, params tts.TTSParams) (out io.ReadCloser, t *string, size *int, err model.AppError) {
	var ttsErr error

	provider, err = a.applyTtsProfile(strings.ToLower(provider), &params)
	if err != nil {
		return
	}

	if params.Region != "" && !model.IsValidCloudRegion(params.Region) {
		return nil, nil, nil, model.NewBadRequestError("tts.profile.region.invalid", "Invalid region")
	}

	if fn, ok := ttsEngine[provider]; ok {
		out, t, size, ttsErr = fn(params)
		if ttsErr != nil {
			switch ttsErr.(type) {
			case model.AppError:
				err = ttsErr.(model.AppError)
			default:
				err = model.NewInternalError("tts.app_error", ttsErr.Error())
			}
		}
	} else {
		return nil, nil, nil, model.NewNotFoundError("tts.valid.not_found", "Not found provider")
	}

	return
}

func (a *App) applyTtsProfile(provider string, params *tts.TTSParams) (string, model.AppError) {
	byId := params.ProfileId > 0
	optional := !byId && provider != TtsProfile

	if params.DomainId == 0 {
		if optional {
			return provider, nil
		}
		return "", model.NewBadRequestError("tts.profile.domain.required", "domain_id is required")
	}

	profile, err := a.Store.CognitiveProfile().SearchTtsProfile(int64(params.DomainId), params.ProfileId)
	if err != nil {
		if optional && err.GetStatusCode() == http.StatusNotFound {
			return provider, nil
		}
		return "", err
	}

	profileProvider := strings.ToLower(profile.Provider)
	if optional && profileProvider != provider {
		return provider, nil
	}

	if !profile.Enabled {
		if optional {
			return provider, nil
		}
		return "", model.NewBadRequestError("tts.profile.disabled", "Profile is disabled")
	}

	type ttsProfileProperties struct {
		Key    json.RawMessage `json:"key"`
		Token  string          `json:"token"`
		Region string          `json:"region"`
		Locale string          `json:"locale"`
	}

	var props ttsProfileProperties
	if len(profile.Properties) != 0 {
		if jErr := json.Unmarshal(profile.Properties, &props); jErr != nil {
			return "", model.NewInternalError("tts.profile.properties.app_error", jErr.Error())
		}
	}

	params.Key = props.Key
	params.Token = props.Token
	params.Region = props.Region
	params.Locale = props.Locale

	return profileProvider, nil
}
