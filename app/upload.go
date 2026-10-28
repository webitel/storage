package app

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"os"
	"path"
	"strings"
	"time"

	"github.com/hashicorp/golang-lru/v2/expirable"
	"golang.org/x/sync/singleflight"

	"github.com/webitel/webitel-go-kit/pkg/logging"
	watcherkit "github.com/webitel/webitel-go-kit/pkg/watcher"

	"github.com/webitel/storage/model"
	"github.com/webitel/storage/utils"
)

var (
	domainProfileCache = expirable.NewLRU[int64, utils.FileBackend](100, nil, time.Second*15)
	sgDomainProfile    = singleflight.Group{}
)

// AddUploadJobFile додає файл до черги завантаження
func (app *App) AddUploadJobFile(ctx context.Context, src io.Reader, file *model.JobUploadFile) model.AppError {
	size, err := app.FileCache.Write(src, file)
	if err != nil {
		return err
	}

	file.Size = size
	file.Instance = app.GetInstanceId()

	file, err = app.Store.UploadJob().Create(file)
	if err != nil {
		app.logger.ErrorCtx(ctx, "create upload job", logging.Str("call_id", file.Uuid), logging.Err(err))
		if errRem := app.FileCache.Remove(file); errRem != nil {
			app.logger.ErrorCtx(ctx, "remove cache file", logging.Str("call_id", file.Uuid), logging.Err(errRem))
		}
	} else {
		app.logger.DebugCtx(ctx, "upload job created",
			logging.Int64("job_id", file.Id),
			logging.Str("call_id", file.Uuid),
			logging.Str("file_name", file.Name),
			logging.Int64("size", file.Size),
			logging.Str("mime_type", file.MimeType),
		)
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
func (app *App) SyncUpload(ctx context.Context, src io.Reader, file *model.JobUploadFile) model.AppError {
	store, err := app.domainStore(file.DomainId)
	if err != nil {
		return err
	}

	return app.upload(ctx, src, store.Id(), store, file)
}

// SyncUploadToProfile синхронно завантажує файл у профіль користувача
func (app *App) SyncUploadToProfile(ctx context.Context, src io.Reader, profileId int, file *model.JobUploadFile) model.AppError {
	store, err := app.GetFileBackendStoreById(file.DomainId, profileId)
	if err != nil {
		return err
	}

	return app.upload(ctx, src, &profileId, store, file)
}

func (app *App) useClamd(file *model.BaseFile) bool {
	return app.clamd != nil && file.Channel != nil && *file.Channel == model.UploadFileChannelChat
}

func (app *App) upload(ctx context.Context, src io.Reader, profileId *int, store utils.FileBackend, file *model.JobUploadFile) model.AppError {
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
			app.logger.WarnCtx(ctx, "virus detected on upload", logging.Str("file_name", file.Name), logging.Str("signature", *ms.Desc))
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
			app.logger.WarnCtx(ctx, "video thumbnail skipped", logging.Str("file_name", file.Name), logging.Err(spoolErr))
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
			app.logger.WarnCtx(ctx, "media probe not started", logging.Str("file_name", file.Name), logging.Err(probeErr))
		} else {
			reader = io.TeeReader(reader, probe)
		}
	}

	// Завантаження основного файлу
	sf, err := app.syncUpload(store, reader, file, profileId)
	if probe != nil {
		d, probeErr := probe.Duration()
		app.applyMediaDuration(ctx, file, sf, d, probeErr)
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
			app.applyMediaDuration(ctx, file, sf, d, probeErr)
		}

		if sf.Thumbnail, err = app.videoThumbnail(ctx, store, spool, file); err != nil {
			return err
		}
		file.Thumbnail = sf.Thumbnail
	}

	file.Id, err = app.storeFile(ctx, store, sf)
	if err != nil {
		return err
	}

	return nil
}

// applyMediaDuration is best effort: a file ffprobe cannot read is still stored,
// just without a duration.
func (app *App) applyMediaDuration(ctx context.Context, file *model.JobUploadFile, sf *model.File, d time.Duration, err error) {
	if err != nil {
		app.logger.WarnCtx(ctx, "media probe failed", logging.Str("file_name", file.Name), logging.Err(err))
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
func (app *App) videoThumbnail(ctx context.Context, store utils.FileBackend, spool *spoolFile, file *model.JobUploadFile) (*model.Thumbnail, model.AppError) {
	if spool.err != nil {
		app.logger.WarnCtx(ctx, "video thumbnail skipped", logging.Str("file_name", file.Name), logging.Err(spool.err))
		return nil, nil
	}

	png, scale, err := utils.VideoThumbnailFromFile(spool.f.Name(), app.thumbnailSettings.DefaultScale)
	if err != nil {
		app.logger.WarnCtx(ctx, "video thumbnail failed", logging.Str("file_name", file.Name), logging.Err(err))
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
	thumbnailFile.Properties = maps.Clone(file.Properties)

	delete(thumbnailFile.Properties, "location")
	delete(thumbnailFile.Properties, "directory")

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
func (app *App) storeFile(ctx context.Context, store utils.FileBackend, file *model.File) (int64, model.AppError) {
	res := <-app.Store.File().Create(file)
	if res.Err != nil {
		return 0, res.Err
	}

	file.Id, _ = res.Data.(int64)

	app.logger.DebugCtx(ctx, "file stored",
		logging.Int64("file_id", file.Id),
		logging.Str("file_name", file.GetStoreName()),
		logging.Str("store", store.Name()),
		logging.Int64("size", file.Size),
		logging.Bool("encrypted", file.IsEncrypted()),
		logging.Bool("sha256", file.SHA256Sum != nil),
		logging.Str("clamd", file.BaseFile.StringMalware()),
	)

	// TODO
	if file.Channel != nil && *file.Channel == model.UploadFileChannelCase {
		if notifyErr := app.watcherManager.Notify(
			model.PermissionScopeFiles,
			watcherkit.EventTypeCreate,
			NewFileWatcherData(file),
		); notifyErr != nil {
			app.logger.ErrorCtx(ctx, "notify file watchers", logging.Int64("file_id", file.Id), logging.Err(notifyErr))
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
