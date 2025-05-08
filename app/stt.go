package app

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/webitel/storage/stt/google"
	"golang.org/x/sync/singleflight"

	"github.com/webitel/storage/stt"

	"github.com/webitel/storage/stt/microsoft"

	"github.com/webitel/storage/model"
	"github.com/webitel/wlog"
)

var (
	sttGroup singleflight.Group
)

func (app *App) GetSttProfileById(id int) (*model.CognitiveProfile, model.AppError) {
	return app.Store.CognitiveProfile().GetById(int64(id))
}

func (app *App) JobCallbackUri(profileId int64) string {
	return app.Config().ServiceSettings.PublicHost + "/api/storage/jobs/callback?profile_id=" + strconv.Itoa(int(profileId))
}

func (app *App) GetSttProfile(id *int, syncTime *int64) (*model.CognitiveProfile, model.AppError) {
	if id == nil {
		return nil, model.NewInternalError("app.stt.valid.id", "Profile ID is required")
	}

	p, err, _ := sttGroup.Do(fmt.Sprintf("stt-%d", *id), func() (interface{}, error) {
		return app.getSttProfile(id, syncTime)
	})

	if err != nil {
		switch err.(type) {
		case model.AppError:
			return nil, err.(model.AppError)
		default:
			return nil, model.NewInternalError("app.stt.app_err", err.Error())
		}
	}

	return p.(*model.CognitiveProfile), nil
}

func (app *App) getSttProfile(id *int, syncTime *int64) (p *model.CognitiveProfile, appError model.AppError) {
	var ok bool
	var cache interface{}

	cache, ok = app.sttProfilesCache.Get(*id)
	if ok {
		p = cache.(*model.CognitiveProfile)
		if syncTime != nil && p.GetSyncTag() == *syncTime {
			return
		}
	}
	p = nil

	if p == nil || syncTime == nil {
		p, appError = app.GetSttProfileById(*id)
		if appError != nil {
			return
		}
		if syncTime != nil {
			p.SyncTag = *syncTime
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

func (app *App) TranscriptFile(fileId int64, options model.TranscriptOptions) (*model.FileTranscript, model.AppError) {
	var fileUri string
	p, err := app.GetSttProfile(options.ProfileId, options.ProfileSyncTime)
	if err != nil {
		return nil, err
	}

	//if !p.Enabled {
	//	return nil, model.NewInternalError("app.stt.transcript.valid", "Profile is disabled")
	//}

	stt, ok := p.Instance.(stt.Stt)
	if !ok {
		return nil, model.NewInternalError("app.stt.transcript.valid", "Bad client interface")
	}

	fileUri, err = app.GeneratePreSignedResourceSignatureBulk(fileId, p.DomainId, model.AnyFileRouteName, "download", "", nil)
	if err != nil {
		return nil, err
	}

	ctx, _ := context.WithTimeout(context.TODO(), time.Hour*2) // TODO

	//app.jobCallback.Add(fileId, cn)
	//defer app.jobCallback.Remove(fileId)

	if transcript, e := stt.Transcript(ctx, fileId, app.publicUri(fileUri), p.GetLocale(options.Locale)); e != nil {
		return nil, model.NewInternalError("app.stt.transcript.err", e.Error())
	} else {
		transcript.File = model.Lookup{
			Id: int(fileId),
		}
		transcript.Profile = &model.Lookup{
			Id: int(p.Id),
		}

		return app.Store.TranscriptFile().Store(&transcript)
	}
}

func (app *App) CreateTranscriptFilesJob(domainId int64, options *model.TranscriptOptions) ([]*model.FileTranscriptJob, model.AppError) {
	return app.Store.TranscriptFile().CreateJobs(domainId, *options)
}

func (app *App) TranscriptFilePhrases(domainId, id int64, search *model.ListRequest) ([]*model.TranscriptPhrase, bool, model.AppError) {
	phrases, err := app.Store.TranscriptFile().GetPhrases(domainId, id, search)
	if err != nil {
		return nil, false, err
	}

	search.RemoveLastElemIfNeed(&phrases)
	return phrases, search.EndOfList(), nil
}

func (app *App) RemoveTranscript(domainId int64, ids []int64, uuid []string) ([]int64, model.AppError) {
	return app.Store.TranscriptFile().Delete(domainId, ids, uuid)
}

func (app *App) PutTranscript(ctx context.Context, domainId int64, uuid string, tr model.FileTranscript) (int64, model.AppError) {
	return app.Store.TranscriptFile().Put(ctx, domainId, uuid, tr)
}

func (app *App) publicUri(uri string) string {
	return app.Config().ServiceSettings.PublicHost + uri
}
