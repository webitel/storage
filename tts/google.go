package tts

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"io/ioutil"
	"strings"

	texttospeech "cloud.google.com/go/texttospeech/apiv1"
	"google.golang.org/api/option"
	texttospeechpb "google.golang.org/genproto/googleapis/cloud/texttospeech/v1"
)

func Google(params TTSParams) (io.ReadCloser, *string, *int, error) {
	// Instantiates a client.
	ctx := context.Background()
	var err error
	var client *texttospeech.Client

	options := make([]option.ClientOption, 0, 1)

	if len(params.Key) != 0 {
		options = append(options, option.WithCredentialsJSON(params.Key))
	} else if params.KeyLocation != "" {
		options = append(options, option.WithCredentialsFile(params.KeyLocation))
	}

	client, err = texttospeech.NewClient(ctx, options...)

	if err != nil {
		return nil, nil, nil, err
	}

	// Perform the text-to-speech request on the text input with the selected
	// voice parameters and audio file type.
	req := texttospeechpb.SynthesizeSpeechRequest{
		// Build the voice request, select the language code ("en-US") and the SSML
		// voice gender ("neutral").
		Voice: &texttospeechpb.VoiceSelectionParams{
			LanguageCode: params.Language,
		},
		// Select the type of audio file you want returned.
		AudioConfig: &texttospeechpb.AudioConfig{
			AudioEncoding: texttospeechpb.AudioEncoding_LINEAR16,
			SpeakingRate:  1,
			Pitch:         1,
			//VolumeGainDb:     0,
			SampleRateHertz: 8000,
			//EffectsProfileId: nil,
		},
	}

	if params.SpeakingRate != 0 {
		req.AudioConfig.SpeakingRate = params.SpeakingRate
	}

	if params.Pitch != 0 {
		req.AudioConfig.Pitch = params.Pitch
	}

	if params.VolumeGainDb != 0 {
		req.AudioConfig.VolumeGainDb = params.VolumeGainDb
	}

	if params.EffectsProfileId != nil {
		req.AudioConfig.EffectsProfileId = params.EffectsProfileId
	}
	params.Voice = strings.ToUpper(params.Voice)
	switch params.Voice {
	case "MALE":
		req.Voice.SsmlGender = texttospeechpb.SsmlVoiceGender_MALE
	case "FEMALE":
		req.Voice.SsmlGender = texttospeechpb.SsmlVoiceGender_FEMALE
	case "NEUTRAL":
		req.Voice.SsmlGender = texttospeechpb.SsmlVoiceGender_NEUTRAL
	default:
		req.Voice.Name = params.Voice
	}

	v := "audio/ogg"
	if params.Format == "mp3" {
		v = "audio/mp3"
		req.AudioConfig.SampleRateHertz = 22050
		req.AudioConfig.AudioEncoding = texttospeechpb.AudioEncoding_MP3
	}

	// Set the text input to be synthesized.
	if params.TextType == "ssml" {
		req.Input = &texttospeechpb.SynthesisInput{
			InputSource: &texttospeechpb.SynthesisInput_Ssml{Ssml: params.Text},
		}
	} else {
		req.Input = &texttospeechpb.SynthesisInput{
			InputSource: &texttospeechpb.SynthesisInput_Text{Text: params.Text},
		}
	}

	resp, err := client.SynthesizeSpeech(ctx, &req)
	if err != nil {
		return nil, nil, nil, err
	}

	r := ioutil.NopCloser(bytes.NewReader(resp.GetAudioContent()))
	client.Close() // FIXME

	size := len(resp.GetAudioContent())

	return r, &v, &size, nil
}

func GoogleVoice(params TTSVoiceParams) (*string, error) {
	voices := []Voice{
		{VoiceID: "FEMALE", Name: "FEMALE"},
		{VoiceID: "MALE", Name: "MALE"},
		{VoiceID: "NEUTRAL", Name: "NEUTRAL"},
	}
	voicesJSON, err := json.Marshal(voices)
	if err != nil {
		return nil, err
	}
	var result = (string(voicesJSON))

	return &result, nil
}