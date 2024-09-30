package tts

import (
	"encoding/json"
	"fmt"
	"io"
	"net/url"
)

type TTSEngine interface {
	GetStream(TTSParams) (io.ReadCloser, string, error)
}

type TTSParams struct {
	DomainId    int             `json:"-"`
	ProfileId   int             `json:"-"`
	Key         json.RawMessage `json:"key"`
	Token       string          `json:"token"`
	KeyLocation string          `json:"key_location"`
	Region      string          `json:"region"`
	Locale      string          `json:"locale"`

	Format         string `json:"-"`
	Voice          string `json:"-"`
	Language       string `json:"-"`
	Text, TextType string `json:"-"`

	Rate       int `json:"-"`
	Background *struct {
		FileUri string
		Volume  float64
		FadeIn  int64
		FadeOut int64
	}
	//google
	SpeakingRate     float64  `json:"-"`
	Pitch            float64  `json:"-"`
	VolumeGainDb     float64  `json:"-"`
	EffectsProfileId []string `json:"-"`

	VoiceSettings url.Values
}

func (p TTSParams) BackgroundNode() string {
	if p.Background != nil {
		return fmt.Sprintf(`<mstts:backgroundaudio src="%s" volume="%f" fadein="%d" fadeout="%d"/>`,
			p.Background.FileUri, p.Background.Volume, p.Background.FadeIn, p.Background.FadeOut)
	}

	return ""
}
