package sysurl

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/webitel/webitel-go-kit/pkg/presign"
)

var reservedHeaders = map[string]struct{}{
	"url":          {},
	"method":       {},
	ParamExpires:   {},
	ParamSignature: {},
}

// Builder creates signed /sys links. Each method returns a relative URL
// (escaped path plus query) that the caller joins with the storage base URL.
//
// Expiry is bucketed. The expires parameter is the current time rounded up to
// the next bucket boundary plus one more bucket, in Unix milliseconds. With
// the default 24h bucket a link lives between 24h and 48h. All links for the
// same input built within one bucket are identical, so downstream caches
// (for example the FreeSWITCH HTTP cache) keep hitting until the bucket
// changes. Set the bucket with WithBucket.
//
// Every signed parameter must have exactly one value. A signed key must not
// be expires, signature, a route-specific unsigned key (name on
// RouteRecordingUpload) or a format key that starts with a dot.
//
// The optional format argument is appended unsigned as an empty ".<format>"
// parameter, which FreeSWITCH uses to pick the file format. It may contain only
// lowercase ASCII letters and digits; an empty format adds nothing.
type Builder struct {
	signer presign.Signer
	bucket int64
	now    func() time.Time
}

// NewBuilder returns a Builder that signs with signer. It returns
// ErrNoSigner for a nil signer and ErrInvalidOption for a bad option.
func NewBuilder(signer presign.Signer, opts ...Option) (*Builder, error) {
	if signer == nil {
		return nil, ErrNoSigner
	}
	o, err := newOptions(opts)
	if err != nil {
		return nil, err
	}
	return &Builder{signer: signer, bucket: o.bucket.Milliseconds(), now: o.now}, nil
}

// RecordingUpload builds a RouteRecordingUpload link to POST /sys/recordings.
// It signs domain (domainID), id and every extra parameter. The name
// parameter and format are not signed. An empty id is rejected.
//
// The name is written unescaped so FreeSWITCH can expand channel variables
// such as ${strepoch()} before the upload. The caller must escape any literal
// text in name that is not safe in a query value.
func (b *Builder) RecordingUpload(domainID int64, id, name, format string, extra url.Values) (string, error) {
	if id == "" {
		return "", fmt.Errorf("%w: empty id", ErrInvalidArgument)
	}
	signed := cloneValues(extra)
	signed.Set("domain", strconv.FormatInt(domainID, 10))
	signed.Set("id", id)
	return b.build(RouteRecordingUpload, "/sys/recordings", signed, name, format)
}

// AiUpload builds a RouteAiUpload link to /sys/recordings/ai/{id}. It signs
// every parameter in params. The id must be non-empty and contain no slash.
func (b *Builder) AiUpload(id string, params url.Values, format string) (string, error) {
	path, err := pathWithID("/sys/recordings/ai/", id, "")
	if err != nil {
		return "", err
	}
	return b.build(RouteAiUpload, path, cloneValues(params), "", format)
}

// AiSpeech builds a RouteAiSpeech link to /sys/recordings/ai/{id}. Only the
// path and expiry are signed. The id must be non-empty and contain no slash.
func (b *Builder) AiSpeech(id, format string) (string, error) {
	path, err := pathWithID("/sys/recordings/ai/", id, "")
	if err != nil {
		return "", err
	}
	return b.build(RouteAiSpeech, path, url.Values{}, "", format)
}

// AiMetadata builds a RouteAiMetadata link to
// /sys/recordings/ai/{id}/metadata. Only the path and expiry are signed. The
// id must be non-empty and contain no slash.
func (b *Builder) AiMetadata(id string) (string, error) {
	path, err := pathWithID("/sys/recordings/ai/", id, "/metadata")
	if err != nil {
		return "", err
	}
	return b.build(RouteAiMetadata, path, url.Values{}, "", "")
}

// MediaStream builds a RouteMediaStream link to /sys/media/{fileID}/stream.
// It signs domain_id (domainID); fileID is signed as part of the path.
func (b *Builder) MediaStream(domainID, fileID int64, format string) (string, error) {
	path := "/sys/media/" + strconv.FormatInt(fileID, 10) + "/stream"
	signed := url.Values{"domain_id": {strconv.FormatInt(domainID, 10)}}
	return b.build(RouteMediaStream, path, signed, "", format)
}

// TTS builds a RouteTTS link to /sys/tts, or to /sys/tts/{profileID} when
// profileID is set. It signs every parameter in params. A non-empty
// profileID must contain no slash.
func (b *Builder) TTS(profileID string, params url.Values, format string) (string, error) {
	path := "/sys/tts"
	if profileID != "" {
		var err error
		if path, err = pathWithID("/sys/tts/", profileID, ""); err != nil {
			return "", err
		}
	}
	return b.build(RouteTTS, path, cloneValues(params), "", format)
}

// RedirectPlayback builds a RouteRedirectPlayback link to
// /sys/redirect/playback. It signs url (target), method when set, and each
// header as its own query parameter. Header names url, method, expires,
// signature, empty names and names that start with a dot are rejected, as is
// an empty target.
func (b *Builder) RedirectPlayback(target, method string, headers map[string]string, format string) (string, error) {
	if target == "" {
		return "", fmt.Errorf("%w: empty url", ErrInvalidArgument)
	}
	signed := url.Values{"url": {target}}
	if method != "" {
		signed.Set("method", method)
	}
	for name, value := range headers {
		if _, ok := reservedHeaders[name]; ok || name == "" || name[0] == '.' {
			return "", fmt.Errorf("%w: header %q", ErrInvalidArgument, name)
		}
		signed.Set(name, value)
	}
	return b.build(RouteRedirectPlayback, "/sys/redirect/playback", signed, "", format)
}

func (b *Builder) build(route Route, path string, signed url.Values, name, format string) (string, error) {
	for key, values := range signed {
		if len(values) != 1 || isUnsignedParam(route, key, values) {
			return "", fmt.Errorf("%w: parameter %q", ErrInvalidArgument, key)
		}
	}
	if !validFormat(format) {
		return "", fmt.Errorf("%w: format %q", ErrInvalidArgument, format)
	}

	expires := b.expires()
	sig, err := b.signer.Generate(canonical(route, path, expires, signed))
	if err != nil {
		return "", err
	}

	var s strings.Builder
	s.WriteString((&url.URL{Path: path}).EscapedPath())
	s.WriteByte('?')
	if q := signed.Encode(); q != "" {
		s.WriteString(q)
		s.WriteByte('&')
	}
	s.WriteString(ParamExpires + "=")
	s.WriteString(strconv.FormatInt(expires, 10))
	s.WriteString("&" + ParamSignature + "=")
	s.WriteString(sig)
	if name != "" {
		s.WriteString("&" + paramName + "=")
		s.WriteString(name)
	}
	if format != "" {
		s.WriteString("&.")
		s.WriteString(format)
	}
	return s.String(), nil
}

func (b *Builder) expires() int64 {
	now := b.now().UnixMilli()
	return (now+b.bucket-1)/b.bucket*b.bucket + b.bucket
}

func pathWithID(prefix, id, suffix string) (string, error) {
	if id == "" || strings.ContainsRune(id, '/') {
		return "", fmt.Errorf("%w: id %q", ErrInvalidArgument, id)
	}
	return prefix + id + suffix, nil
}

func validFormat(format string) bool {
	for _, c := range format {
		if (c < 'a' || c > 'z') && (c < '0' || c > '9') {
			return false
		}
	}
	return true
}

func cloneValues(v url.Values) url.Values {
	out := make(url.Values, len(v)+2)
	for key, values := range v {
		out[key] = append([]string(nil), values...)
	}
	return out
}
