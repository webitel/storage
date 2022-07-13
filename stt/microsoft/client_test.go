package microsoft

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"
)

func TestClient(t *testing.T) {
	c := &client{
		key:    os.Getenv("MS_KEY"),
		region: os.Getenv("MS_REGION"),
		http:   http.Client{},
	}

	ctx, _ := context.WithTimeout(context.Background(), time.Minute)

	task, err := c.TranscriptJob(0, "https://dev.webitel.com/api/storage/recordings/59673/stream?access_token=qutef4hgejfpmgyaqpdfq8d5mo", "uk-UA")
	if err != nil {
		t.Error(err.Error())
	}

	c.WaitFoSuccess(ctx, task)

	data, err := c.LoadTranscript(task)
	if err != nil {
		t.Error(err.Error())
	}
	fmt.Println(string(data))
}
