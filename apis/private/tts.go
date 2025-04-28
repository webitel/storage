package private

import (
	"fmt"
	. "github.com/webitel/storage/apis/helper"
	"github.com/webitel/storage/utils"
	"github.com/webitel/wlog"
	"io"
	"net/http"
	"strconv"
	"sync"
	"time"
)

var ttsPerformCache = utils.NewLru(4000)

type ttsPerform struct {
	id              string
	requestId       string
	callId          string
	key             string
	src             io.ReadCloser
	size            *int
	mime            *string
	cancelSleepChan chan struct{}
	mx              sync.RWMutex
}

func (tts *ttsPerform) String() string {
	return fmt.Sprintf("%s/%s/%s", tts.callId, tts.id, tts.requestId)
}

func (tts *ttsPerform) stopPerform() {
	tts.mx.Lock()
	if tts.cancelSleepChan != nil {
		close(tts.cancelSleepChan)
		tts.cancelSleepChan = nil
	}
	tts.mx.Unlock()
	ttsPerformCache.Remove(tts.key)
}

func (tts *ttsPerform) timeout() {
	tts.stopPerform()
	wlog.Debug(fmt.Sprintf("[%s] timeout tts", tts))
	tts.src.Close()
}

func (tts *ttsPerform) store() {
	tts.cancelSleepChan = schedule(tts.timeout, time.Second*5)
	if _, ok := ttsPerformCache.Get(tts.key); ok {
		wlog.Error(fmt.Sprintf("[%s] tts key in cache", tts))
		return
	}
	ttsPerformCache.Add(tts.key, tts)
}

func (api *API) InitTTS() {
	api.Routes.TTS.Handle("/{id}", api.ApiHandler(doTTSByProfile)).Methods("GET")
	api.Routes.TTS.Handle("/", api.ApiHandler(doTTSByProfile)).Methods("GET")
	api.Routes.TTS.Handle("", api.ApiHandler(doTTSByProfile)).Methods("GET")
}

func doTTSByProfile(c *Context, w http.ResponseWriter, r *http.Request) {
	wlog.Debug(fmt.Sprintf("[%s] start %s", c.RequestId, r.RequestURI))
	params := TtsParamsFromRequest(r)
	if r.Header.Get("X-TTS-Prepare") == "true" {
		t := time.Now()
		tts := &ttsPerform{
			id:        r.Header.Get("X-TTS-Prepare-Id"),
			callId:    r.Header.Get("X-TTS-Call-Id"),
			requestId: c.RequestId,
			key:       r.RequestURI,
		}
		tts.src, tts.mime, tts.size, c.Err = c.App.TTS(c.Params.Id, params)
		if c.Err != nil {
			wlog.Debug(fmt.Sprintf("[%s] store tts error: %s, duration %v", tts, c.Err.Error(), time.Since(t)))
			return
		}

		tts.store()
		wlog.Debug(fmt.Sprintf("[%s] store tts, generate duration %v", tts, time.Since(t)))
		w.WriteHeader(http.StatusOK)
	} else {
		u, ok := ttsPerformCache.Get(r.RequestURI)
		if ok {
			tts := u.(*ttsPerform)
			tts.stopPerform()
			wlog.Debug(fmt.Sprintf("[%s] play tts", tts))

			defer tts.src.Close()
			if tts.mime != nil {
				w.Header().Set("Content-Type", *tts.mime)
			}
			if tts.size != nil {
				w.Header().Set("Content-Length", strconv.Itoa(*tts.size))
			}

			w.WriteHeader(http.StatusOK)
			if params.Format == "mp3" {
				ttsCopy(w, tts.src)
			} else {
				io.Copy(w, tts.src)
			}
		} else {
			ttsByProfile(c, w, r)
		}
	}

}

func ttsByProfile(c *Context, w http.ResponseWriter, r *http.Request) {
	params := TtsParamsFromRequest(r)

	out, t, size, err := c.App.TTS(c.Params.Id, params)
	if err != nil {
		c.Err = err
		return
	}

	defer out.Close()

	if t != nil {
		w.Header().Set("Content-Type", *t)
	}
	if size != nil {
		w.Header().Set("Content-Length", strconv.Itoa(*size))
	}

	wlog.Debug(fmt.Sprintf("[%s] play tts", c.RequestId))

	if params.Format == "mp3" {
		ttsCopy(w, out)
	} else {
		io.Copy(w, out)
	}
}

func ttsCopy(dst io.Writer, src io.Reader) {
	buf := make([]byte, 8192/2) // SWITCH_RECOMMENDED_BUFFER_SIZE / 2

	var n int
	var err2 error
	for {
		n, _ = src.Read(buf)
		if n <= 0 {
			return
		}
		_, err2 = dst.Write(buf[:n])
		if err2 != nil {
			return
		}
	}
}
