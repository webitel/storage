package mimex

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"os"
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

			if is := IsGeneric(tt.mime); is != tt.want {
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

func TestResolveURLUploadMIME_TestData(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                     string
		filepath                 string
		declaredType             string
		clientHint, expectedMIME string
	}{
		{
			name:         "ASiC-E Container (.asice)",
			filepath:     "../../test_data/WTEL-10996.asice",
			expectedMIME: "application/vnd.etsi.asic-e+zip",
		},
		{
			name:         "Legacy Word Document (.doc)",
			filepath:     "../../test_data/WTEL-10996.doc",
			declaredType: "application/msword",
			expectedMIME: "application/msword",
		},
		{
			name:         "Word Document (.docx)",
			filepath:     "../../test_data/WTEL-10996.docx",
			expectedMIME: "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		},
		{
			name:         "Excel Spreadsheet (.xlsx)",
			filepath:     "../../test_data/WTEL-10996.xlsx",
			expectedMIME: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
		},
		{
			name:         "ASiC-E with generic zip header",
			filepath:     "../../test_data/WTEL-10996.asice",
			declaredType: "application/zip",
			expectedMIME: "application/vnd.etsi.asic-e+zip",
		}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			file, err := os.Open(tt.filepath)
			if err != nil {
				t.Errorf("opening file: %v path: %q", err, tt.filepath)
			}
			defer file.Close()

			resp := &http.Response{
				Header: http.Header{},
				Body:   file,
			}

			if tt.declaredType != "" {
				resp.Header.Set("Content-Type", tt.declaredType)
			}

			bodyReaderCloser, mimeType, err := ResolveURLUploadMIME(resp, tt.clientHint)
			if err != nil {
				t.Fatalf("resolving mime: %v", err)
			}

			if mimeType != tt.expectedMIME {
				t.Errorf("expected mime type %q, got %q", tt.expectedMIME, mimeType)
			}

			if bodyReaderCloser == nil {
				t.Error("expected non-nil body reader")
				return
			}

			_ = bodyReaderCloser.Close()
		})
	}
}

func TestDetect(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		raw          []byte
		expectedMIME string
		expectedExt  string
	}{
		{
			name:         "Nil slice returns empty",
			raw:          nil,
			expectedMIME: "",
			expectedExt:  "",
		},
		{
			name:         "Empty slice returns empty",
			raw:          []byte{},
			expectedMIME: "",
			expectedExt:  "",
		},
		{
			name:         "Plain text strips charset parameter via normalize",
			raw:          []byte("Simple UTF-8 plain text content"),
			expectedMIME: "text/plain",
			expectedExt:  ".txt",
		},
		{
			name:         "PNG magic bytes",
			raw:          []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A},
			expectedMIME: "image/png",
			expectedExt:  ".png",
		},
		{
			name:         "Random binary bytes fallback to octet-stream",
			raw:          []byte{0x00, 0xFF, 0x12, 0x34, 0x56, 0x78, 0x9A, 0xBC},
			expectedMIME: "application/octet-stream",
			expectedExt:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			gotMIME, gotExt := Detect(tt.raw)
			if gotMIME != tt.expectedMIME {
				t.Errorf("Detect() mime = %q, want %q", gotMIME, tt.expectedMIME)
			}
			if gotExt != tt.expectedExt {
				t.Errorf("Detect() ext = %q, want %q", gotExt, tt.expectedExt)
			}
		})
	}
}

func TestIsAudio(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		raw      []byte
		expected bool
	}{{
		name:     "Nil slice",
		raw:      nil,
		expected: false,
	}, {
		name:     "Empty slice",
		raw:      []byte{},
		expected: false,
	}, {
		name:     "WAV audio header",
		raw:      []byte("RIFF\x24\x00\x00\x00WAVEfmt "),
		expected: true,
	}, {
		name:     "MP3 raw sync frame (MPEG-1 Layer 3)",
		raw:      []byte{0xFF, 0xFB, 0x90, 0x00},
		expected: true,
	}, {
		name:     "FLAC audio header",
		raw:      []byte("fLaC\x00\x00\x00\x22"),
		expected: true,
	}, {
		name:     "Non-audio RIFF container (WebP image)",
		raw:      []byte("RIFF\x24\x00\x00\x00WEBPVP8 "),
		expected: false,
	}, {
		name:     "PNG image header",
		raw:      []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A},
		expected: false,
	}, {
		name:     "Plain text",
		raw:      []byte("Just some plain text content"),
		expected: false,
	}, {
		name:     "Random binary bytes",
		raw:      []byte{0x00, 0x01, 0x02, 0x03, 0xFA, 0xFB, 0xFC, 0xFD},
		expected: false,
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := IsAudio(tt.raw); got != tt.expected {
				t.Errorf("IsAudio() = %v, want %v", got, tt.expected)
			}
		})
	}
}
