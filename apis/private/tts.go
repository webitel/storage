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

var ttsPerformCache *utils.Cache = utils.NewLru(4000)

type ttsPerform struct {
	id              string
	key             string
	src             io.ReadCloser
	size            *int
	mime            *string
	cancelSleepChan chan struct{}
	mx              sync.RWMutex
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
	wlog.Debug(fmt.Sprintf("[%s] timeout tts", tts.id))
	tts.src.Close()
}

func (tts *ttsPerform) store() {
	tts.cancelSleepChan = schedule(tts.timeout, time.Second*5)
	ttsPerformCache.Add(tts.key, tts)
	wlog.Debug(fmt.Sprintf("[%s] store tts", tts.id))
}

func (api *API) InitTTS() {
	api.Routes.TTS.Handle("/{id}", api.ApiHandler(doTTSByProfile)).Methods("GET")
	api.Routes.TTS.Handle("/", api.ApiHandler(doTTSByProfile)).Methods("GET")
	api.Routes.TTS.Handle("", api.ApiHandler(doTTSByProfile)).Methods("GET")
}

func doTTSByProfile(c *Context, w http.ResponseWriter, r *http.Request) {
	params := TtsParamsFromRequest(r)
	if r.Header.Get("X-TTS-Prepare") == "true" {
		tts := &ttsPerform{
			id:  params.Id,
			key: r.RequestURI,
		}
		tts.src, tts.mime, tts.size, c.Err = c.App.TTS(c.Params.Id, params)
		if c.Err != nil {
			return
		}

		tts.store()
		w.WriteHeader(http.StatusOK)
	} else {
		u, ok := ttsPerformCache.Get(r.RequestURI)
		if ok {
			tts := u.(*ttsPerform)
			tts.stopPerform()
			wlog.Debug(fmt.Sprintf("[%s] play tts", tts.id))

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

	wlog.Debug(fmt.Sprintf("[%s] play tts", params.Id))

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
