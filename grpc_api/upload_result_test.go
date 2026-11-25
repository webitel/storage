package grpc_api

import (
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/webitel/storage/model"
)

type fakeUpload struct {
	done chan struct{}
	err  error
	file *model.JobUploadFile
}

func (f *fakeUpload) WaitUploaded() chan struct{} { return f.done }
func (f *fakeUpload) Err() error                  { return f.err }
func (f *fakeUpload) File() *model.JobUploadFile  { return f.file }

func finished(err error, file *model.JobUploadFile) *fakeUpload {
	done := make(chan struct{})
	close(done)

	return &fakeUpload{done: done, err: err, file: file}
}

func TestUploadedFile(t *testing.T) {
	t.Parallel()

	stored := &model.JobUploadFile{Id: 42}
	writeErr := model.NewInternalError("utils.file.s3.writing.origin", "no such host")

	tests := []struct {
		name     string
		upload   *fakeUpload
		wantFile *model.JobUploadFile
		wantErr  error
	}{
		{name: "stored", upload: finished(nil, stored), wantFile: stored},
		{
			// The write failed after the last chunk: the error, not a file without an id.
			name:    "write failed",
			upload:  finished(writeErr, &model.JobUploadFile{}),
			wantErr: writeErr,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			file, err := uploadedFile(tc.upload)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("err = %v, want %v", err, tc.wantErr)
			}

			if file != tc.wantFile {
				t.Errorf("file = %+v, want %+v", file, tc.wantFile)
			}
		})
	}
}

// The error must reach the gRPC interceptor as an AppError so it maps to a status.
func TestUploadedFile_KeepsAppError(t *testing.T) {
	t.Parallel()

	_, err := uploadedFile(finished(model.NewInternalError("utils.file.s3.writing.origin", "no such host"), nil))

	appErr, ok := err.(model.AppError) //nolint:errorlint // the interceptor type-switches on the value itself
	if !ok {
		t.Fatalf("err = %T, want model.AppError", err)
	}

	if appErr.GetStatusCode() != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", appErr.GetStatusCode(), http.StatusInternalServerError)
	}
}

func TestUploadedFile_WaitsForUpload(t *testing.T) {
	t.Parallel()

	up := &fakeUpload{done: make(chan struct{}), file: &model.JobUploadFile{Id: 7}}
	got := make(chan *model.JobUploadFile, 1)

	go func() {
		f, _ := uploadedFile(up)
		got <- f
	}()

	select {
	case <-got:
		t.Fatal("uploadedFile returned before the upload finished")
	case <-time.After(50 * time.Millisecond):
	}

	close(up.done)

	select {
	case f := <-got:
		if f.Id != 7 {
			t.Errorf("file id = %d, want 7", f.Id)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("uploadedFile did not return after the upload finished")
	}
}

func TestUploadFailed(t *testing.T) {
	t.Parallel()

	writeErr := model.NewInternalError("utils.file.s3.writing.origin", "no such host")

	tests := []struct {
		name    string
		upload  *fakeUpload
		wantErr error
	}{
		{name: "write failed", upload: finished(writeErr, nil), wantErr: writeErr},
		{name: "finished without error", upload: finished(nil, &model.JobUploadFile{Id: 1})},
		// Still running: the client may resume, so nothing to report and no waiting.
		{name: "in progress", upload: &fakeUpload{done: make(chan struct{})}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if err := uploadFailed(tc.upload); !errors.Is(err, tc.wantErr) {
				t.Fatalf("err = %v, want %v", err, tc.wantErr)
			}
		})
	}
}
