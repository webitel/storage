package apis

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/webitel/storage/model"
	"github.com/webitel/storage/utils"
)

type memBackend struct {
	utils.FileBackend
	data    []byte
	offsets []int64
}

func (b *memBackend) Reader(_ utils.File, offset int64) (io.ReadCloser, model.AppError) {
	b.offsets = append(b.offsets, offset)

	return io.NopCloser(bytes.NewReader(b.data[offset:])), nil
}

func TestWriteFileBody(t *testing.T) {
	data := []byte("0123456789")

	cases := []struct {
		name        string
		method      string
		rangeHeader string
		wantCode    int
		wantBody    string
		wantLength  string
		wantRange   string
		wantReads   int
	}{
		{"no range is unchanged", http.MethodGet, "", http.StatusOK, "0123456789", "10", "", 1},
		{"single range", http.MethodGet, "bytes=2-5", http.StatusPartialContent, "2345", "4", "bytes 2-5/10", 1},
		{"suffix range", http.MethodGet, "bytes=-3", http.StatusPartialContent, "789", "3", "bytes 7-9/10", 1},
		{"open range", http.MethodGet, "bytes=8-", http.StatusPartialContent, "89", "2", "bytes 8-9/10", 1},
		{"malformed range ignored", http.MethodGet, "items=1-2", http.StatusOK, "0123456789", "10", "", 1},
		{"multi range serves whole file", http.MethodGet, "bytes=0-1,4-5", http.StatusOK, "0123456789", "10", "", 1},
		{"head reads nothing", http.MethodHead, "", http.StatusOK, "", "10", "", 0},
		{"head with range", http.MethodHead, "bytes=0-0", http.StatusPartialContent, "", "1", "bytes 0-0/10", 0},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			backend := &memBackend{data: data}
			file := &model.File{BaseFile: model.BaseFile{Name: "v.mp4", MimeType: "video/mp4", Size: int64(len(data))}}

			r := httptest.NewRequest(tc.method, "/any/file/download", nil)
			if tc.rangeHeader != "" {
				r.Header.Set("Range", tc.rangeHeader)
			}

			w := httptest.NewRecorder()
			writeFileBody(&Context{}, w, r, file, backend, "v.mp4")

			if w.Code != tc.wantCode {
				t.Errorf("code = %d, want %d", w.Code, tc.wantCode)
			}

			if got := w.Body.String(); got != tc.wantBody {
				t.Errorf("body = %q, want %q", got, tc.wantBody)
			}

			if got := w.Header().Get("Content-Length"); got != tc.wantLength {
				t.Errorf("Content-Length = %q, want %q", got, tc.wantLength)
			}

			if got := w.Header().Get("Content-Range"); got != tc.wantRange {
				t.Errorf("Content-Range = %q, want %q", got, tc.wantRange)
			}

			if w.Header().Get("Accept-Ranges") != "bytes" || w.Header().Get("Content-Type") != "video/mp4" {
				t.Errorf("headers = %v", w.Header())
			}

			if len(backend.offsets) != tc.wantReads {
				t.Errorf("backend reads = %d, want %d", len(backend.offsets), tc.wantReads)
			}
		})
	}
}
