package sysurl

import (
	"errors"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type Route string

const (
	RouteRecordingUpload  Route = "recordings.upload"
	RouteAiUpload         Route = "recordings.ai.upload"
	RouteAiSpeech         Route = "recordings.ai.speech"
	RouteAiMetadata       Route = "recordings.ai.metadata"
	RouteMediaStream      Route = "media.stream"
	RouteTTS              Route = "tts"
	RouteRedirectPlayback Route = "redirect.playback"
)

const (
	ParamExpires   = "expires"
	ParamSignature = "signature"

	DefaultBucket = 24 * time.Hour
	DefaultSkew   = time.Minute

	canonicalTag = "webitel.sysurl.v1"
	paramName    = "name"
)

var (
	ErrNoSigner         = errors.New("sysurl: signer is required")
	ErrInvalidOption    = errors.New("sysurl: invalid option")
	ErrInvalidArgument  = errors.New("sysurl: invalid argument")
	ErrUnknownRoute     = errors.New("sysurl: unknown route")
	ErrMalformedQuery   = errors.New("sysurl: malformed query")
	ErrMissingSignature = errors.New("sysurl: missing expires or signature")
	ErrDuplicateParam   = errors.New("sysurl: duplicate parameter")
	ErrExpired          = errors.New("sysurl: link expired")
	ErrInvalidSignature = errors.New("sysurl: invalid signature")
)

// routes is the complete list of all URLs with their isUnsignedParam params
var routes = map[Route]map[string]struct{}{
	RouteRecordingUpload:  {paramName: {}},
	RouteAiUpload:         nil,
	RouteAiSpeech:         nil,
	RouteAiMetadata:       nil,
	RouteMediaStream:      nil,
	RouteTTS:              nil,
	RouteRedirectPlayback: nil,
}

func Routes() []Route {
	return []Route{
		RouteRecordingUpload,
		RouteAiUpload,
		RouteAiSpeech,
		RouteAiMetadata,
		RouteMediaStream,
		RouteTTS,
		RouteRedirectPlayback,
	}
}

func (r Route) Valid() bool {
	_, ok := routes[r]
	return ok
}

type Option func(*options)

type options struct {
	bucket time.Duration
	skew   time.Duration
	now    func() time.Time
}

func WithBucket(d time.Duration) Option {
	return func(o *options) { o.bucket = d }
}

func WithSkew(d time.Duration) Option {
	return func(o *options) { o.skew = d }
}

func WithClock(now func() time.Time) Option {
	return func(o *options) { o.now = now }
}

func newOptions(opts []Option) (options, error) {
	o := options{bucket: DefaultBucket, skew: DefaultSkew, now: time.Now}
	for _, opt := range opts {
		opt(&o)
	}
	if o.bucket < time.Millisecond || o.skew < 0 || o.now == nil {
		return o, ErrInvalidOption
	}
	return o, nil
}

func isUnsignedParam(route Route, key string, values []string) bool {
	if key == ParamExpires || key == ParamSignature {
		return true
	}
	if _, ok := routes[route][key]; ok {
		return true
	}
	return isFormatKey(key, values)
}

func isFormatKey(key string, values []string) bool {
	if len(key) < 2 || key[0] != '.' {
		return false
	}
	for _, v := range values {
		if v != "" {
			return false
		}
	}
	return true
}

func canonical(route Route, path string, expires int64, signed url.Values) []byte {
	var b strings.Builder
	b.WriteString(canonicalTag)
	b.WriteByte('\n')
	b.WriteString(string(route))
	b.WriteByte('\n')
	b.WriteString(path)
	b.WriteByte('\n')
	b.WriteString(strconv.FormatInt(expires, 10))
	b.WriteByte('\n')
	b.WriteString(signed.Encode())
	return []byte(b.String())
}
