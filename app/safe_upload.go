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
)

var safeUploadProcess *utils.Cache = utils.NewLru(4000)

type SafeUploadState int

const (
	SafeUploadStateActive SafeUploadState = iota
	SafeUploadStateSleep
	SafeUploadStateFinished
)

type SafeUpload struct {
	id        string
	state     SafeUploadState
	app       *App
	reader    *io.PipeReader
	writer    *io.PipeWriter
	backend   utils.FileBackend
	request   *model.JobUploadFile
	profileId *int
	end       chan struct{}
	err       chan error
	size      int
	mx        sync.RWMutex
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

func (s *SafeUpload) setError(err error) {
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

	if err != nil {
		wlog.Error(err.String())
	}
	s.setState(SafeUploadStateFinished)
	wlog.Debug(fmt.Sprintf("finished safe upload id=%s, name=%s, size=%d", s.id, s.request.Name, s.Size()))
}

func (s *SafeUpload) Sleep() {
	s.setState(SafeUploadStateSleep)
	wlog.Debug(fmt.Sprintf("sleep safe upload id=%s, name=%s, size=%d", s.id, s.request.Name, s.Size()))
	addSafeUploadProcess(s)
}

func addSafeUploadProcess(s *SafeUpload) {
	safeUploadProcess.Add(s.id, s)
}

func RecoverySafeUploadProcess(id string) (*SafeUpload, error) {
	su, ok := getSafeUploadProcess(id)
	if !ok {
		return nil, errors.New("not found " + id)
	}

	if su.State() != SafeUploadStateSleep {
		return nil, errors.New("upload state " + strconv.Itoa(int(su.State())))
	}
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
		end:       make(chan struct{}),
		err:       make(chan error),
		request:   req,
		reader:    r,
		writer:    w,
	}

	go s.run()

	return s
}
