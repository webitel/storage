package app

import (
	"encoding/json"
	"io"
	"strings"

	"github.com/webitel/wlog"

	engine "github.com/webitel/engine/model"
	"github.com/webitel/storage/model"

	tts2 "github.com/webitel/storage/tts"
)

const (
	TtsProfile    = ""
	TtsPoly       = "Polly"
	TtsMicrosoft  = "Microsoft"
	TtsGoogle     = "Google"
	TtsYandex     = "Yandex"
	TtsWebitel    = "Webitel"
	TtsElevenlabs = "Elevenlabs"
)

type ttsFunction func(tts2.TTSParams) (io.ReadCloser, *string, error)

var (
	ttsEngine = map[string]ttsFunction{
		strings.ToLower(TtsPoly):       tts2.Poly,
		strings.ToLower(TtsMicrosoft):  tts2.Microsoft,
		strings.ToLower(TtsGoogle):     tts2.Google,
		strings.ToLower(TtsYandex):     tts2.Yandex,
		strings.ToLower(TtsWebitel):    tts2.Webitel,
		strings.ToLower(TtsElevenlabs): tts2.Elevenlabs,
	}
)

func (a *App) TTS(provider string, params tts2.TTSParams) (out io.ReadCloser, t *string, err engine.AppError) {
	var ttsErr error

	if params.ProfileId > 0 && len(params.Key) == 0 {
		var ttsProfile *model.TtsProfile
		ttsProfile, err = a.Store.CognitiveProfile().SearchTtsProfile(int64(params.DomainId), params.ProfileId)
		if err != nil {

			return
		}

		if !ttsProfile.Enabled {
			err = engine.NewBadRequestError("tts.profile.disabled", "Profile is disabled")

			return
		}

		provider = ttsProfile.Provider

		if jErr := json.Unmarshal(ttsProfile.Properties, &params); jErr != nil {
			wlog.Error(jErr.Error())
		}

	}
	provider = strings.ToLower(provider)
	if fn, ok := ttsEngine[provider]; ok {
		out, t, ttsErr = fn(params)
		if ttsErr != nil {
			switch ttsErr.(type) {
			case engine.AppError:
				err = ttsErr.(engine.AppError)
			default:
				err = engine.NewInternalError("tts.app_error", ttsErr.Error())
			}
		}
	} else {
		return nil, nil, engine.NewNotFoundError("tts.valid.not_found", "Not found provider")
	}

	return
}
