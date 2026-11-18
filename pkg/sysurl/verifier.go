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

// Verifier checks links created by a Builder that uses the same key.
type Verifier struct {
	signer presign.Signer
	skew   int64
	now    func() time.Time
}

// NewVerifier returns a Verifier that checks signatures with signer. It
// returns ErrNoSigner for a nil signer and ErrInvalidOption for a bad option.
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

// Verify checks that r carries a valid, unexpired signature for route. It
// rebuilds the signed data from route, r.URL.Path, the expires parameter and
// every query parameter except expires, signature, the route's unsigned keys
// and empty format keys. A link stays valid until expires plus the skew.
// Errors wrap ErrUnknownRoute, ErrMalformedQuery, ErrMissingSignature,
// ErrDuplicateParam, ErrExpired or ErrInvalidSignature.
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

// Strip removes the expires and signature parameters from r.URL.RawQuery and
// keeps the other parameters in their original order and encoding. Call it
// after Verify so handlers and proxied requests do not see the signature.
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
