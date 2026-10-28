package synchronizer

import (
	"context"
	"encoding/json"
	"time"

	"github.com/webitel/webitel-go-kit/pkg/logging"

	"github.com/webitel/storage/app"
	"github.com/webitel/storage/model"
)

type SttJob struct {
	file   model.SyncJob
	app    *app.App
	logger *logging.Logger
}

type params struct {
	ProfileId int    `json:"profile_id"`
	Locale    string `json:"locale"`
}

func (s *SttJob) Execute(ctx context.Context) {
	var p model.TranscriptOptions
	json.Unmarshal(s.file.Config, &p)

	n := time.Now()

	var locale string
	if p.Locale != nil {
		locale = *p.Locale
	}
	s.logger.DebugCtx(ctx, "stt job started", logging.Str("locale", locale))
	t, err := s.app.TranscriptFile(s.file.FileId, p)
	if err != nil {
		s.logger.ErrorCtx(ctx, "transcript file", logging.Err(err))
		if err = s.app.Store.SyncFile().SetError(s.file.Id, err); err != nil {
			s.logger.ErrorCtx(ctx, "set sync job error", logging.Err(err))
		}
		return
	} else {
		s.logger.DebugCtx(ctx, "file transcribed", logging.Str("transcript", t.Transcript))
	}

	err = s.app.Store.SyncFile().Remove(s.file.Id)
	if err != nil {
		s.logger.ErrorCtx(ctx, "remove sync job", logging.Err(err))
	}
	s.logger.DebugCtx(ctx, "stt job finished", logging.Dur("duration", time.Since(n)))

}
