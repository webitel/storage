package mimex

import (
	"archive/zip"
	"bytes"
	"errors"
	"io"
	"net/http"
	"testing"
)

func TestIsGeneric(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name, mime string
		want       bool
	}{
		{name: "Empty input mime should return true", mime: "", want: true},
		{name: "Octet-stream should return true", mime: mimeOctetStream, want: true},
		{name: "Binary octet-stream should return true", mime: mimeBinaryOctetStream, want: true},
		{name: "Non generic mime type should return false", mime: "image/jpeg", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if is := isGeneric(tt.mime); is != tt.want {
				t.Errorf("isGeneric() want: %t\ngot: %t", tt.want, is)
			}
		})
	}
}

func TestNormalize(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name, input, want string
	}{
		{name: "Simple format", input: "image/png", want: "image/png"},
		{name: "Format with params", input: "text/html; charset=utf-8", want: "text/html"},
		{name: "Invalid mime type", input: "invalid mime type", want: ""},
		{name: "Empty string", input: "", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := normalize(tt.input); got != tt.want {
				t.Errorf("normalize() want: %q\ngot: %q", tt.want, got)
			}
		})
	}
}

func TestPickMime(t *testing.T) {
	t.Parallel()

	const (
		docx = "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
		xlsx = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
		zip  = "application/zip"
	)

	tests := []struct {
		name, sniffed, declared, want string
	}{
		{name: "Declared is empty, fallback to sniffed", sniffed: "image/png", declared: "", want: "image/png"},
		{name: "Sniffed is application/octet-stream, fallback to declared", sniffed: "application/octet-stream", declared: "application/pdf", want: "application/pdf"},
		{name: "Sniffed is binary/octet-stream, fallback to declared", sniffed: "binary/octet-stream", declared: "text/plain", want: "text/plain"},
		{name: "Sniffed is empty string, fallback to declared", sniffed: "", declared: "application/json", want: "application/json"},
		{name: "Exact match between sniffed and declared", sniffed: "image/jpeg", declared: "image/jpeg", want: "image/jpeg"},
		{name: "Declared is parent of sniffed (zip is parent of docx)", sniffed: docx, declared: zip, want: docx},
		{name: "Complete mismatch, trust sniffed over suspicious declared", sniffed: "image/png", declared: "application/pdf", want: "image/png"},
		{name: "Pure DOCX detection when header is missing", sniffed: docx, declared: "", want: docx},
		{name: "Spoofing protection: declared Word, but sniffed Excel", sniffed: xlsx, declared: docx, want: xlsx},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := pickMime(tt.sniffed, tt.declared); got != tt.want {
				t.Errorf("pickMime(%q, %q) got: %q\nwant: %q", tt.sniffed, tt.declared, got, tt.want)
			}
		})
	}
}

type failingReadCloser struct{ err error }

func (f failingReadCloser) Read(p []byte) (int, error) { return 0, f.err }
func (f failingReadCloser) Close() error               { return nil }

func TestResolveURLUploadMIME(t *testing.T) {
	t.Parallel()

	pngMagicBytes := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}

	t.Run("Success: Normal flow with valid Content-Type and matching body", func(t *testing.T) {
		t.Parallel()

		resp := &http.Response{
			Header: http.Header{
				"Content-Type": []string{"image/png"},
			},
			Body: io.NopCloser(bytes.NewReader(pngMagicBytes)),
		}

		bodyReaderCloser, mimeType, err := ResolveURLUploadMIME(resp, "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if mimeType != "image/png" {
			t.Errorf("expected mime type 'image/png', got %q", mimeType)
		}

		if bodyReaderCloser == nil {
			t.Error("expected non-nil body reader")
		}

		readBytes, err := io.ReadAll(bodyReaderCloser)
		if err != nil {
			t.Fatalf("failed to read from bodyReader: %v", err)
		}

		if !bytes.Equal(readBytes, pngMagicBytes) {
			t.Errorf("body content corrupted or lost during peeking")
		}

		_ = bodyReaderCloser.Close()
	})

	t.Run("Success: Generic Content-Type falls back to clientHint", func(t *testing.T) {
		t.Parallel()

		resp := &http.Response{
			Header: http.Header{
				"Content-Type": []string{mimeOctetStream},
			},
			Body: io.NopCloser(bytes.NewReader([]byte("some random text data"))),
		}

		_, mimeType, err := ResolveURLUploadMIME(resp, "application/json")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if mimeType == "" {
			t.Errorf("expected resolved mime type, got empty string")
		}
	})

	t.Run("Success: Body shorter than sniffLen (triggers io.EOF safely)", func(t *testing.T) {
		t.Parallel()

		shortBody := []byte("short")

		resp := &http.Response{
			Header: http.Header{
				"Content-Type": []string{"text/plain"},
			},
			Body: io.NopCloser(bytes.NewReader(shortBody)),
		}

		_, mimeType, err := ResolveURLUploadMIME(resp, "")
		if err != nil {
			t.Fatalf("expected no error on short body, got %v", err)
		}

		if mimeType != "text/plain" {
			t.Errorf("expected 'text/plain', got %q", mimeType)
		}
	})

	t.Run("Error: Reading head fails with underlying IO error", func(t *testing.T) {
		t.Parallel()

		expectedIOErr := errors.New("connection reset by peer")
		resp := &http.Response{
			Header: http.Header{},
			Body:   failingReadCloser{err: expectedIOErr},
		}

		_, _, err := ResolveURLUploadMIME(resp, "")
		if err == nil {
			t.Fatal("expected error, got nil")
		}

		if !errors.Is(err, ErrReadingHead) {
			t.Errorf("expected error to wrap %v, got %v", ErrReadingHead, err)
		}

		if !errors.Is(err, expectedIOErr) {
			t.Errorf("expected error to wrap underlying %v, got %v", expectedIOErr, err)
		}
	})
}

func TestResolveURLUploadMIME_RealWordDocument(t *testing.T) {
	var buf bytes.Buffer

	zipWriter := zip.NewWriter(&buf)

	if _, err := zipWriter.Create("word/document.xml"); err != nil {
		t.Fatalf("failed to create zip entry for docx mock: %v", err)
	}

	if err := zipWriter.Close(); err != nil {
		t.Fatalf("failed to close zip writer: %v", err)
	}

	resp := &http.Response{
		Header: http.Header{},
		Body:   io.NopCloser(&buf),
	}

	bodyReaderCloser, mimeType, err := ResolveURLUploadMIME(resp, "")
	if err != nil {
		t.Fatalf("unexpected error resolving docx mime: %v", err)
	}

	expectedMime := "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
	if mimeType != expectedMime {
		t.Errorf("expected mime type %q, got %q", expectedMime, mimeType)
	}

	if bodyReaderCloser == nil {
		t.Error("expected non-nil body reader for docx")
	}

	_ = bodyReaderCloser.Close()
}
