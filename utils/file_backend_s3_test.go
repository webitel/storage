package utils

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/webitel/storage/model"
)

// An upload's store name is "<uuid>_<id>_" + the client's file name, so
// "../../etc/cron.d/x" must not move the key out of the directory prefix.
func TestS3ObjectKeyPathEscape(t *testing.T) {
	for _, tc := range []struct{ dir, name string }{
		{"1/2026/10/30", "9_abc_/../../../../etc/cron.d/zz"},
		{"1/2026/10/30", "9_abc_/../../tmp/x"},
		{"1/2026/10/30", ".."},
		{"1/../../x", "9_abc_ok.txt"},
		{"", "../x"},
	} {
		if _, err := objectKey(tc.dir, tc.name); err == nil || err.GetId() != ErrPathEscapeId {
			t.Errorf("objectKey(%q, %q): err = %v, want %s", tc.dir, tc.name, err, ErrPathEscapeId)
		}
	}

	for _, tc := range []struct{ dir, name, want string }{
		{"1/2026/10/30", "9_abc_ok.txt", "1/2026/10/30/9_abc_ok.txt"},
		{"1/2026/10/30", "9_abc_a..b.txt", "1/2026/10/30/9_abc_a..b.txt"},
		{"1/2026/10/30", "9_abc_sub/x.txt", "1/2026/10/30/9_abc_sub/x.txt"},
	} {
		key, err := objectKey(tc.dir, tc.name)
		if err != nil || key != tc.want {
			t.Errorf("objectKey(%q, %q) = %q, %v; want %q", tc.dir, tc.name, key, err, tc.want)
		}
	}
}

// The region is prefixed only to a bare host name; an endpoint given as a
// URL is used as is, and the region only signs requests.
func TestS3Endpoint(t *testing.T) {
	for _, tc := range []struct {
		endpoint, region string
		forcePathStyle   bool
		want             string
	}{
		{"https://alac80-rz-prd.halykbank.nb", "alac80", false, "https://alac80-rz-prd.halykbank.nb"},
		{"http://minio:9000", "us-east-1", false, "http://minio:9000"},
		{"https://alac80-rz-prd.halykbank.nb", "alac80", true, "https://alac80-rz-prd.halykbank.nb"},
		{"halykbank.nb", "alac80-rz-prd", false, "alac80-rz-prd.halykbank.nb"},
		{"halykbank.nb", "alac80-rz-prd", true, "halykbank.nb"},
		{"halykbank.nb", "", false, "halykbank.nb"},
		{GoogleStorage, "eu", false, GoogleStorage},
	} {
		b := &S3FileBackend{endpoint: tc.endpoint, region: tc.region, forcePathStyle: tc.forcePathStyle}
		if got := b.getEndpoint(); got == nil || *got != tc.want {
			t.Errorf("getEndpoint(%q, %q, %v) = %v; want %q", tc.endpoint, tc.region, tc.forcePathStyle, got, tc.want)
		}
	}

	if got := (&S3FileBackend{endpoint: "amazonaws.com", region: "eu-west-1"}).getEndpoint(); got != nil {
		t.Errorf("getEndpoint(amazonaws.com) = %q; want nil", *got)
	}
}

// A key that is already in the bucket is not written over.
func TestS3WriteRefusesExistingKey(t *testing.T) {
	var puts int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodHead:
			if strings.HasSuffix(r.URL.Path, "/9_taken.mp3") {
				w.Header().Set("Content-Length", "3")
				w.WriteHeader(http.StatusOK)
				return
			}
			w.WriteHeader(http.StatusNotFound)
		case http.MethodPut:
			puts++
			io.Copy(io.Discard, r.Body)
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer srv.Close()

	be := &S3FileBackend{bucket: "b", endpoint: srv.URL, forcePathStyle: true, region: "us-east-1", accessKey: "k", accessToken: "s"}
	if err := be.Init(); err != nil {
		t.Fatal(err)
	}

	f := &model.JobUploadFile{}
	f.Uuid = "9"
	f.Name = "taken.mp3"
	if _, err := be.Write(strings.NewReader("new"), f); err == nil || err.GetId() != ErrObjectExistsId {
		t.Fatalf("existing key: err = %v, want %s", err, ErrObjectExistsId)
	}
	if puts != 0 {
		t.Fatalf("existing key was written over (%d puts)", puts)
	}

	f.Name = "free.mp3"
	if _, err := be.Write(strings.NewReader("new"), f); err != nil {
		t.Fatalf("free key: %v", err)
	}
	if puts != 1 {
		t.Fatalf("free key: %d puts, want 1", puts)
	}
}
