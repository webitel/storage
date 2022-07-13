package grpc_api

import (
	"context"

	"github.com/webitel/storage/model"

	"github.com/webitel/protos/storage"
	"github.com/webitel/storage/controller"
)

type fileTranscript struct {
	ctrl *controller.Controller
}

func NewFileTranscriptApi(api *controller.Controller) *fileTranscript {
	return &fileTranscript{
		ctrl: api,
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
