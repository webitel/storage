package synchronizer

import (
	"fmt"
	"github.com/webitel/storage/app"
	"github.com/webitel/storage/model"
	"github.com/webitel/storage/utils"
	"github.com/webitel/wlog"
	"io"
)

type transcoding struct {
	file model.SyncJob
	app  *app.App
}

func (j *transcoding) Execute() {
	var src io.ReadCloser

	defer func() {
		_ = j.app.Store.SyncFile().Remove(j.file.Id)
	}()

	file, store, err := j.app.GetFileWithProfile(j.file.DomainId, j.file.FileId)
	if err != nil {
		wlog.Error(err.Error())
		return
	}

	src, err = store.Reader(file, 0)
	if err != nil {
		wlog.Error(err.Error())
		//todo set db error
		return
	}
	defer src.Close()

	pr, pw := io.Pipe()

	th, e := utils.NewTranscoding(src, pw)
	if e != nil {
		wlog.Error(e.Error())
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
			wlog.Error(err.Error())
		}
		thumbnailFile.Size = size
	}()

	th.Start()
	th.Wait()
	pw.Close()
	err = <-rec
	if err != nil {
		wlog.Error(err.Error())
	} else {
		j.app.Store.File().Create(&thumbnailFile)
		j.app.Store.File().MarkRemove(j.file.DomainId, []int64{j.file.FileId})
		wlog.Debug(fmt.Sprintf("file %d transcoding \"%s\" from store \"%s\"", thumbnailFile.Id, thumbnailFile.Name, store.Name()))
	}
}
