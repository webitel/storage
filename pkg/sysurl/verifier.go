package sysurl

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/webitel/webitel-go-kit/pkg/presign"
)

type Verifier struct {
	signer presign.Signer
	skew   int64
	now    func() time.Time
}

func NewVerifier(signer presign.Signer, opts ...Option) (*Verifier, error) {
	if signer == nil {
		return nil, ErrNoSigner
	}
	o, err := newOptions(opts)
	if err != nil {
		return nil, err
	}
	return &Verifier{signer: signer, skew: o.skew.Milliseconds(), now: o.now}, nil
}

func (v *Verifier) Verify(route Route, r *http.Request) error {
	if !route.Valid() {
		return fmt.Errorf("%w: %q", ErrUnknownRoute, route)
	}
	query, err := url.ParseQuery(r.URL.RawQuery)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrMalformedQuery, err)
	}

	expiresValues, sigValues := query[ParamExpires], query[ParamSignature]
	if len(expiresValues) == 0 || len(sigValues) == 0 {
		return ErrMissingSignature
	}
	if len(expiresValues) > 1 || len(sigValues) > 1 {
		return fmt.Errorf("%w: expires or signature", ErrDuplicateParam)
	}
	expires, err := strconv.ParseInt(expiresValues[0], 10, 64)
	if err != nil {
		return fmt.Errorf("%w: expires", ErrMalformedQuery)
	}
	if v.now().UnixMilli()-v.skew > expires {
		return ErrExpired
	}

	signed := make(url.Values, len(query))
	for key, values := range query {
		if isUnsignedParam(route, key, values) {
			continue
		}
		if len(values) != 1 {
			return fmt.Errorf("%w: %q", ErrDuplicateParam, key)
		}
		signed[key] = values
	}

	if !v.signer.Valid(string(canonical(route, r.URL.Path, expires, signed)), sigValues[0]) {
		return ErrInvalidSignature
	}
	return nil
}

func Strip(r *http.Request) {
	if r.URL.RawQuery == "" {
		return
	}
	parts := strings.Split(r.URL.RawQuery, "&")
	kept := parts[:0]
	for _, part := range parts {
		key, _, _ := strings.Cut(part, "=")
		if k, err := url.QueryUnescape(key); err == nil && (k == ParamExpires || k == ParamSignature) {
			continue
		}
		kept = append(kept, part)
	}
	r.URL.RawQuery = strings.Join(kept, "&")
}
