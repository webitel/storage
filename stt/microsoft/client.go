package microsoft

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"time"

	"github.com/webitel/storage/model"

	"github.com/pkg/errors"
)

const (
	ClientName = "Microsoft"
	HookName   = "Webitel STT"
)

var (
	ErrNotFound = errors.New("Not found files")
)

type client struct {
	id        int
	key       string
	region    string
	http      http.Client
	host      string
	signature string
	cbUri     string
}

type Config struct {
	Id       int    `json:"id"`
	Callback string `json:"callback"`
	Key      string `json:"key"`
	Region   string `json:"region"`
}

type transcriptRequest struct {
	ContentUrls []string `json:"contentUrls"`
	Properties  struct {
		WordLevelTimestampsEnabled bool   `json:"wordLevelTimestampsEnabled"`
		ProfanityFilterMode        string `json:"profanityFilterMode"`
	} `json:"properties"`
	Locale           string                 `json:"locale"`
	DisplayName      string                 `json:"displayName"`
	CustomProperties map[string]interface{} `json:"customProperties"`
}

type File struct {
	Name  string `json:"name"`
	Links struct {
		ContentUrl string `json:"contentUrl"`
	} `json:"links"`
}

type Files struct {
	Values []*File `json:"values"`
}

type Transcript struct {
	RecognizedPhrases []struct {
		Offset   string `json:"offset"`
		Duration string `json:"duration"`
		Channel  uint32 `json:"channel"`
		NBest    []struct {
			Words []struct {
				Word     string `json:"word"`
				Offset   string `json:"offset"`
				Duration string `json:"duration"`
			} `json:"words"`
			Itn     string `json:"itn"`
			Display string `json:"display"`
			Lexical string `json:"lexical"`
		} `json:"nBest"`
	} `json:"recognizedPhrases"`
	CombinedRecognizedPhrases []struct {
		Lexical string `json:"lexical"`
		Channel int    `json:"channel"`
		Display string `json:"display"`
	} `json:"combinedRecognizedPhrases"`
}

type Task struct {
	Id     string `json:"id"`
	Self   string `json:"self"`
	Status string `json:"status"`
	Links  struct {
		Files string `json:"files"`
	} `json:"links"`
	Properties struct {
		Error *struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	} `json:"properties"`
	CustomProperties map[string]interface{} `json:"customProperties"`
	Code             *string                `json:"code"`
	Message          *string                `json:"message"`
}

func NewClient(config Config) (*client, error) {
	h := hmac.New(sha256.New, []byte("bla bla"))
	c := &client{
		id:        config.Id,
		key:       config.Key,
		region:    config.Region,
		cbUri:     config.Callback,
		http:      http.Client{},
		host:      fmt.Sprintf("https://%s.api.cognitive.microsoft.com", config.Region),
		signature: hex.EncodeToString(h.Sum(nil)),
	}

	//c.getWebHook()

	return c, nil
}

type Hook struct {
	DisplayName string `json:"displayName"`
	Status      string `json:"status"`
	WebUrl      string `json:"webUrl"`
	Links       struct {
		Test string `json:"test"`
	} `json:"links"`
}

func (c *client) Transcript(ctx context.Context, id int64, fileUri, locale string) (model.FileTranscript, error) {
	var data []byte

	task, err := c.TranscriptJob(id, fileUri, locale)
	if err != nil {
		return model.FileTranscript{}, err
	}

	if _, err = c.WaitFoSuccess(ctx, task); err != nil {
		return model.FileTranscript{}, err
	}

	if task.Properties.Error != nil {
		return model.FileTranscript{}, errors.New(task.Properties.Error.Message)
	}

	data, err = c.LoadTranscript(task)
	if err != nil {
		return model.FileTranscript{}, err
	}

	ph, cs := getTranscript(data)

	res := model.FileTranscript{
		Log:       data,
		CreatedAt: time.Now(),
		Locale:    locale,
		Phrases:   ph,
		Channels:  cs,
	}

	return res, nil
}

func (c *client) Callback(req map[string]interface{}) error {
	panic("TODO")
}

func (t Task) Finished() bool {
	return t.Status == "Succeeded" || t.Status == "Failed"
}

func (c *client) TranscriptJob(fileId int64, fileUrl string, locale string) (*Task, error) {

	tr := &transcriptRequest{
		ContentUrls: []string{fileUrl},
		Properties: struct {
			WordLevelTimestampsEnabled bool   `json:"wordLevelTimestampsEnabled"`
			ProfanityFilterMode        string `json:"profanityFilterMode"`
		}{
			WordLevelTimestampsEnabled: false,
			ProfanityFilterMode:        "None",
		},
		Locale:      locale,
		DisplayName: fmt.Sprintf("Transcription using default model for %s", locale),
		CustomProperties: map[string]interface{}{
			"FileId": fileId,
		},
	}

	data, _ := json.Marshal(tr)

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/speechtotext/v3.0/transcriptions", c.host), bytes.NewBuffer(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Ocp-Apim-Subscription-Key", c.key)
	req.Header.Set("Content-Type", "application/json")

	res, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	data, err = ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	var t Task
	err = json.Unmarshal(data, &t)
	if err != nil {
		return nil, err
	}

	if res.StatusCode >= http.StatusMultipleChoices {
		if t.Message != nil {
			return nil, errors.New(*t.Message)
		} else {
			return nil, errors.New(fmt.Sprintf("Unknown error, code = %d", res.StatusCode))
		}
	}

	return &t, nil
}

func (c *client) Finished(t *Task) (bool, error) {
	// todo or error ?
	var data []byte
	if t.Finished() {
		return true, nil
	}

	req, err := http.NewRequest("GET", t.Self, nil)
	if err != nil {
		return false, err
	}
	req.Header.Set("Ocp-Apim-Subscription-Key", c.key)

	res, err := c.http.Do(req)
	if err != nil {
		return false, err
	}
	defer res.Body.Close()

	data, err = ioutil.ReadAll(res.Body)
	if err != nil {
		return false, err
	}

	err = json.Unmarshal(data, &t)
	if err != nil {
		return false, err
	}

	return t.Finished(), nil
}

func (c *client) LoadTranscript(t *Task) ([]byte, error) {
	files, err := c.GetFiles(t)
	if err != nil {
		return nil, err
	}

	if len(files) == 0 {
		return nil, ErrNotFound
	}

	file := files[0]
	var data []byte

	req, err := http.NewRequest("GET", file.Links.ContentUrl, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Ocp-Apim-Subscription-Key", c.key)

	res, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if data, err = ioutil.ReadAll(res.Body); err != nil {
		return nil, err
	}

	return data, nil
}

func (c *client) GetFiles(t *Task) ([]*File, error) {
	var data []byte

	req, err := http.NewRequest("GET", t.Links.Files, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Ocp-Apim-Subscription-Key", c.key)

	res, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if data, err = ioutil.ReadAll(res.Body); err != nil {
		return nil, err
	}

	fmt.Println(string(data))

	var result Files
	if err = json.Unmarshal(data, &result); err != nil {
		return nil, err
	}

	return result.Values, nil
}

func (c *client) WaitFoSuccess(ctx context.Context, t *Task) (ok bool, err error) {

	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(time.Second * 10):
			if ok, err = c.Finished(t); ok || err != nil {
				return
			}
		}
	}
}

func (c *client) getWebHook() (interface{}, error) {
	var data []byte

	req, err := http.NewRequest("GET", fmt.Sprintf("%s/speechtotext/v3.0/webhooks", c.host), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Ocp-Apim-Subscription-Key", c.key)

	res, err := c.http.Do(req)
	defer res.Body.Close()

	if data, err = ioutil.ReadAll(res.Body); err != nil {
		return nil, err
	}

	var hooks struct {
		Values []Hook `json:"values"`
	}

	err = json.Unmarshal(data, &hooks)
	if err != nil {
		return nil, err
	}

	var h *Hook

	for _, v := range hooks.Values {
		if v.DisplayName == HookName {
			h = &v
			break
		}
	}

	if h == nil {
		if h, err = c.registerWebHook(c.cbUri); err != nil {
			return nil, err
		}
	}

	err = c.testHook(*h)
	if err != nil {
		return nil, err
	}

	return data, nil
}

func (c *client) testHook(h Hook) error {

	req, err := http.NewRequest("POST", h.Links.Test, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Ocp-Apim-Subscription-Key", c.key)

	res, err := c.http.Do(req)
	defer res.Body.Close()

	if err != nil {
		return err
	}

	return nil
}

func (c *client) registerWebHook(uri string) (*Hook, error) {
	var data []byte

	in := `{
  "displayName": "%s",
  "properties": {
    "secret": "%s",
    "profile_id": %d
  },
  "webUrl": "%s",
  "events": {
    "transcriptionCompletion": true
  },
  "description": "I registered this URL to get a POST request for each completed transcription."
}`
	in = fmt.Sprintf(in, HookName, c.signature, c.id, uri)

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/speechtotext/v3.0/webhooks", c.host), bytes.NewBuffer([]byte(in)))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Ocp-Apim-Subscription-Key", c.key)

	res, err := c.http.Do(req)
	defer res.Body.Close()

	if data, err = ioutil.ReadAll(res.Body); err != nil {
		return nil, err
	}

	var h Hook
	err = json.Unmarshal(data, &h)
	if err != nil {
		return nil, err
	}
	return &h, nil
}

func getTranscript(data []byte) ([]model.TranscriptPhrase, []model.TranscriptChannel) {
	var n Transcript
	if err := json.Unmarshal(data, &n); err != nil {
		//TODO error
		return nil, nil
	}

	res := make([]model.TranscriptPhrase, 0, len(n.RecognizedPhrases))

	for _, v := range n.RecognizedPhrases {
		if len(v.NBest) < 1 {
			continue
		}

		words := make([]model.TranscriptWord, 0, len(v.NBest[0].Words))

		for _, w := range v.NBest[0].Words {
			words = append(words, model.TranscriptWord{
				Word: w.Word,
				TranscriptRange: model.TranscriptRange{
					StartSec: (ParseDuration(w.Offset)).Seconds(),
					EndSec:   (ParseDuration(w.Offset) + ParseDuration(w.Duration)).Seconds(),
				},
			})
		}

		res = append(res, model.TranscriptPhrase{
			TranscriptRange: model.TranscriptRange{
				StartSec: (ParseDuration(v.Offset)).Seconds(),
				EndSec:   (ParseDuration(v.Offset) + ParseDuration(v.Duration)).Seconds(),
			},
			Channel: v.Channel,
			Itn:     v.NBest[0].Itn,
			Display: v.NBest[0].Display,
			Lexical: v.NBest[0].Lexical,
			Words:   words,
		})
	}

	sort.Slice(res, func(i, j int) bool {
		return res[i].StartSec < res[j].StartSec
	})

	cs := make([]model.TranscriptChannel, 0, len(n.CombinedRecognizedPhrases))
	for _, v := range n.CombinedRecognizedPhrases {
		cs = append(cs, model.TranscriptChannel{
			Channel: v.Channel,
			Display: v.Display,
			Lexical: v.Lexical,
		})
	}

	return res, cs
}

var durationRegex = regexp.MustCompile(`P([\d\.]+Y)?([\d\.]+M)?([\d\.]+D)?T?([\d\.]+H)?([\d\.]+M)?([\d\.]+?S)?`)

// ParseDuration converts a ISO8601 duration into a time.Duration
func ParseDuration(str string) time.Duration {
	matches := durationRegex.FindStringSubmatch(str)

	years := parseDurationPart(matches[1], time.Hour*24*365)
	months := parseDurationPart(matches[2], time.Hour*24*30)
	days := parseDurationPart(matches[3], time.Hour*24)
	hours := parseDurationPart(matches[4], time.Hour)
	minutes := parseDurationPart(matches[5], time.Second*60)
	seconds := parseDurationPart(matches[6], time.Second)

	return time.Duration(years + months + days + hours + minutes + seconds)
}

func parseDurationPart(value string, unit time.Duration) time.Duration {
	if len(value) != 0 {
		if parsed, err := strconv.ParseFloat(value[:len(value)-1], 64); err == nil {
			return time.Duration(float64(unit) * parsed)
		}
	}
	return 0
}
