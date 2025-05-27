package app

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"github.com/webitel/storage/model"
	"github.com/webitel/storage/utils"
	watcherkit "github.com/webitel/webitel-go-kit/pkg/watcher"
	"github.com/webitel/wlog"
	"io"
)

// AddUploadJobFile додає файл до черги завантаження
func (app *App) AddUploadJobFile(src io.Reader, file *model.JobUploadFile) model.AppError {
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
func (app *App) SyncUpload(src io.Reader, file *model.JobUploadFile) model.AppError {
	if !app.UseDefaultStore() {
		return model.NewInternalError("SyncUpload", "default store error")
	}

	return app.upload(src, nil, app.DefaultFileStore, file)
}

// SyncUploadToProfile синхронно завантажує файл у профіль користувача
func (app *App) SyncUploadToProfile(src io.Reader, profileId int, file *model.JobUploadFile) model.AppError {
	store, err := app.GetFileBackendStoreById(file.DomainId, profileId)
	if err != nil {
		return err
	}

	return app.upload(src, &profileId, store, file)
}

// upload - основний метод завантаження файлу з підтримкою мініатюр
func (app *App) upload(src io.Reader, profileId *int, store utils.FileBackend, file *model.JobUploadFile) model.AppError {
	var reader io.Reader
	var thumbnail *utils.Thumbnail
	var ch chan model.AppError
	var err model.AppError

	if file.GenerateThumbnail {
		reader, thumbnail, ch, err = app.setupThumbnail(src, store, file)
		if err != nil {
			return err
		}
		if thumbnail != nil {
			defer func() {
				thumbnail.Close()
			}()
		}
	} else {
		reader = src
	}

	// Завантаження основного файлу
	sf, err := app.syncUpload(store, reader, file, profileId)
	if err != nil {
		return err
	}
	file.Size = sf.Size

	// Завершення обробки мініатюри, якщо вона існує
	if ch != nil {
		if err := <-ch; err != nil {
			return err
		}
		sf.Thumbnail = thumbnail.UserData.(*model.Thumbnail)
		file.Thumbnail = sf.Thumbnail
	}

	file.Id, err = app.storeFile(store, sf)
	if err != nil {
		return err
	}

	return nil
}

// setupThumbnail налаштовує мініатюру для файлу, якщо це зображення або відео
func (app *App) setupThumbnail(src io.Reader, store utils.FileBackend, file *model.JobUploadFile) (io.Reader, *utils.Thumbnail, chan model.AppError, model.AppError) {
	if !utils.IsSupportThumbnail(file.MimeType) {
		return src, nil, nil, nil
	}

	thumbnail, err := utils.NewThumbnail(file.MimeType, app.thumbnailSettings.DefaultScale)
	if err != nil {
		return nil, nil, nil, model.NewInternalError("ThumbnailError", err.Error())
	}

	reader := io.TeeReader(src, thumbnail)

	thumbnailFile := *file
	thumbnailFile.Name = "thumbnail_" + file.Name + ".png"
	thumbnailFile.ViewName = &thumbnailFile.Name
	thumbnailFile.MimeType = "image/png"
	ch := make(chan model.AppError)

	go func() {
		if f, e := app.syncUpload(store, thumbnail.Reader(), &thumbnailFile, nil); e != nil {
			ch <- e
		} else {
			thumbnail.UserData = &model.Thumbnail{BaseFile: f.BaseFile, Scale: thumbnail.Scale()}
		}
		close(ch)
	}()

	return reader, thumbnail, ch, nil
}

// syncUpload здійснює запис файлу до файлового сховища
func (app *App) syncUpload(store utils.FileBackend, src io.Reader, file *model.JobUploadFile, profileId *int) (*model.File, model.AppError) {
	f := &model.File{
		DomainId:  file.DomainId,
		Uuid:      file.Uuid,
		CreatedAt: model.GetMillis(),
		BaseFile: model.BaseFile{
			Size:           file.Size,
			Name:           file.Name,
			ViewName:       file.ViewName,
			MimeType:       file.MimeType,
			Properties:     file.Properties,
			Instance:       app.GetInstanceId(),
			Channel:        file.Channel,
			RetentionUntil: file.RetentionUntil,
			UploadedBy:     file.UploadedBy,
		},
		ProfileId: profileId,
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
func (app *App) storeFile(store utils.FileBackend, file *model.File) (int64, model.AppError) {
	res := <-app.Store.File().Create(file)
	if res.Err != nil {
		return 0, res.Err
	}

	wlog.Debug(fmt.Sprintf("Stored %s in %s, %d bytes [encrypted=%v, SHA256=%v]", file.GetStoreName(), store.Name(), file.Size, file.IsEncrypted(), file.SHA256Sum != nil))

	//TODO
	if file.Channel != nil && *file.Channel == model.UploadFileChannelCase {
		if notifyErr := app.watcherManager.Notify(
			model.PermissionScopeFiles,
			watcherkit.EventTypeCreate,
			NewFileWatcherData(file),
		); notifyErr != nil {
			wlog.Error(fmt.Sprintf("could not notify file store: %s, ", notifyErr.Error()))
		}
	}
	return res.Data.(int64), nil
}

type FileWatcherData struct {
	file *model.File
	Args map[string]any
}

func NewFileWatcherData(file *model.File) *FileWatcherData {
	return &FileWatcherData{
		file: file,
		Args: map[string]any{"obj": file},
	}
}

func (wd *FileWatcherData) Marshal() ([]byte, error) {
	return json.Marshal(wd.file)
}

func (wd *FileWatcherData) GetArgs() map[string]any {
	return wd.Args
}
