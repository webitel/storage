package private

import (
	"errors"
	"fmt"
	"github.com/webitel/storage/utils"
	"github.com/webitel/wlog"
	"net/http"
	"strconv"
	"sync"
	"time"
)

var safeAiProcess *utils.Cache = utils.NewLru(4000)

type SafeAiState int

const (
	SafeAiStateActive SafeAiState = iota
	SafeAiStateWait
	SafeAiStateFinished
)

type SafeAi struct {
	id              string
	state           SafeAiState
	cancelSleepChan chan struct{}
	res             *http.Response

	mx sync.RWMutex
}

func StoreSafeAi(id string, res *http.Response, delay time.Duration) {
	ai := &SafeAi{
		id:              id,
		res:             res,
		cancelSleepChan: make(chan struct{}),
	}

	ai.Wait(delay)
}

func RecoverySafeAi(id string) (*SafeAi, error) {
	su, ok := getSafeAiProcess(id)
	if !ok {
		return nil, errors.New("not found " + id)
	}

	if su.State() != SafeAiStateWait {
		return nil, errors.New("ai state " + strconv.Itoa(int(su.State())))
	}
	su.cancelSleep()
	su.setState(SafeAiStateActive)

	return su, nil
}

func (s *SafeAi) Wait(delay time.Duration) {
	s.setState(SafeAiStateWait)
	wlog.Debug(fmt.Sprintf("wait ai id=%s %s", s.id, delay))
	addSafeAiProcess(s)
	s.mx.Lock()
	s.cancelSleepChan = schedule(s.timeout, delay)
	s.mx.Unlock()
}

func (s *SafeAi) Close() {
	if s.res != nil {
		s.res.Body.Close()
		s.res = nil
	}
	safeAiProcess.Remove(s.id)
	wlog.Debug(fmt.Sprintf("destroy ai id=%s", s.id))
}

func (s *SafeAi) timeout() {
	s.Close()
	s.cancelSleepChan = nil
}

func addSafeAiProcess(s *SafeAi) {
	safeAiProcess.Add(s.id, s)
}

func getSafeAiProcess(id string) (*SafeAi, bool) {
	s, ok := safeAiProcess.Get(id)
	if !ok {
		return nil, false
	}

	return s.(*SafeAi), true
}

func (s *SafeAi) Id() string {
	return s.id
}

func (s *SafeAi) cancelSleep() {
	s.mx.Lock()
	if s.cancelSleepChan != nil {
		close(s.cancelSleepChan)
		s.cancelSleepChan = nil
	}
	s.mx.Unlock()
}

func (s *SafeAi) setState(state SafeAiState) {
	s.mx.Lock()
	s.state = state
	s.mx.Unlock()
}

func (s *SafeAi) State() SafeAiState {
	s.mx.RLock()
	state := s.state
	s.mx.RUnlock()
	return state
}

func schedule(what func(), delay time.Duration) chan struct{} {
	stop := make(chan struct{})

	go func() {
		select {
		case <-time.After(delay):
			what()
			return
		case <-stop:
			return
		}
	}()

	return stop
}
