package app

import (
	"context"
	"strconv"
	"time"

	engine "github.com/webitel/engine/model"
	"github.com/webitel/storage/stt/google"

	"github.com/webitel/storage/stt"

	"github.com/webitel/storage/stt/microsoft"

	"github.com/webitel/storage/model"
	"github.com/webitel/wlog"
)

func (app *App) GetSttProfileById(id int) (*model.CognitiveProfile, engine.AppError) {
	return app.Store.CognitiveProfile().GetById(int64(id))
}

func (app *App) JobCallbackUri(profileId int64) string {
	return app.Config().ServiceSettings.PublicHost + "/api/storage/jobs/callback?profile_id=" + strconv.Itoa(int(profileId))
}

func (app *App) GetSttProfile(id *int, syncTime *int64) (p *model.CognitiveProfile, appError engine.AppError) {
	var ok bool
	var cache interface{}

	if id == nil {
		return nil, engine.NewInternalError("", "")
	}

	cache, ok = app.sttProfilesCache.Get(*id)
	if ok {
		p = cache.(*model.CognitiveProfile)
		if syncTime != nil && p.GetSyncTime() == *syncTime {
			return
		}
	}

	if p == nil || syncTime == nil {
		p, appError = app.GetSttProfileById(*id)
		if appError != nil {
			return
		}
	}

	if appError != nil {
		return
	}

	var err error
	switch p.Provider {
	case microsoft.ClientName:

		if p.Instance, err = microsoft.NewClient(microsoft.ConfigFromJson(*id, app.JobCallbackUri(p.Id), p.JsonProperties())); err != nil {
			// TODO
		}
	case google.ClientName:
		if p.Instance, err = google.New(google.ConfigFromJson(p.JsonProperties())); err != nil {

		}

	default:
		//todo error
	}

	app.sttProfilesCache.Add(*id, p)
	wlog.Info("[stt] Added to cache", wlog.String("name", p.Name))
	return p, nil
}

func (app *App) TranscriptFile(fileId int64, options model.TranscriptOptions) (*model.FileTranscript, engine.AppError) {
	var fileUri string
	p, err := app.GetSttProfile(options.ProfileId, options.ProfileSyncTime)
	if err != nil {
		return nil, err
	}

	//if !p.Enabled {
	//	return nil, engine.NewInternalError("app.stt.transcript.valid", "Profile is disabled")
	//}

	stt, ok := p.Instance.(stt.Stt)
	if !ok {
		return nil, engine.NewInternalError("app.stt.transcript.valid", "Bad client interface")
	}

	fileUri, err = app.GeneratePreSignetResourceSignature(model.AnyFileRouteName, "download", fileId, p.DomainId)
	if err != nil {
		return nil, err
	}

	ctx, _ := context.WithTimeout(context.TODO(), time.Hour*2) // TODO

	//app.jobCallback.Add(fileId, cn)
	//defer app.jobCallback.Remove(fileId)

	if transcript, e := stt.Transcript(ctx, fileId, app.publicUri(fileUri), p.GetLocale(options.Locale)); e != nil {
		return nil, engine.NewInternalError("app.stt.transcript.err", e.Error())
	} else {
		transcript.File = model.Lookup{
			Id: int(fileId),
		}
		transcript.Profile = model.Lookup{
			Id: int(p.Id),
		}

		return app.Store.TranscriptFile().Store(&transcript)
	}
}

func (app *App) CreateTranscriptFilesJob(domainId int64, options *model.TranscriptOptions) ([]*model.FileTranscriptJob, engine.AppError) {
	return app.Store.TranscriptFile().CreateJobs(domainId, *options)
}

func (app *App) TranscriptFilePhrases(domainId, id int64, search *model.ListRequest) ([]*model.TranscriptPhrase, bool, engine.AppError) {
	phrases, err := app.Store.TranscriptFile().GetPhrases(domainId, id, search)
	if err != nil {
		return nil, false, err
	}

	search.RemoveLastElemIfNeed(&phrases)
	return phrases, search.EndOfList(), nil
}

func (app *App) RemoveTranscript(domainId int64, ids []int64, uuid []string) ([]int64, engine.AppError) {
	return app.Store.TranscriptFile().Delete(domainId, ids, uuid)
}

func (app *App) publicUri(uri string) string {
	return app.Config().ServiceSettings.PublicHost + uri
}
