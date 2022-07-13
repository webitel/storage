package synchronizer

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/webitel/storage/app"
	"github.com/webitel/storage/model"
	"github.com/webitel/wlog"
)

type SttJob struct {
	file model.SyncJob
	app  *app.App
}

type params struct {
	ProfileId int    `json:"profile_id"`
	Locale    string `json:"locale"`
}

func (s *SttJob) Execute() {
	var p model.TranscriptOptions
	json.Unmarshal(s.file.Config, &p)

	n := time.Now()

	s.transcript(p)

	err := s.app.Store.SyncFile().Remove(s.file.Id)
	if err != nil {
		wlog.Error(fmt.Sprintf("[stt] file %d, error: %s", s.file.FileId, err.Error()))
	}
	wlog.Debug(fmt.Sprintf("[stt] job_id: %d, file_id: %d stop, time %v", s.file.Id, s.file.FileId, time.Since(n)))

}

func (s *SttJob) transcript(params model.TranscriptOptions) {
	wlog.Debug(fmt.Sprintf("[stt] job_id: %d, file_id: %d start transcript to %s", s.file.Id, s.file.FileId, params.Locale))
	t, err := s.app.TranscriptFile(s.file.FileId, params)
	if err != nil {
		wlog.Error(err.Error())
	} else {
		wlog.Debug(fmt.Sprintf("[stt] file %d, transcript: %s", s.file.FileId, t.Transcript))
	}
}
