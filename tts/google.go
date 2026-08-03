package tts

import (
	"bytes"
	"context"
	"io"
	"io/ioutil"
	"strings"

	texttospeech "cloud.google.com/go/texttospeech/apiv1"
	texttospeechpb "cloud.google.com/go/texttospeech/apiv1/texttospeechpb"
	"google.golang.org/api/option"
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
			ModelName:    strings.TrimSpace(params.VoiceSettings.Get("model")),
		},
		// Select the type of audio file you want returned.
		AudioConfig: &texttospeechpb.AudioConfig{
			AudioEncoding: texttospeechpb.AudioEncoding_LINEAR16,
			// SpeakingRate:  1,
			// Pitch:         1,
			// VolumeGainDb:     0,
			SampleRateHertz: 8000,
			// EffectsProfileId: nil,
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
	if req.Voice.ModelName != "" {
		// Gemini-TTS models require a Gemini prebuilt voice name (e.g. "Kore",
		// "Puck"), which is case-sensitive, and reject SsmlGender. Pass the voice
		// through verbatim without upper-casing or gender mapping.
		req.Voice.Name = strings.TrimSpace(params.Voice)
	} else {
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

	prompt := strings.TrimSpace(params.VoiceSettings.Get("prompt"))

	// Prompt (style instructions) is only honored by Gemini-TTS models
	// (set via VoiceSelectionParams.ModelName), e.g. "gemini-2.5-flash-tts".
	if prompt != "" {
		req.Input.Prompt = &prompt
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
