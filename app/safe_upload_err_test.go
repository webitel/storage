package app

import (
	"sync"
	"testing"

	"github.com/webitel/storage/model"
)

// Err is read by the gRPC handler while run() may still be recording the
// upload result; -race fails this test if the read is unsynchronized.
func TestSafeUploadErr_ConcurrentWithResult(t *testing.T) {
	t.Parallel()

	s := &SafeUpload{}
	writeErr := model.NewInternalError("utils.file.s3.writing.origin", "no such host")

	var wg sync.WaitGroup

	wg.Add(2)

	go func() {
		defer wg.Done()

		s.mx.Lock()
		s.err = writeErr
		s.mx.Unlock()
	}()

	go func() {
		defer wg.Done()

		for range 100 {
			_ = s.Err()
		}
	}()

	wg.Wait()

	if err := s.Err(); err == nil || err.Error() != writeErr.Error() {
		t.Fatalf("Err() = %v, want %v", err, writeErr)
	}
}

func TestSafeUploadErr_NilWithoutResult(t *testing.T) {
	t.Parallel()

	if err := (&SafeUpload{}).Err(); err != nil {
		t.Fatalf("Err() = %v, want nil", err)
	}
}
