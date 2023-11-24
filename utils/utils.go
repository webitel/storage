package utils

import (
	"github.com/h2non/filetype"
	"io"
	"net"
	"net/http"
	"strings"

	"github.com/pkg/errors"

	"github.com/webitel/storage/model"
)

var (
	ErrorMaxLimit      = errors.New("max size")
	ErrorExtUnknown    = errors.New("extension of file is unknown")
	ErrorExtSuspicious = errors.New("actual file extension doesn't match declared Content-Type")
	ErrorExtNotAllowed = errors.New("file extension is not allowed")
)

const (
	minMagicBytesLength = 100
)

func GetIpAddress(r *http.Request) string {
	address := ""

	header := r.Header.Get(model.HEADER_FORWARDED)
	if len(header) > 0 {
		addresses := strings.Fields(header)
		if len(addresses) > 0 {
			address = strings.TrimRight(addresses[0], ",")
		}
	}

	if len(address) == 0 {
		address = r.Header.Get(model.HEADER_REAL_IP)
	}

	if len(address) == 0 {
		address, _, _ = net.SplitHostPort(r.RemoteAddr)
	}

	return address
}

// NewSecureReader returns a Reader that reads from r
// but stops with EOF after n bytes.
// The underlying implementation is a *NewSecureReader.
func NewSecureReader(r io.Reader, n int64, declaredFileMime string, allowedMime []string) io.Reader {
	return &SecureReader{R: r, N: n, allowedMimes: allowedMime, declaredFileMime: declaredFileMime}
}

// A NewSecureReader reads from R but limits the amount of
// data returned to just N bytes. Each call to Read
// updates N to reflect the new amount remaining.
// Read returns EOF when N <= 0 or when the underlying R returns EOF.
type SecureReader struct {
	R                io.Reader // underlying reader
	N                int64     // max bytes remaining
	allowedMimes     []string  // allowed mime types
	declaredFileMime string    // declared type of file
	securityChecked  bool      // determines if file checked for security purpose
}

func (l *SecureReader) Read(p []byte) (n int, err error) {
	if l.N <= 0 {
		return 0, io.EOF
	}
	if int64(len(p)) > l.N {
		return n, ErrorMaxLimit
	}

	n, err = l.R.Read(p)
	if err != nil {
		return
	}
	err = l.securityCheck(p)
	if err != nil {
		return 0, err
	}
	l.N -= int64(n)
	return
}

func (l *SecureReader) securityCheck(p []byte) error {
	if !l.securityChecked {
		l.securityChecked = true
		err := l.checkMimeType(p)
		if err != nil {
			return err
		}
	}

	return nil
}

func (l *SecureReader) checkMimeType(mBytes []byte) error {
	kind, err := filetype.Match(mBytes)
	if err != nil {
		return err
	}

	if kind == filetype.Unknown {
		return ErrorExtUnknown
	}

	if l.declaredFileMime != kind.MIME.Value { // Header Content-Type doesn't match magic bytes (! suspicious !)
		return ErrorExtSuspicious
	}

	for _, mime := range l.allowedMimes {
		if mime == l.declaredFileMime || mime == "*" {
			return nil
		}
	}
	// File mime type is not in the allowed list
	return ErrorExtNotAllowed
}
