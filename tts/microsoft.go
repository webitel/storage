package tts

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"

	engine "github.com/webitel/engine/model"
	"github.com/webitel/wlog"
)

func Microsoft(req TTSParams) (io.ReadCloser, *string, *int, error) {
	var request *http.Request
	var data string
	token, err := microsoftToken(fixKey(req.Key), req.Region)
	if err != nil {
		return nil, nil, nil, err
	}

	if req.Language == "" {
		req.Language = req.Locale
	}

	data = fmt.Sprintf(`<speak version='1.0' xmlns="http://www.w3.org/2001/10/synthesis" xmlns:mstts="https://www.w3.org/2001/mstts" xml:lang='%s'>
	%s
	<voice xml:lang='%s' xml:gender='%s' name='%s'>
	%s
	 </voice>
</speak>
`, req.Language, req.BackgroundNode(), req.Language, req.Voice, microsoftLocalesNameMapping(req.Language, req.Voice), req.Text)

	request, err = http.NewRequest("POST", fmt.Sprintf("https://%s.tts.speech.microsoft.com/cognitiveservices/v1", req.Region), bytes.NewBuffer([]byte(data)))
	if err != nil {
		return nil, nil, nil, err
	}

	request.Header.Set("Content-Type", "application/ssml+xml")
	request.Header.Set("User-Agent", "WebitelACR")
	request.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	if strings.Index(req.Format, "wav") > -1 {
		request.Header.Set("X-Microsoft-OutputFormat", "riff-8khz-8bit-mono-mulaw")
	} else {
		request.Header.Set("X-Microsoft-OutputFormat", "audio-16khz-32kbitrate-mono-mp3")
	}

	result, err := http.DefaultClient.Do(request)
	if err != nil {
		return nil, nil, nil, err
	}

	if result.StatusCode != http.StatusOK {
		e, _ := ioutil.ReadAll(result.Body)
		if (len(e)) == 0 {
			e = []byte("empty response")
		}
		if e != nil {
			wlog.Error("[tts] microsoft error: " + string(e))

			return nil, nil, nil, engine.NewCustomCodeError("tts.microsoft", string(e), result.StatusCode)
		}
	}

	contentType := result.Header.Get("Content-Type")

	if contentType == "" {
		contentType = "audio/wav"
	}

	return result.Body, &contentType, nil, nil
}

func microsoftToken(key, region string) (string, error) {
	req, err := http.NewRequest("POST", fmt.Sprintf("https://%s.api.cognitive.microsoft.com/sts/v1.0/issueToken", region), nil)
	if err != nil {
		return "", err
	}

	req.Header.Set("Context-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Ocp-Apim-Subscription-Key", key)

	res, err := http.DefaultClient.Do(req)

	if err != nil {
		return "", err
	}

	defer res.Body.Close()

	var data []byte
	data, err = ioutil.ReadAll(res.Body)

	if err != nil {
		return "", err
	}

	return string(data), nil
}

func fixKey(key []byte) string {
	if len(key) < 3 {
		return ""
	}
	if key[0] == '"' {
		key = key[1:]
	}
	l := len(key)

	if key[l-1] == '"' {
		key = key[:l-1]
	}

	return string(key)
}
