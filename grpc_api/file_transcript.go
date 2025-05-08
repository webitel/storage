package grpc_api

import (
	"context"
	"fmt"
	"github.com/webitel/storage/model"
	"golang.org/x/sync/singleflight"
	"time"

	"github.com/webitel/storage/gen/engine"
	"github.com/webitel/storage/gen/storage"

	"github.com/webitel/storage/controller"
)

type fileTranscript struct {
	ctrl            *controller.Controller
	getProfileGroup singleflight.Group
	storage.UnsafeFileTranscriptServiceServer
}

func NewFileTranscriptApi(c *controller.Controller) *fileTranscript {
	return &fileTranscript{
		ctrl:            c,
		getProfileGroup: singleflight.Group{},
	}
}

func (api *fileTranscript) CreateFileTranscript(ctx context.Context, in *storage.StartFileTranscriptRequest) (*storage.StartFileTranscriptResponse, error) {
	session, err := api.ctrl.GetSessionFromCtx(ctx)
	if err != nil {
		return nil, err
	}

	ops := &model.TranscriptOptions{
		FileIds: in.GetFileId(),
		Uuid:    in.GetUuid(),
	}

	if in.GetLocale() != "" {
		ops.Locale = &in.Locale
	}

	if in.GetProfile().GetId() > 0 {
		ops.ProfileId = model.NewInt(int(in.GetProfile().GetId()))
	}

	list, err := api.ctrl.TranscriptFiles(session, ops)
	if err != nil {
		return nil, err
	}

	res := &storage.StartFileTranscriptResponse{
		Items: make([]*storage.StartFileTranscriptResponse_TranscriptJob, 0, len(list)),
	}

	for _, v := range list {
		res.Items = append(res.Items, &storage.StartFileTranscriptResponse_TranscriptJob{
			Id:        v.Id,
			FileId:    v.FileId,
			CreatedAt: v.CreatedAt,
		})
	}

	return res, nil
}

func (api *fileTranscript) GetFileTranscriptPhrases(ctx context.Context, in *storage.GetFileTranscriptPhrasesRequest) (*storage.ListPhrases, error) {
	session, err := api.ctrl.GetSessionFromCtx(ctx)
	if err != nil {
		return nil, err
	}

	var list []*model.TranscriptPhrase
	var endOfList bool

	req := &model.ListRequest{
		Page:    int(in.GetPage()),
		PerPage: int(in.GetSize()),
	}

	list, endOfList, err = api.ctrl.TranscriptFilePhrases(session, in.GetId(), req)

	if err != nil {
		return nil, err
	}

	items := make([]*storage.TranscriptPhrase, 0, len(list))
	for _, v := range list {
		items = append(items, &storage.TranscriptPhrase{
			StartSec: float32(v.StartSec),
			EndSec:   float32(v.EndSec),
			Channel:  v.Channel,
			Phrase:   v.Display,
		})
	}
	return &storage.ListPhrases{
		Next:  !endOfList,
		Items: items,
	}, nil
}

func (api *fileTranscript) DeleteFileTranscript(ctx context.Context, in *storage.DeleteFileTranscriptRequest) (*storage.DeleteFileTranscriptResponse, error) {
	session, err := api.ctrl.GetSessionFromCtx(ctx)
	if err != nil {
		return nil, err
	}

	var ids []int64
	ids, err = api.ctrl.DeleteTranscript(session, in.GetId(), in.GetUuid())
	if err != nil {
		return nil, err
	}

	return &storage.DeleteFileTranscriptResponse{
		Items: ids,
	}, nil
}

func (api *fileTranscript) FileTranscriptSafe(ctx context.Context, in *storage.FileTranscriptSafeRequest) (*storage.FileTranscriptSafeResponse, error) {
	ops := &model.TranscriptOptions{
		FileIds: []int64{in.GetFileId()},
	}

	if in.GetLocale() != "" {
		ops.Locale = &in.Locale
	}

	if in.GetProfileId() > 0 {
		ops.ProfileId = model.NewInt(int(in.GetProfileId()))
	}
	value, err, _ := api.getProfileGroup.Do(fmt.Sprintf("%d-%d", in.GetDomainId(), ops.ProfileId), func() (interface{}, error) {
		return api.ctrl.GetProfileWithoutAuth(in.GetDomainId(), int64(*ops.ProfileId))
	})

	profile := value.(*model.CognitiveProfile)
	if !profile.Enabled {
		return nil, model.NewBadRequestError("grpc_api.file_transcript.create_file_transcript_safe.profile_disabled.error", fmt.Sprintf("profile id=%d is disabled", ops.ProfileId))
	}
	syncTime := profile.UpdatedAt.Unix()
	ops.ProfileSyncTime = &syncTime
	t, err := api.ctrl.TranscriptFilesSafe(in.FileId, ops)
	if err != nil {
		return nil, err
	}

	res := &storage.FileTranscriptSafeResponse{
		Id: t.Id,
		File: &engine.Lookup{
			Id:   int64(t.File.Id),
			Name: t.File.Name,
		},
		Transcript: t.Transcript,
		CreatedAt:  t.CreatedAt.Unix(),
		Locale:     t.Locale,
	}

	if t.Profile != nil {
		res.Profile = &engine.Lookup{
			Id:   int64(t.Profile.Id),
			Name: t.Profile.Name,
		}
	}

	return res, nil
}

func (api *fileTranscript) PutFileTranscript(ctx context.Context, in *storage.PutFileTranscriptRequest) (*storage.PutFileTranscriptResponse, error) {
	session, err := api.ctrl.GetSessionFromCtx(ctx)
	if err != nil {
		return nil, err
	}

	if in.GetUuid() == "" {
		return nil, model.NewBadRequestError("storage.stt.transcript.valid.uuid", "UUID is required")
	}

	if in.GetFileId() == 0 {
		return nil, model.NewBadRequestError("storage.stt.transcript.valid.file_id", "file_id is required")
	}

	req := model.FileTranscript{
		Id: 0,
		File: model.Lookup{
			Id: int(in.GetFileId()),
		},
		Profile:    nil,
		Transcript: in.GetText(),
		Log:        nil,
		CreatedAt:  time.Now(),
		Locale:     in.GetLocale(),
		Phrases:    nil,
		Channels:   nil,
	}

	req.Phrases = make([]model.TranscriptPhrase, 0, len(in.GetPhrases()))
	for _, v := range in.GetPhrases() {
		req.Phrases = append(req.Phrases, model.TranscriptPhrase{
			TranscriptRange: model.TranscriptRange{
				StartSec: float64(v.StartSec),
				EndSec:   float64(v.EndSec),
			},
			Channel: v.Channel,
			Itn:     "",
			Display: v.Phrase,
			Lexical: v.Phrase,
			Words:   nil,
		})
	}
	var id int64
	id, err = api.ctrl.PutTranscript(ctx, session, in.GetUuid(), req)
	if err != nil {
		return nil, err
	}

	return &storage.PutFileTranscriptResponse{
		Id: id,
	}, nil
}
