package synchronizer

import (
	"context"
	"io"

	"github.com/webitel/webitel-go-kit/pkg/logging"

	"github.com/webitel/storage/app"
	"github.com/webitel/storage/model"
	"github.com/webitel/storage/utils"
)

type transcoding struct {
	file   model.SyncJob
	app    *app.App
	logger *logging.Logger
}

func (j *transcoding) Execute(ctx context.Context) {
	var src io.ReadCloser

	defer func() {
		_ = j.app.Store.SyncFile().Remove(j.file.Id)
	}()

	file, store, err := j.app.GetFileWithProfile(j.file.DomainId, j.file.FileId)
	if err != nil {
		j.logger.ErrorCtx(ctx, "get file with profile", logging.Err(err))
		return
	}

	src, err = store.Reader(file, 0)
	if err != nil {
		j.logger.ErrorCtx(ctx, "open file reader", logging.Err(err))
		//todo set db error
		return
	}
	defer src.Close()

	pr, pw := io.Pipe()

	th, e := utils.NewTranscoding(src, pw)
	if e != nil {
		j.logger.ErrorCtx(ctx, "start transcoding", logging.Err(e))
		return
	}

	thumbnailFile := *file
	thumbnailFile.Size = 0
	thumbnailFile.Name = model.NewId()[:5] + "_" + file.Name + ".mp4"
	thumbnailFile.ViewName = &thumbnailFile.Name
	//thumbnailFile.MimeType = "video/x-matroska"
	thumbnailFile.MimeType = "video/mp4"

	rec := make(chan model.AppError)

	go func() {
		defer func() {
			close(rec)
		}()
		size, err := store.Write(pr, &thumbnailFile)
		if err != nil {
			j.logger.ErrorCtx(ctx, "write transcoded file", logging.Err(err))
		}
		thumbnailFile.Size = size
	}()

	th.Start()
	th.Wait()
	pw.Close()
	err = <-rec
	if err != nil {
		j.logger.ErrorCtx(ctx, "transcode file", logging.Err(err))
	} else {
		j.app.Store.File().Create(&thumbnailFile)
		j.app.Store.File().MarkRemove(j.file.DomainId, []int64{j.file.FileId})
		j.logger.DebugCtx(ctx, "file transcoded", logging.Str("file_name", thumbnailFile.Name), logging.Str("store", store.Name()))
	}
}
