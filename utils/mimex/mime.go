package mimex

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"strings"

	"github.com/gabriel-vasile/mimetype"
)

const (
	mimeOctetStream       = "application/octet-stream"
	mimeBinaryOctetStream = "binary/octet-stream"
	sniffLen              = 512
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

func isGeneric(m string) bool {
	m = strings.TrimSpace(m)

	return m == "" || m == mimeOctetStream || m == mimeBinaryOctetStream
}

func pickMime(sniffed, declared string) string {
	switch {
	case declared == "":
		return sniffed
	case isGeneric(sniffed):
		return declared
	}

	for m := mimetype.Lookup(declared); m != nil; m = m.Parent() {
		if m.Is(sniffed) {
			return declared
		}
	}

	return sniffed
}

func ResolveURLUploadMIME(res *http.Response, clientHint string) (io.ReadCloser, string, error) {
	br := bufio.NewReaderSize(res.Body, sniffLen)

	head, err := br.Peek(sniffLen)
	if err != nil && !errors.Is(err, io.EOF) && !errors.Is(err, io.ErrUnexpectedEOF) {
		return nil, "", fmt.Errorf("%w: %w", ErrReadingHead, err)
	}

	declared := normalize(res.Header.Get("Content-Type"))
	if isGeneric(declared) {
		declared = normalize(clientHint)
	}

	sniffed := normalize(mimetype.Detect(head).String())

	return bodyReader{br, res.Body}, pickMime(sniffed, declared), nil
}
