package app

import (
	"crypto/sha256"
	"fmt"
	engine "github.com/webitel/engine/model"
	"github.com/webitel/storage/model"
	"github.com/webitel/storage/utils"
	"github.com/webitel/wlog"
	"io"
)

func (app *App) AddUploadJobFile(src io.Reader, file *model.JobUploadFile) engine.AppError {
	size, err := app.FileCache.Write(src, file)
	if err != nil {
		return err
	}

	file.Size = size
	file.Instance = app.GetInstanceId()

	file, err = app.Store.UploadJob().Create(file)
	if err != nil {
		wlog.Error(fmt.Sprintf("Failed to store file %s, %v", file.Uuid, err))
		if errRem := app.FileCache.Remove(file); errRem != nil {
			wlog.Error(fmt.Sprintf("Failed to remove cache file %v", err))
		}
	} else {
		wlog.Debug(fmt.Sprintf("create new file job %d upload file: %s [%d %s]", file.Id, file.Name, file.Size, file.MimeType))
	}

	return err
}

func (app *App) SyncUpload(src io.Reader, file *model.JobUploadFile) engine.AppError {
	if app.UseDefaultStore() {
		// error
	}

	sf, err := app.syncUpload(app.DefaultFileStore, src, file, nil)
	if err != nil {
		return err
	}

	return app.storeFile(app.DefaultFileStore, sf)
}

func (app *App) SyncUploadToProfile(src io.Reader, profileId int, file *model.JobUploadFile) engine.AppError {
	store, err := app.GetFileBackendStoreById(file.DomainId, profileId)
	if err != nil {
		return err
	}

	var reader io.Reader = src
	var ch chan engine.AppError
	var th *utils.Thumbnail

	if true {
		var e error
		th, e = utils.NewThumbnail(file.MimeType, "")
		if e != nil {
			panic(e.Error())
		}

		reader = io.TeeReader(src, th)

		var o = *file
		o.Name = "thumbnail_" + file.Name
		o.MimeType = "image/png"
		ch = make(chan engine.AppError)

		go func() {
			f, e := app.syncUpload(store, th.Reader(), &o, &profileId)
			if e != nil {
				fmt.Println(e)
			}
			th.UserData = &model.Thumbnail{
				BaseFile: f.BaseFile,
			}
			close(ch)
		}()
	}

	var sf *model.File
	sf, err = app.syncUpload(store, reader, file, &profileId)
	if err != nil {
		return err
	}

	if ch != nil {
		th.Close()
		err2 := <-ch

		if err2 != nil {

		}

		sf.Thumbnail = th.UserData.(*model.Thumbnail)
	}

	return app.storeFile(store, sf)
}

func (app *App) SafeUploadFileStream(store utils.FileBackend, src io.Reader, file *model.File) engine.AppError {
	h := sha256.New()
	tr := io.TeeReader(src, h)

	if store == nil {
		var err engine.AppError
		var todo int64 = 1
		if store, err = app.GetFileBackendStore(file.ProfileId, &todo); err != nil {
			return err
		}
	}

	size, err := store.Write(tr, file)
	if err != nil && err.GetId() != utils.ErrFileWriteExistsId {
		return err
	}

	// fixme
	file.Size += size
	sha := fmt.Sprintf("%x", h.Sum(nil))
	file.SHA256Sum = &sha
	return nil
}

func (app *App) syncUpload(store utils.FileBackend, src io.Reader, file *model.JobUploadFile, profileId *int) (*model.File, engine.AppError) {
	f := &model.File{
		DomainId:  file.DomainId,
		Uuid:      file.Uuid,
		CreatedAt: model.GetMillis(),
		BaseFile: model.BaseFile{
			Size:       file.Size,
			Name:       file.Name,
			ViewName:   file.ViewName,
			MimeType:   file.MimeType,
			Properties: model.StringInterface{},
			Instance:   app.GetInstanceId(),
		},
		ProfileId: profileId,
		Channel:   file.Channel,
	}

	h := sha256.New()
	tr := io.TeeReader(src, h)

	size, err := store.Write(tr, f)
	if err != nil && err.GetId() != utils.ErrFileWriteExistsId {
		return nil, err
	}
	// fixme
	file.Size = size
	sha := fmt.Sprintf("%x", h.Sum(nil))
	file.SHA256Sum = &sha
	f.Size = file.Size
	f.SHA256Sum = file.SHA256Sum

	return f, nil
}

func (app *App) storeFile(store utils.FileBackend, file *model.File) engine.AppError {
	res := <-app.Store.File().Create(file)
	if res.Err != nil {
		return res.Err
	} else {
		file.Id = res.Data.(int64)
	}
	wlog.Debug(fmt.Sprintf("store %s to %s %d bytes [sha256=%v]", file.GetStoreName(), store.Name(), file.Size, file.SHA256Sum))
	return nil
}
