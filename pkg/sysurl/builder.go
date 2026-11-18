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

type Builder struct {
	signer presign.Signer
	bucket int64
	now    func() time.Time
}

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

func (b *Builder) RecordingUpload(domainID int64, id, name, format string, extra url.Values) (string, error) {
	if id == "" {
		return "", fmt.Errorf("%w: empty id", ErrInvalidArgument)
	}
	signed := cloneValues(extra)
	signed.Set("domain", strconv.FormatInt(domainID, 10))
	signed.Set("id", id)
	return b.build(RouteRecordingUpload, "/sys/recordings", signed, name, format)
}

func (b *Builder) AiUpload(id string, params url.Values, format string) (string, error) {
	path, err := pathWithID("/sys/recordings/ai/", id, "")
	if err != nil {
		return "", err
	}
	return b.build(RouteAiUpload, path, cloneValues(params), "", format)
}

func (b *Builder) AiSpeech(id, format string) (string, error) {
	path, err := pathWithID("/sys/recordings/ai/", id, "")
	if err != nil {
		return "", err
	}
	return b.build(RouteAiSpeech, path, url.Values{}, "", format)
}

func (b *Builder) AiMetadata(id string) (string, error) {
	path, err := pathWithID("/sys/recordings/ai/", id, "/metadata")
	if err != nil {
		return "", err
	}
	return b.build(RouteAiMetadata, path, url.Values{}, "", "")
}

func (b *Builder) MediaStream(domainID, fileID int64, format string) (string, error) {
	path := "/sys/media/" + strconv.FormatInt(fileID, 10) + "/stream"
	signed := url.Values{"domain_id": {strconv.FormatInt(domainID, 10)}}
	return b.build(RouteMediaStream, path, signed, "", format)
}

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
