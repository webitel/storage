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
		DomainId:  domainId,
		ProfileId: profileId,

		Key:      query.Get("key"),
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

	params.KeyLocation = query.Get("keyLocation")

	return params
}
