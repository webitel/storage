package helper

import (
	"net/http"
	"strconv"
	"strings"

	tts2 "github.com/webitel/storage/tts"
)

func TtsParamsFromRequest(r *http.Request) tts2.TTSParams {
	var profileId int
	var domainId int
	var tmp string

	query := r.URL.Query()

	if query.Has("profile_id") {
		profileId, _ = strconv.Atoi(query.Get("profile_id"))
	}

	if query.Has("domain_id") {
		domainId, _ = strconv.Atoi(query.Get("domain_id"))
	}

	params := tts2.TTSParams{
		Id:        query.Get("id"),
		DomainId:  domainId,
		ProfileId: profileId,

		Key:      []byte(query.Get("key")),
		Token:    query.Get("token"),
		Format:   query.Get("format"),
		Voice:    query.Get("voice"),
		Region:   query.Get("region"),
		Text:     query.Get("text"),
		TextType: query.Get("text_type"),
		Language: query.Get("language"),
	}

	rate, _ := strconv.Atoi(query.Get("rate"))
	params.Rate = rate

	if tmp = query.Get("speakingRate"); tmp != "" {
		params.SpeakingRate, _ = strconv.ParseFloat(tmp, 32)
	}

	if tmp = query.Get("pitch"); tmp != "" {
		params.Pitch, _ = strconv.ParseFloat(tmp, 32)
	}

	if tmp = query.Get("volumeGainDb"); tmp != "" {
		params.VolumeGainDb, _ = strconv.ParseFloat(tmp, 32)
	}

	if tmp = query.Get("effectsProfileId"); tmp != "" {
		params.EffectsProfileId = strings.Split(tmp, ",")
	}

	if tmp = query.Get("bg_url"); tmp != "" {
		params.Background = &struct {
			FileUri string
			Volume  float64
			FadeIn  int64
			FadeOut int64
		}{FileUri: tmp, Volume: 0.7, FadeIn: 3000, FadeOut: 4000}

		if tmp = query.Get("bg_vol"); tmp != "" {
			params.Background.Volume, _ = strconv.ParseFloat(tmp, 32)
		}
		if tmp = query.Get("bg_fin"); tmp != "" {
			params.Background.FadeIn, _ = strconv.ParseInt(tmp, 10, 64)
		}
		if tmp = query.Get("bg_fout"); tmp != "" {
			params.Background.FadeOut, _ = strconv.ParseInt(tmp, 10, 64)
		}
	}
	if query.Get("bg") == "true" {
		params.BackgroundPlayback = true
	}

	params.KeyLocation = query.Get("keyLocation")
	params.VoiceSettings = query

	return params
}
