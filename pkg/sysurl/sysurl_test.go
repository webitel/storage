package sysurl_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/webitel/crypto/cryptobox"
	"github.com/webitel/storage/pkg/sysurl"
	"github.com/webitel/webitel-go-kit/pkg/presign"
)

const testKeyring = "sysurl-test-phrase-2026!"

var testNow = time.Date(2026, 11, 14, 10, 30, 0, 0, time.UTC)

func signer(t *testing.T, spec string) presign.Signer {
	t.Helper()
	keys, err := cryptobox.ParseKeyring(spec)
	if err != nil {
		t.Fatalf("ParseKeyring: %v", err)
	}
	s, err := presign.NewMACKeyring(keys)
	if err != nil {
		t.Fatalf("NewMACKeyring: %v", err)
	}
	return s
}

func clock(at time.Time) sysurl.Option {
	return sysurl.WithClock(func() time.Time { return at })
}

func builder(t *testing.T, at time.Time) *sysurl.Builder {
	t.Helper()
	b, err := sysurl.NewBuilder(signer(t, testKeyring), clock(at))
	if err != nil {
		t.Fatalf("NewBuilder: %v", err)
	}
	return b
}

func verifier(t *testing.T, at time.Time) *sysurl.Verifier {
	t.Helper()
	v, err := sysurl.NewVerifier(signer(t, testKeyring), clock(at))
	if err != nil {
		t.Fatalf("NewVerifier: %v", err)
	}
	return v
}

func mustLink(t *testing.T) func(string, error) string {
	return func(link string, err error) string {
		t.Helper()
		if err != nil {
			t.Fatalf("build: %v", err)
		}
		return link
	}
}

func request(method, link string) *http.Request {
	return httptest.NewRequest(method, "http://storage:10021"+link, nil)
}

func TestRoundTrip(t *testing.T) {
	must := mustLink(t)
	b := builder(t, testNow)
	v := verifier(t, testNow)

	cases := []struct {
		name   string
		route  sysurl.Route
		method string
		link   string
	}{
		{"recording upload", sysurl.RouteRecordingUpload, http.MethodPut,
			must(b.RecordingUpload(1, "4b1c0c5e-2b57-4d0b-9f83-0d6c6d1c9a10", "1763116200_%2B380501234567.mp3", "mp3",
				url.Values{"email_msg": {"hello"}, "email_sbj": {"subject"}, "thumbnail": {"true"}}))},
		{"recording upload without name", sysurl.RouteRecordingUpload, http.MethodPost,
			must(b.RecordingUpload(1, "4b1c0c5e", "", "", nil))},
		{"ai upload", sysurl.RouteAiUpload, http.MethodPut,
			must(b.AiUpload("a1b2c3", url.Values{
				"domain":       {"1"},
				"id":           {"a1b2c3"},
				"url":          {"https://ai.example.com/v1/speech?x=1&y=2"},
				"addresses":    {"YWRkcg=="},
				"context":      {"Y3R4"},
				"chat_history": {"W10="},
			}, "wav"))},
		{"ai speech", sysurl.RouteAiSpeech, http.MethodGet, must(b.AiSpeech("a1b2c3", "wav"))},
		{"ai metadata", sysurl.RouteAiMetadata, http.MethodGet, must(b.AiMetadata("a1b2c3"))},
		{"media stream", sysurl.RouteMediaStream, http.MethodGet, must(b.MediaStream(1, 42, "mp3"))},
		{"tts", sysurl.RouteTTS, http.MethodGet,
			must(b.TTS("", url.Values{"domain_id": {"1"}, "profile_id": {"7"}, "text": {"Hello, world ${x} & more"}, "r": {"abcde"}}, "opus"))},
		{"tts profile", sysurl.RouteTTS, http.MethodGet,
			must(b.TTS("7", url.Values{"domain_id": {"1"}, "text": {"hi"}}, "mp3"))},
		{"redirect playback", sysurl.RouteRedirectPlayback, http.MethodGet,
			must(b.RedirectPlayback("https://audio.example.com/a.mp3", "GET", map[string]string{"Authorization": "Bearer x"}, "mp3"))},
		{"redirect playback no method", sysurl.RouteRedirectPlayback, http.MethodGet,
			must(b.RedirectPlayback("https://audio.example.com/a.wav", "", nil, "wav"))},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := v.Verify(tc.route, request(tc.method, tc.link)); err != nil {
				t.Fatalf("Verify(%s): %v", tc.link, err)
			}
		})
	}
}

func TestFreeSWITCHSafe(t *testing.T) {
	must := mustLink(t)
	b := builder(t, testNow)
	link := must(b.TTS("", url.Values{"text": {"Hello, world ${caller_id_number} {x} 'q'"}}, "wav"))
	if i := strings.Index(link, "?"); i >= 0 {
		for _, c := range []string{",", " ", "$", "{", "}", "'"} {
			if strings.Contains(link[i:], c) {
				t.Fatalf("link query contains %q: %s", c, link)
			}
		}
	}
}

func TestLinkLayout(t *testing.T) {
	must := mustLink(t)
	b := builder(t, testNow)
	link := must(b.RecordingUpload(3, "abc", "${strepoch()}.mp3", "mp3", nil))
	if !strings.HasPrefix(link, "/sys/recordings?domain=3&id=abc&expires=") {
		t.Fatalf("unexpected prefix: %s", link)
	}
	if !strings.HasSuffix(link, "&name=${strepoch()}.mp3&.mp3") {
		t.Fatalf("name and format must follow the signature unescaped: %s", link)
	}
}

func TestDeterministicWithinBucket(t *testing.T) {
	must := mustLink(t)
	first := must(builder(t, testNow).MediaStream(1, 42, "wav"))
	later := must(builder(t, testNow.Add(3*time.Hour)).MediaStream(1, 42, "wav"))
	if first != later {
		t.Fatalf("links differ inside one bucket:\n%s\n%s", first, later)
	}
	next := must(builder(t, testNow.Add(24*time.Hour)).MediaStream(1, 42, "wav"))
	if first == next {
		t.Fatal("link must change in the next bucket")
	}
}

func TestExpires(t *testing.T) {
	must := mustLink(t)
	link := must(builder(t, testNow).MediaStream(1, 42, ""))
	u, _ := url.Parse(link)
	got, err := strconv.ParseInt(u.Query().Get(sysurl.ParamExpires), 10, 64)
	if err != nil {
		t.Fatal(err)
	}
	day := time.Date(2026, 11, 16, 0, 0, 0, 0, time.UTC).UnixMilli()
	if got != day {
		t.Fatalf("expires = %d, want %d", got, day)
	}

	boundary := time.Date(2026, 11, 14, 0, 0, 0, 0, time.UTC)
	link = must(builder(t, boundary).MediaStream(1, 42, ""))
	u, _ = url.Parse(link)
	if got := u.Query().Get(sysurl.ParamExpires); got != strconv.FormatInt(boundary.Add(24*time.Hour).UnixMilli(), 10) {
		t.Fatalf("expires on boundary = %s", got)
	}
}

func TestExpiry(t *testing.T) {
	must := mustLink(t)
	link := must(builder(t, testNow).MediaStream(1, 42, "mp3"))
	u, _ := url.Parse(link)
	expires, _ := strconv.ParseInt(u.Query().Get(sysurl.ParamExpires), 10, 64)
	at := time.UnixMilli(expires)

	if err := verifier(t, at.Add(59*time.Second)).Verify(sysurl.RouteMediaStream, request(http.MethodGet, link)); err != nil {
		t.Fatalf("inside skew: %v", err)
	}
	err := verifier(t, at.Add(61*time.Second)).Verify(sysurl.RouteMediaStream, request(http.MethodGet, link))
	if !errors.Is(err, sysurl.ErrExpired) {
		t.Fatalf("after skew: %v", err)
	}
}

func TestRejected(t *testing.T) {
	must := mustLink(t)
	b := builder(t, testNow)
	v := verifier(t, testNow)

	media := must(b.MediaStream(1, 42, "mp3"))
	speech := must(b.AiSpeech("a1b2c3", "wav"))
	recording := must(b.RecordingUpload(1, "abc", "rec.mp3", "mp3", nil))
	redirect := must(b.RedirectPlayback("https://audio.example.com/a.mp3", "GET", nil, "mp3"))

	cases := []struct {
		name  string
		route sysurl.Route
		link  string
		want  error
	}{
		{"other domain", sysurl.RouteMediaStream, strings.Replace(media, "domain_id=1", "domain_id=2", 1), sysurl.ErrInvalidSignature},
		{"other file", sysurl.RouteMediaStream, strings.Replace(media, "/42/", "/43/", 1), sysurl.ErrInvalidSignature},
		{"extra param", sysurl.RouteMediaStream, media + "&profile_id=9", sysurl.ErrInvalidSignature},
		{"format key with value", sysurl.RouteMediaStream, media + "=x", sysurl.ErrInvalidSignature},
		{"duplicate signed param", sysurl.RouteMediaStream, media + "&domain_id=1", sysurl.ErrDuplicateParam},
		{"duplicate signature", sysurl.RouteMediaStream, media + "&signature=00", sysurl.ErrDuplicateParam},
		{"other route same signature", sysurl.RouteAiUpload, speech + "&url=http://169.254.169.254/", sysurl.ErrInvalidSignature},
		{"route swap", sysurl.RouteAiMetadata, speech, sysurl.ErrInvalidSignature},
		{"name on other route", sysurl.RouteMediaStream, media + "&name=x", sysurl.ErrInvalidSignature},
		{"injected header", sysurl.RouteRedirectPlayback, redirect + "&X-Forwarded-For=10.0.0.1", sysurl.ErrInvalidSignature},
		{"injected url", sysurl.RouteRedirectPlayback, strings.Replace(redirect, "audio.example.com", "10.0.0.7", 1), sysurl.ErrInvalidSignature},
		{"name smuggles domain", sysurl.RouteRecordingUpload, strings.Replace(recording, "name=rec.mp3", "name=rec.mp3&domain=2", 1), sysurl.ErrDuplicateParam},
		{"missing signature", sysurl.RouteMediaStream, "/sys/media/42/stream?domain_id=1&.mp3", sysurl.ErrMissingSignature},
		{"bad expires", sysurl.RouteMediaStream, "/sys/media/42/stream?domain_id=1&expires=x&signature=00", sysurl.ErrMalformedQuery},
		{"malformed query", sysurl.RouteMediaStream, media + "&a=%zz", sysurl.ErrMalformedQuery},
		{"unknown route", sysurl.Route("files.delete"), media, sysurl.ErrUnknownRoute},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := v.Verify(tc.route, request(http.MethodGet, tc.link))
			if !errors.Is(err, tc.want) {
				t.Fatalf("Verify(%s) = %v, want %v", tc.link, err, tc.want)
			}
		})
	}
}

func TestUnsignedRecordingName(t *testing.T) {
	must := mustLink(t)
	link := must(builder(t, testNow).RecordingUpload(1, "abc", "${strepoch()}_${url_encode ${wbt_from_number}}.mp3", "mp3", nil))
	expanded := strings.Replace(link, "${strepoch()}_${url_encode ${wbt_from_number}}", "1763116200_%2B380501234567", 1)
	if err := verifier(t, testNow).Verify(sysurl.RouteRecordingUpload, request(http.MethodPut, expanded)); err != nil {
		t.Fatalf("expanded name: %v", err)
	}
}

func TestOtherKeyRejected(t *testing.T) {
	must := mustLink(t)
	link := must(builder(t, testNow).MediaStream(1, 42, "mp3"))
	other, err := sysurl.NewVerifier(signer(t, "another-test-phrase-2026!"), clock(testNow))
	if err != nil {
		t.Fatal(err)
	}
	if err := other.Verify(sysurl.RouteMediaStream, request(http.MethodGet, link)); !errors.Is(err, sysurl.ErrInvalidSignature) {
		t.Fatalf("other key: %v", err)
	}
}

func TestBuilderRejects(t *testing.T) {
	b := builder(t, testNow)
	cases := map[string]func() (string, error){
		"name in extra":       func() (string, error) { return b.RecordingUpload(1, "abc", "", "", url.Values{"name": {"x"}}) },
		"expires in extra":    func() (string, error) { return b.RecordingUpload(1, "abc", "", "", url.Values{"expires": {"1"}}) },
		"format key in extra": func() (string, error) { return b.TTS("", url.Values{".mp3": {""}}, "") },
		"multi value":         func() (string, error) { return b.TTS("", url.Values{"text": {"a", "b"}}, "") },
		"empty recording id":  func() (string, error) { return b.RecordingUpload(1, "", "", "", nil) },
		"empty ai id":         func() (string, error) { return b.AiSpeech("", "wav") },
		"slash in id":         func() (string, error) { return b.AiMetadata("a/b") },
		"bad format":          func() (string, error) { return b.MediaStream(1, 42, "mp3&x=1") },
		"empty url":           func() (string, error) { return b.RedirectPlayback("", "", nil, "mp3") },
		"reserved header": func() (string, error) {
			return b.RedirectPlayback("https://a", "", map[string]string{"signature": "x"}, "mp3")
		},
		"header url": func() (string, error) {
			return b.RedirectPlayback("https://a", "", map[string]string{"url": "x"}, "mp3")
		},
		"dot header": func() (string, error) {
			return b.RedirectPlayback("https://a", "", map[string]string{".wav": ""}, "mp3")
		},
	}
	for name, build := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := build(); !errors.Is(err, sysurl.ErrInvalidArgument) {
				t.Fatalf("err = %v, want ErrInvalidArgument", err)
			}
		})
	}
}

func TestConstructors(t *testing.T) {
	if _, err := sysurl.NewBuilder(nil); !errors.Is(err, sysurl.ErrNoSigner) {
		t.Fatalf("NewBuilder(nil) = %v", err)
	}
	if _, err := sysurl.NewVerifier(nil); !errors.Is(err, sysurl.ErrNoSigner) {
		t.Fatalf("NewVerifier(nil) = %v", err)
	}
	s := signer(t, testKeyring)
	if _, err := sysurl.NewBuilder(s, sysurl.WithBucket(0)); !errors.Is(err, sysurl.ErrInvalidOption) {
		t.Fatalf("zero bucket = %v", err)
	}
	if _, err := sysurl.NewVerifier(s, sysurl.WithSkew(-time.Second)); !errors.Is(err, sysurl.ErrInvalidOption) {
		t.Fatalf("negative skew = %v", err)
	}
}

func TestRoutes(t *testing.T) {
	for _, r := range sysurl.Routes() {
		if !r.Valid() {
			t.Fatalf("route %q not valid", r)
		}
	}
	if sysurl.Route("unknown").Valid() {
		t.Fatal("unknown route must not be valid")
	}
}

func TestStrip(t *testing.T) {
	must := mustLink(t)
	link := must(builder(t, testNow).RecordingUpload(1, "abc", "${strepoch()}.mp3", "mp3", nil))
	r := request(http.MethodPut, link)
	sysurl.Strip(r)
	if want := "domain=1&id=abc&name=${strepoch()}.mp3&.mp3"; r.URL.RawQuery != want {
		t.Fatalf("RawQuery = %q, want %q", r.URL.RawQuery, want)
	}

	r = request(http.MethodGet, "/sys/redirect/playback?url=x&expire%73=1&%73ignature=2&Authorization=y")
	sysurl.Strip(r)
	if want := "url=x&Authorization=y"; r.URL.RawQuery != want {
		t.Fatalf("escaped keys: RawQuery = %q, want %q", r.URL.RawQuery, want)
	}

	r = request(http.MethodGet, "/sys/media/1/stream")
	sysurl.Strip(r)
	if r.URL.RawQuery != "" {
		t.Fatalf("empty query changed: %q", r.URL.RawQuery)
	}
}
