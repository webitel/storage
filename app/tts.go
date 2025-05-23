package app

import (
	"encoding/json"
	"io"
	"strings"

	"github.com/webitel/storage/model"
	tts2 "github.com/webitel/storage/tts"
	"github.com/webitel/wlog"
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

type ttsFunction func(tts2.TTSParams) (io.ReadCloser, *string, *int, error)

var (
	ttsEngine = map[string]ttsFunction{
		strings.ToLower(TtsPoly):       tts2.Poly,
		strings.ToLower(TtsMicrosoft):  tts2.Microsoft,
		strings.ToLower(TtsGoogle):     tts2.Google,
		strings.ToLower(TtsYandex):     tts2.Yandex,
		strings.ToLower(TtsWebitel):    tts2.Webitel,
		strings.ToLower(TtsElevenLabs): tts2.ElevenLabs,
	}
)

func (a *App) TTS(provider string, params tts2.TTSParams) (out io.ReadCloser, t *string, size *int, err model.AppError) {
	var ttsErr error

	if params.ProfileId > 0 && len(params.Key) == 0 {
		var ttsProfile *model.TtsProfile
		ttsProfile, err = a.Store.CognitiveProfile().SearchTtsProfile(int64(params.DomainId), params.ProfileId)
		if err != nil {

			return
		}

		if !ttsProfile.Enabled {
			err = model.NewBadRequestError("tts.profile.disabled", "Profile is disabled")

			return
		}

		provider = ttsProfile.Provider

		if jErr := json.Unmarshal(ttsProfile.Properties, &params); jErr != nil {
			wlog.Error(jErr.Error())
		}

	}
	provider = strings.ToLower(provider)
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
