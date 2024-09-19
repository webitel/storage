package app

import (
	"fmt"
	"github.com/pkg/errors"
	engine "github.com/webitel/engine/model"
	"github.com/webitel/storage/model"
	"github.com/webitel/storage/utils"
	"github.com/webitel/wlog"
	"io"
	"strconv"
	"sync"
	"time"
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
	reader          *io.PipeReader
	writer          *io.PipeWriter
	backend         utils.FileBackend
	request         *model.JobUploadFile
	profileId       *int
	cancelSleepChan chan struct{}
	err             chan error // todo
	uploaded        chan struct{}
	size            int
	mx              sync.RWMutex
	Progress        bool
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

func (s *SafeUpload) Write(src []byte) error {
	n, err := s.writer.Write(src)
	s.mx.Lock()
	s.size += n
	s.mx.Unlock()
	return err
}

func (s *SafeUpload) run() {
	wlog.Debug(fmt.Sprintf("start safe upload id=%s, name=%s", s.id, s.request.Name))
	var err engine.AppError
	if s.profileId != nil {
		err = s.app.SyncUploadToProfile(s.reader, *s.profileId, s.request)
	} else {
		err = s.app.SyncUpload(s.reader, s.request)
	}

	s.setState(SafeUploadStateFinished)
	if err != nil {
		wlog.Debug(fmt.Sprintf("finished safe upload id=%s, name=%s, error=%s", s.id, s.request.Name, err.GetDetailedError()))
	} else {
		wlog.Debug(fmt.Sprintf("finished safe upload id=%s, name=%s, size=%d", s.id, s.request.Name, s.Size()))
	}
	close(s.uploaded)
}

func addSafeUploadProcess(s *SafeUpload) {
	safeUploadProcess.Add(s.id, s)
}

func (s *SafeUpload) timeout() {
	s.SetError(errors.New("timeout"))
	s.cancelSleepChan = nil
}

func (s *SafeUpload) Sleep() {
	s.setState(SafeUploadStateSleep)
	wlog.Debug(fmt.Sprintf("sleep safe upload id=%s, name=%s, size=%d", s.id, s.request.Name, s.Size()))
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

func RecoverySafeUploadProcess(id string) (*SafeUpload, error) {
	su, ok := getSafeUploadProcess(id)
	if !ok {
		return nil, errors.New("not found " + id)
	}

	if su.State() != SafeUploadStateSleep {
		return nil, errors.New("upload state " + strconv.Itoa(int(su.State())))
	}
	su.setState(SafeUploadStateActive)
	su.cancelSleep()
	wlog.Debug(fmt.Sprintf("recovery upload id=%s, name=%s, size=%d", su.id, su.request.Name, su.Size()))

	return su, nil
}

func getSafeUploadProcess(id string) (*SafeUpload, bool) {
	s, ok := safeUploadProcess.Get(id)
	if !ok {
		return nil, false
	}

	return s.(*SafeUpload), true
}

func (app *App) NewSafeUpload(profileId *int, req *model.JobUploadFile) *SafeUpload {
	return newSafeUpload(app, profileId, req)
}

func newSafeUpload(app *App, profileId *int, req *model.JobUploadFile) *SafeUpload {
	r, w := io.Pipe()
	if profileId != nil {
		req.Name = fmt.Sprintf("%s_%s", model.NewId()[0:7], req.Name)
	}
	s := &SafeUpload{
		app:       app,
		state:     SafeUploadStateActive,
		profileId: profileId,
		id:        model.NewId(),
		err:       make(chan error),
		uploaded:  make(chan struct{}),
		request:   req,
		reader:    r,
		writer:    w,
	}

	go s.run()

	return s
}

func schedule(what func(), delay time.Duration) chan struct{} {
	stop := make(chan struct{})

	go func() {
		for {
			select {
			case <-time.After(delay):
				what()
			case <-stop:
				return
			}
		}
	}()

	return stop
}
