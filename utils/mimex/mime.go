package mimex

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"strings"

	"github.com/gabriel-vasile/mimetype"
)

func init() {
	mimetype.Lookup("application/zip").Extend(isASiCE, "application/vnd.etsi.asic-e+zip", ".asice")
}

func isASiCE(raw []byte, _ uint32) bool {
	return len(raw) > 30 && bytes.HasPrefix(raw[30:], []byte("mimetypeapplication/vnd.etsi.asic-e+zip"))
}

const (
	mimeOctetStream       = "application/octet-stream"
	mimeBinaryOctetStream = "binary/octet-stream"
	sniffLen              = 4096
)

var ErrReadingHead = errors.New("mime: could not read head of body")

type bodyReader struct {
	io.Reader
	io.Closer
}

func normalize(s string) string {
	m, _, err := mime.ParseMediaType(s)
	if err != nil && !errors.Is(err, mime.ErrInvalidMediaParameter) {
		return ""
	}

	return m
}

func IsGeneric(m string) bool {
	m = strings.TrimSpace(m)

	return m == "" || m == mimeOctetStream || m == mimeBinaryOctetStream
}

func pickMime(sniffed, declared string) string {
	switch {
	case declared == "":
		return sniffed
	case IsGeneric(sniffed):
		return declared
	}

	if Matches(declared, sniffed) {
		return declared
	}

	return sniffed
}

func Matches(declared, detected string) bool {
	for m := mimetype.Lookup(declared); m != nil; m = m.Parent() {
		if m.Is(detected) {
			return true
		}
	}

	return false
}

func IsAudio(raw []byte) bool {
	if len(raw) == 0 {
		return false
	}

	mt := mimetype.Detect(raw)

	return strings.HasPrefix(mt.String(), "audio/")
}

func Detect(raw []byte) (string, string) {
	if len(raw) == 0 {
		return "", ""
	}

	mt := mimetype.Detect(raw)

	return normalize(mt.String()), mt.Extension()
}

func ResolveURLUploadMIME(res *http.Response, clientHint string) (io.ReadCloser, string, error) {
	br := bufio.NewReaderSize(res.Body, sniffLen)

	head, err := br.Peek(sniffLen)
	if err != nil && !errors.Is(err, io.EOF) && !errors.Is(err, io.ErrUnexpectedEOF) {
		return nil, "", fmt.Errorf("%w: %w", ErrReadingHead, err)
	}

	declared := normalize(res.Header.Get("Content-Type"))
	if IsGeneric(declared) {
		declared = normalize(clientHint)
	}

	sniffed := normalize(mimetype.Detect(head).String())

	return bodyReader{br, res.Body}, pickMime(sniffed, declared), nil
}
