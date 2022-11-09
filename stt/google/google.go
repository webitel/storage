package google

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"time"

	"github.com/webitel/storage/model"

	"cloud.google.com/go/storage"

	speech "cloud.google.com/go/speech/apiv1"
	"cloud.google.com/go/speech/apiv1/speechpb"
)

const (
	ClientName = "Google"
)

type Stt struct {
	bucket     string
	speechCli  *speech.Client
	storageCli *storage.Client
}

func New(conf Config) (*Stt, error) {
	var err error
	ctx := context.Background()
	c := &Stt{
		bucket:     conf.Bucket,
		speechCli:  nil,
		storageCli: nil,
	}

	c.speechCli, err = speech.NewClient(ctx)
	if err != nil {
		return nil, err
	}

	c.storageCli, err = storage.NewClient(ctx)
	if err != nil {
		return nil, err
	}

	return c, nil
}

func (g *Stt) Transcript(ctx context.Context, id int64, fileUri, locale string) (model.FileTranscript, error) {
	var stdout io.ReadCloser
	var err error
	var op *speech.LongRunningRecognizeOperation
	var resp *speechpb.LongRunningRecognizeResponse

	cmd := exec.Command("ffmpeg", "-y", // Yes to all
		//"-hide_banner", "-loglevel", "panic", // Hide all logs
		"-i", fileUri,
		"-f", "s16le",
		"-ar", "16000",
		"-acodec", "pcm_s16le",
		"pipe:1", // output to stdout
	)

	stdout, err = cmd.StdoutPipe()
	if err != nil {
		return model.FileTranscript{}, err
	}
	err = cmd.Start()
	if err != nil {
		return model.FileTranscript{}, err
	}

	fileName := fmt.Sprintf("%d_stt.pcm", id)
	var gcsURI string

	gcsURI, err = g.upload(ctx, stdout, fileName)
	if err != nil {
		return model.FileTranscript{}, err
	}

	req := &speechpb.LongRunningRecognizeRequest{
		Config: &speechpb.RecognitionConfig{
			Encoding:                            speechpb.RecognitionConfig_LINEAR16,
			SampleRateHertz:                     16000,
			EnableSeparateRecognitionPerChannel: true,
			LanguageCode:                        locale,
			AlternativeLanguageCodes:            nil,
			MaxAlternatives:                     1,
			ProfanityFilter:                     false,
			Adaptation:                          nil,
			SpeechContexts:                      nil,
			EnableWordTimeOffsets:               true,
			EnableWordConfidence:                true,
			EnableAutomaticPunctuation:          true,
			EnableSpokenPunctuation:             nil,
			EnableSpokenEmojis:                  nil,
			DiarizationConfig:                   nil,
			Metadata:                            nil,
			Model:                               "",
			UseEnhanced:                         false,
		},
		Audio: &speechpb.RecognitionAudio{
			AudioSource: &speechpb.RecognitionAudio_Uri{Uri: gcsURI},
		},
	}

	op, err = g.speechCli.LongRunningRecognize(ctx, req)
	if err != nil {
		return model.FileTranscript{}, err
	}

	resp, err = op.Wait(ctx)
	if err != nil {
		return model.FileTranscript{}, err
	}
	ph := make([]model.TranscriptPhrase, 0, 0)
	cs := make([]model.TranscriptChannel, 0, 0)
	// Print the results.
	for _, result := range resp.Results {
		for _, alt := range result.Alternatives {
			ph = append(ph, model.TranscriptPhrase{
				TranscriptRange: model.TranscriptRange{
					StartSec: 0,
					EndSec:   0,
				},
				Channel: uint32(result.ChannelTag),
				Itn:     "",
				Display: alt.Transcript,
				Lexical: alt.Transcript,
				Words:   nil,
			})
		}
	}

	err = g.rm(fileName)
	if err != nil {
		return model.FileTranscript{}, err
	}

	log, _ := json.Marshal(resp)

	transcript := model.FileTranscript{
		Log:       log,
		CreatedAt: time.Now(),
		Locale:    locale,
		Phrases:   ph,
		Channels:  cs,
	}

	return transcript, nil
}

func (g *Stt) Callback(req map[string]interface{}) error {
	panic("TODO")
}

func (g *Stt) rm(object string) error {
	o := g.storageCli.Bucket(g.bucket).Object(object)
	ctx := context.Background()
	// Optional: set a generation-match precondition to avoid potential race
	// conditions and data corruptions. The request to upload is aborted if the
	// object's generation number does not match your precondition.
	attrs, err := o.Attrs(ctx)
	if err != nil {
		return fmt.Errorf("object.Attrs: %v", err)
	}
	o = o.If(storage.Conditions{GenerationMatch: attrs.Generation})

	if err = o.Delete(ctx); err != nil {
		return fmt.Errorf("Object(%q).Delete: %v", object, err)
	}

	return nil
}

func (g *Stt) upload(ctx context.Context, r io.Reader, object string) (string, error) {
	var err error
	o := g.storageCli.Bucket(g.bucket).Object(object)

	// Optional: set a generation-match precondition to avoid potential race
	// conditions and data corruptions. The request to upload is aborted if the
	// object's generation number does not match your precondition.
	// For an object that does not yet exist, set the DoesNotExist precondition.
	o = o.If(storage.Conditions{DoesNotExist: true})
	// If the live object already exists in your bucket, set instead a
	// generation-match precondition using the live object's generation number.
	//attrs, err := o.Attrs(ctx)
	//if err != nil {
	//	return "", err
	//}
	//o = o.If(storage.Conditions{GenerationMatch: attrs.Generation})

	// Upload an object with storage.Writer.
	wc := o.NewWriter(ctx)
	_, err = io.Copy(wc, r)
	if err != nil {
		return "", err
	}

	wc.Close()
	return fmt.Sprintf("gs://%s/%s", g.bucket, object), nil
}
