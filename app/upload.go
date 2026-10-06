package app

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path"
	"strings"
	"time"

	"github.com/hashicorp/golang-lru/v2/expirable"
	"golang.org/x/sync/singleflight"

	watcherkit "github.com/webitel/webitel-go-kit/pkg/watcher"
	"github.com/webitel/wlog"

	"github.com/webitel/storage/model"
	"github.com/webitel/storage/utils"
)

var (
	domainProfileCache = expirable.NewLRU[int64, utils.FileBackend](100, nil, time.Second*15)
	sgDomainProfile    = singleflight.Group{}
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
		wlog.Debug(fmt.Sprintf("created new file job %d, upload file: %s [%d %s]", file.Id, file.Name, file.Size, file.MimeType))
	}

	return err
}

func (app *App) domainStore(domainId int64) (utils.FileBackend, model.AppError) {
	store, ok := domainProfileCache.Get(domainId)
	if ok {
		return store, nil
	}

	v, err, shared := sgDomainProfile.Do(fmt.Sprintf("%d", domainId), func() (any, error) {
		var hk *model.DomainFileBackendHashKey
		var err model.AppError
		var s utils.FileBackend

		hk, err = app.Store.FileBackendProfile().Default(domainId)
		if err != nil {
			return nil, err
		}

		if hk != nil {
			if s, err = app.GetFileBackendStore(&hk.Id, &hk.UpdatedAt); err != nil {
				return nil, err
			}
			return s, nil
		}

		if !app.UseDefaultStore() {
			return nil, model.NewInternalError("SyncUpload", "default store error")
		}

		return app.DefaultFileStore, nil
	})

	if err != nil {
		switch err.(type) {
		case model.AppError:
			return nil, err.(model.AppError)
		default:
			return nil, model.NewInternalError("app.domain_store.app_error", err.Error())
		}
	}

	if !shared {
		domainProfileCache.Add(domainId, v.(utils.FileBackend))
	}

	return v.(utils.FileBackend), nil
}

// SyncUpload синхронно завантажує файл за замовчуванням
func (app *App) SyncUpload(src io.Reader, file *model.JobUploadFile) model.AppError {
	store, err := app.domainStore(file.DomainId)
	if err != nil {
		return err
	}

	return app.upload(src, store.Id(), store, file)
}

// SyncUploadToProfile синхронно завантажує файл у профіль користувача
func (app *App) SyncUploadToProfile(src io.Reader, profileId int, file *model.JobUploadFile) model.AppError {
	store, err := app.GetFileBackendStoreById(file.DomainId, profileId)
	if err != nil {
		return err
	}

	return app.upload(src, &profileId, store, file)
}

func (app *App) useClamd(file *model.BaseFile) bool {
	return app.clamd != nil && file.Channel != nil && *file.Channel == model.UploadFileChannelChat
}

func (app *App) upload(src io.Reader, profileId *int, store utils.FileBackend, file *model.JobUploadFile) model.AppError {
	var reader io.Reader
	var thumbnail *utils.Thumbnail
	var ch chan model.AppError
	var err model.AppError
	var ms *model.MalwareScan

	if app.useClamd(&file.BaseFile) {
		ms = &model.MalwareScan{
			Status:   "ERROR",
			ScanDate: nil,
		}

		fn := path.Join(app.Config().TempDir, model.NewId())
		fSrc, errCl := os.Create(fn)
		if errCl != nil {
			ms.Desc = model.NewString(errCl.Error())
			goto endScan
		}

		defer func() {
			_ = fSrc.Close()
			_ = os.Remove(fn)
		}()

		cancel := make(chan bool)
		defer close(cancel)

		rc := io.TeeReader(src, fSrc) // todo подумати використати io_clam, із мінусів: ми будемо генерувати Thumbnail якщо буде потреба

		scanResultChan, errScan := app.clamd.ScanStream(rc, cancel)
		if errScan != nil {
			ms.Desc = model.NewString(errScan.Error())
			goto endScan
		}

		scanResult := <-scanResultChan

		ms.Status = scanResult.Status
		if scanResult.Description != "" {
			ms.Found = true
			ms.Desc = &scanResult.Description
		}

		fSrc.Seek(0, 0)
		src = fSrc
	}

endScan:

	if ms != nil {

		switch app.clamd.mode {
		case ClamavModeAggressive:
			if ms.Status == "ERROR" || ms.Found {
				return FileMalwareErr // todo err
			}

		case ClamavModeSkip:

		default:
			if ms.Found {
				ms.Quarantine = true
			}
		}

		if ms.Found && ms.Desc != nil {
			app.Log.Warn(fmt.Sprintf("virus detected on upload of file '%s'. Signature: %s", file.Name, *ms.Desc))
			if file.UploadedBy != nil { // internal user
				return FileMalwareErr
			}
		}

		file.Malware = ms
	}

	var spool *spoolFile
	if file.GenerateThumbnail && strings.HasPrefix(file.MimeType, model.VideoMimePrefix) {
		// ffmpeg cannot seek a pipe, so a video with its index (moov) at the end
		// yields no frame; the frame is taken from a spooled copy instead.
		f, spoolErr := os.CreateTemp(app.Config().TempDir, "thumb-src-*")
		if spoolErr != nil {
			wlog.Warn(fmt.Sprintf("video thumbnail of '%s' skipped: %s", file.Name, spoolErr))
			reader = src
		} else {
			spool = &spoolFile{f: f}
			defer func() {
				_ = f.Close()
				_ = os.Remove(f.Name())
			}()
			reader = io.TeeReader(src, spool)
		}
	} else if file.GenerateThumbnail {
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

	// A spooled video is probed from the file after upload; audio, or a video
	// whose spool could not be created, is probed from the stream.
	var probe *utils.MediaProbe
	if file.GenerateThumbnail && utils.IsSupportMediaProbe(file.MimeType) && spool == nil {
		var probeErr error
		if probe, probeErr = utils.NewMediaProbe(); probeErr != nil {
			wlog.Warn(fmt.Sprintf("media probe of '%s' not started: %s", file.Name, probeErr))
		} else {
			reader = io.TeeReader(reader, probe)
		}
	}

	// Завантаження основного файлу
	sf, err := app.syncUpload(store, reader, file, profileId)
	if probe != nil {
		d, probeErr := probe.Duration()
		app.applyMediaDuration(file, sf, d, probeErr)
	}
	if err != nil {
		return err
	}
	file.Size = sf.Size

	// Завершення обробки мініатюри, якщо вона існує
	if ch != nil {
		thumbnail.StopWriter()
		if err := <-ch; err != nil {
			return err
		}
		sf.Thumbnail = thumbnail.UserData.(*model.Thumbnail)
		file.Thumbnail = sf.Thumbnail
	}

	if spool != nil {
		if spool.err == nil {
			d, probeErr := utils.ProbeDurationFromFile(spool.f.Name())
			app.applyMediaDuration(file, sf, d, probeErr)
		}

		if sf.Thumbnail, err = app.videoThumbnail(store, spool, file); err != nil {
			return err
		}
		file.Thumbnail = sf.Thumbnail
	}

	file.Id, err = app.storeFile(store, sf)
	if err != nil {
		return err
	}

	return nil
}

// applyMediaDuration is best effort: a file ffprobe cannot read is still stored,
// just without a duration.
func (app *App) applyMediaDuration(file *model.JobUploadFile, sf *model.File, d time.Duration, err error) {
	if err != nil {
		wlog.Warn(fmt.Sprintf("media probe of '%s' failed: %s", file.Name, err))
		return
	}

	if sf == nil {
		return
	}

	// Copy: the thumbnail job shares the original pointer.
	var props model.CustomFileProperties
	if sf.CustomProperties != nil {
		props = *sf.CustomProperties
	}
	props.Duration = int(d.Milliseconds())
	sf.CustomProperties = &props
	file.CustomProperties = &props
}

// videoThumbnail is best effort: an unreadable video is stored without a
// thumbnail rather than with an empty one.
func (app *App) videoThumbnail(store utils.FileBackend, spool *spoolFile, file *model.JobUploadFile) (*model.Thumbnail, model.AppError) {
	if spool.err != nil {
		wlog.Warn(fmt.Sprintf("video thumbnail of '%s' skipped: %s", file.Name, spool.err))
		return nil, nil
	}

	png, scale, err := utils.VideoThumbnailFromFile(spool.f.Name(), app.thumbnailSettings.DefaultScale)
	if err != nil {
		wlog.Warn(fmt.Sprintf("video thumbnail of '%s' failed: %s", file.Name, err))
		return nil, nil
	}

	thumbnailFile := thumbnailJob(file)

	f, appErr := app.syncUpload(store, bytes.NewReader(png), &thumbnailFile, nil)
	if appErr != nil {
		return nil, appErr
	}

	return &model.Thumbnail{BaseFile: f.BaseFile, Scale: scale}, nil
}

// spoolFile never fails the upload it tees: a write error (e.g. a full temp
// disk) only disables the thumbnail.
type spoolFile struct {
	f   *os.File
	err error
}

func (s *spoolFile) Write(p []byte) (int, error) {
	if s.err == nil {
		_, s.err = s.f.Write(p)
	}

	return len(p), nil
}

func thumbnailJob(file *model.JobUploadFile) model.JobUploadFile {
	thumbnailFile := *file
	thumbnailFile.Name = "thumbnail_" + file.Name + ".png"
	thumbnailFile.ViewName = &thumbnailFile.Name
	thumbnailFile.MimeType = "image/png"

	return thumbnailFile
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

	thumbnailFile := thumbnailJob(file)
	ch := make(chan model.AppError, 1)

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
	if file.CreatedAt == 0 {
		file.CreatedAt = model.GetMillis()
	}

	f := &model.File{
		DomainId:  file.DomainId,
		Uuid:      file.Uuid,
		CreatedAt: file.CreatedAt,
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
			Malware:        file.Malware,
		},
		ProfileId: profileId,
	}

	if file.CustomProperties != nil {
		f.CustomProperties = file.CustomProperties
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

	if f.RetentionUntil == nil && store.ExpireDay() > 0 {
		t := time.Now().AddDate(0, 0, store.ExpireDay())
		file.RetentionUntil = &t
		f.RetentionUntil = &t
	}

	return f, nil
}

// storeFile зберігає інформацію про файл у базі даних
func (app *App) storeFile(store utils.FileBackend, file *model.File) (int64, model.AppError) {
	res := <-app.Store.File().Create(file)
	if res.Err != nil {
		return 0, res.Err
	}

	file.Id, _ = res.Data.(int64)

	wlog.Debug(fmt.Sprintf("stored %s in %s, %d bytes [encrypted=%v, SHA256=%v, clamd=%v]", file.GetStoreName(), store.Name(), file.Size, file.IsEncrypted(), file.SHA256Sum != nil, file.BaseFile.StringMalware()))

	// TODO
	if file.Channel != nil && *file.Channel == model.UploadFileChannelCase {
		if notifyErr := app.watcherManager.Notify(
			model.PermissionScopeFiles,
			watcherkit.EventTypeCreate,
			NewFileWatcherData(file),
		); notifyErr != nil {
			wlog.Error(fmt.Sprintf("could not notify file store: %s, ", notifyErr.Error()))
		}
	}
	return file.Id, nil
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
