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

// AddUploadJobFile додає файл до черги завантаження
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
		wlog.Debug(fmt.Sprintf("Created new file job %d, upload file: %s [%d %s]", file.Id, file.Name, file.Size, file.MimeType))
	}

	return err
}

// SyncUpload синхронно завантажує файл за замовчуванням
func (app *App) SyncUpload(src io.Reader, tryThumbnail bool, file *model.JobUploadFile) engine.AppError {
	if !app.UseDefaultStore() {
		return engine.NewInternalError("SyncUpload", "default store error")
	}

	return app.upload(src, nil, app.DefaultFileStore, tryThumbnail, file)
}

// SyncUploadToProfile синхронно завантажує файл у профіль користувача
func (app *App) SyncUploadToProfile(src io.Reader, profileId int, tryThumbnail bool, file *model.JobUploadFile) engine.AppError {
	store, err := app.GetFileBackendStoreById(file.DomainId, profileId)
	if err != nil {
		return err
	}

	return app.upload(src, &profileId, store, tryThumbnail, file)
}

// upload - основний метод завантаження файлу з підтримкою мініатюр
func (app *App) upload(src io.Reader, profileId *int, store utils.FileBackend, tryThumbnail bool, file *model.JobUploadFile) engine.AppError {
	var reader io.Reader
	var thumbnail *utils.Thumbnail
	var ch chan engine.AppError
	var err engine.AppError

	if tryThumbnail {
		reader, thumbnail, ch, err = app.setupThumbnail(src, file)
		if err != nil {
			return err
		}
	} else {
		reader = src
	}

	// Завантаження основного файлу
	sf, err := app.syncUpload(store, reader, file, profileId)
	if err != nil {
		return err
	}

	// Завершення обробки мініатюри, якщо вона існує
	if ch != nil {
		if err := <-ch; err != nil {
			return err
		}
		sf.Thumbnail = thumbnail.UserData.(*model.Thumbnail)
	}

	return app.storeFile(store, sf)
}

// setupThumbnail налаштовує мініатюру для файлу, якщо це зображення або відео
func (app *App) setupThumbnail(src io.Reader, file *model.JobUploadFile) (io.Reader, *utils.Thumbnail, chan engine.AppError, engine.AppError) {
	if !utils.IsSupportThumbnail(file.MimeType) {
		return src, nil, nil, nil
	}

	thumbnail, err := utils.NewThumbnail(file.MimeType, "")
	if err != nil {
		return nil, nil, nil, engine.NewInternalError("ThumbnailError", err.Error())
	}

	reader := io.TeeReader(src, thumbnail)

	thumbnailFile := *file
	thumbnailFile.Name = "thumbnail_" + file.Name
	thumbnailFile.MimeType = "image/png"
	ch := make(chan engine.AppError)

	go func() {
		if f, e := app.syncUpload(app.DefaultFileStore, thumbnail.Reader(), &thumbnailFile, nil); e != nil {
			ch <- e
		} else {
			thumbnail.UserData = &model.Thumbnail{BaseFile: f.BaseFile}
		}
		close(ch)
	}()

	return reader, thumbnail, ch, nil
}

// syncUpload здійснює запис файлу до файлового сховища
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

	sha := fmt.Sprintf("%x", h.Sum(nil))
	file.SHA256Sum = &sha
	f.Size = size
	f.SHA256Sum = &sha

	return f, nil
}

// storeFile зберігає інформацію про файл у базі даних
func (app *App) storeFile(store utils.FileBackend, file *model.File) engine.AppError {
	res := <-app.Store.File().Create(file)
	if res.Err != nil {
		return res.Err
	}

	file.Id = res.Data.(int64)
	wlog.Debug(fmt.Sprintf("Stored %s in %s, %d bytes [SHA256=%v]", file.GetStoreName(), store.Name(), file.Size, file.SHA256Sum))
	return nil
}
