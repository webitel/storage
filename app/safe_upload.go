package app

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strconv"
	"sync"
	"time"

	"github.com/webitel/webitel-go-kit/pkg/logging"

	"github.com/webitel/storage/model"
	"github.com/webitel/storage/utils"
)

var safeUploadProcess *utils.Cache = utils.NewLru(4000)

type SafeUploadState int

const (
	SafeUploadStateActive SafeUploadState = iota
	SafeUploadStateSleep
	SafeUploadStateFinished
)

type SafeUpload struct {
	id              string
	state           SafeUploadState
	app             *App
	reader          io.Reader
	writer          *io.PipeWriter
	backend         utils.FileBackend
	request         *model.JobUploadFile
	profileId       *int
	cancelSleepChan chan struct{}
	err             model.AppError
	uploaded        chan struct{}
	size            int
	mx              sync.RWMutex
	Progress        bool
	logger          *logging.Logger
}

func (s *SafeUpload) Id() string {
	return s.id
}

func (s *SafeUpload) Size() int {
	s.mx.RLock()
	size := s.size
	s.mx.RUnlock()
	return size
}

func (s *SafeUpload) setState(state SafeUploadState) {
	s.mx.Lock()
	s.state = state
	s.mx.Unlock()
}

func (s *SafeUpload) State() SafeUploadState {
	s.mx.RLock()
	state := s.state
	s.mx.RUnlock()
	return state
}

func (s *SafeUpload) File() *model.JobUploadFile {
	return s.request
}

func (s *SafeUpload) CloseWrite() {
	s.writer.Close()
	s.destruct()
}

func (s *SafeUpload) destruct() {
	safeUploadProcess.Remove(s.id)
}

func (s *SafeUpload) SetError(err error) {
	s.writer.CloseWithError(err)
	s.destruct()
}

func (s *SafeUpload) Err() error {
	return s.err
}

func (s *SafeUpload) Write(src []byte) error {
	if s.err != nil {
		return s.err
	}
	n, err := s.writer.Write(src)
	s.mx.Lock()
	s.size += n
	s.mx.Unlock()
	return err
}

func (s *SafeUpload) run(ctx context.Context) {
	s.logger.DebugCtx(ctx, "safe upload started")

	var err model.AppError
	if s.profileId != nil {
		err = s.app.SyncUploadToProfile(ctx, s.reader, *s.profileId, s.request)
	} else {
		err = s.app.SyncUpload(ctx, s.reader, s.request)
	}

	s.setState(SafeUploadStateFinished)
	if err != nil {
		s.mx.Lock()
		s.err = err
		s.mx.Unlock()

		s.logger.DebugCtx(ctx, "safe upload finished", logging.Str("err_details", err.GetDetailedError()))
	} else {
		s.logger.DebugCtx(ctx, "safe upload finished", logging.Int("size", s.Size()))
	}
	close(s.uploaded)
	s.writer.Close()
}

func addSafeUploadProcess(s *SafeUpload) {
	safeUploadProcess.Add(s.id, s)
}

func (s *SafeUpload) timeout() {
	s.SetError(errors.New("timeout"))
	s.cancelSleepChan = nil
}

func (s *SafeUpload) Sleep(ctx context.Context) {
	s.setState(SafeUploadStateSleep)
	s.logger.DebugCtx(ctx, "safe upload sleep", logging.Int("size", s.Size()))
	addSafeUploadProcess(s)
	s.mx.Lock()
	s.cancelSleepChan = schedule(s.timeout, s.app.Config().MaxSafeUploadSleep)
	s.mx.Unlock()
}

func (s *SafeUpload) SetProgress(v bool) {
	s.Progress = v
}

func (s *SafeUpload) cancelSleep() {
	s.mx.Lock()
	if s.cancelSleepChan != nil {
		close(s.cancelSleepChan)
		s.cancelSleepChan = nil
	}
	s.mx.Unlock()
}

func (s *SafeUpload) WaitUploaded() chan struct{} {
	return s.uploaded
}

func RecoverySafeUploadProcess(ctx context.Context, id string) (*SafeUpload, error) {
	su, ok := getSafeUploadProcess(id)
	if !ok {
		return nil, errors.New("not found " + id)
	}

	if su.State() == SafeUploadStateActive {
		// TODO buffered write ?
		select {
		case <-time.After(time.Second):
		case <-ctx.Done():
		}
	}

	if su.State() != SafeUploadStateSleep {
		return nil, errors.New("upload state " + strconv.Itoa(int(su.State())))
	}
	su.setState(SafeUploadStateActive)
	su.cancelSleep()
	su.logger.DebugCtx(ctx, "safe upload recovered", logging.Int("size", su.Size()))

	return su, nil
}

func getSafeUploadProcess(id string) (*SafeUpload, bool) {
	s, ok := safeUploadProcess.Get(id)
	if !ok {
		return nil, false
	}

	return s.(*SafeUpload), true
}

func (app *App) NewSafeUpload(ctx context.Context, profileId *int, req *model.JobUploadFile) (*SafeUpload, error) {
	return newSafeUpload(ctx, app, profileId, req)
}

func newSafeUpload(ctx context.Context, app *App, profileId *int, req *model.JobUploadFile) (*SafeUpload, error) {
	var err model.AppError
	r, w := io.Pipe()
	if profileId != nil {
		req.Name = fmt.Sprintf("%s_%s", model.NewId()[0:7], req.Name)
	}

	s := &SafeUpload{
		app:       app,
		state:     SafeUploadStateActive,
		profileId: profileId,
		id:        model.NewId(),
		err:       nil,
		uploaded:  make(chan struct{}),
		request:   req,
		writer:    w,
	}
	s.logger = app.logger.With(logging.Str("upload_id", s.id), logging.Str("file_name", req.Name))
	s.reader, err = app.FilePolicyForUpload(ctx, req.DomainId, &req.BaseFile, r)
	if err != nil {
		return nil, err
	}

	go s.run(context.WithoutCancel(ctx))

	return s, nil
}

func schedule(what func(), delay time.Duration) chan struct{} {
	stop := make(chan struct{})

	go func() {
		select {
		case <-time.After(delay):
			what()
		case <-stop:
			return
		}
	}()

	return stop
}
