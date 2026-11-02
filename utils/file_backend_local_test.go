package utils

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/webitel/storage/model"
)

// A store name is built from what the client sends: the upload handler makes
// it "<uuid>_<id>_" + ?name=. A name with ../ must not put, read or remove a
// file outside the storage directory.
func TestLocalFileBackendPathEscape(t *testing.T) {
	base := t.TempDir()
	root := filepath.Join(base, "storage")
	be := &LocalFileBackend{directory: root}

	upload := func(name string) (*model.JobUploadFile, model.AppError) {
		f := &model.JobUploadFile{}
		f.Uuid = "9"
		f.Name = model.NewId() + "_" + name
		_, err := be.Write(strings.NewReader("poc"), f)
		return f, err
	}

	if _, err := upload("ok.txt"); err != nil {
		t.Fatalf("plain name: %v", err)
	}

	for _, name := range []string{"/../../escape.txt", "/../../../escape.txt", "/../../storage-sibling/x.txt"} {
		f, err := upload(name)
		if err == nil || err.GetId() != ErrPathEscapeId {
			t.Errorf("Write %q: err = %v, want %s", name, err, ErrPathEscapeId)
		}
		if _, err := be.Reader(f, 0); err == nil || err.GetId() != ErrPathEscapeId {
			t.Errorf("Reader %q: err = %v, want %s", name, err, ErrPathEscapeId)
		}
		if err := be.Remove(f); err == nil || err.GetId() != ErrPathEscapeId {
			t.Errorf("Remove %q: err = %v, want %s", name, err, ErrPathEscapeId)
		}
	}
	if err := be.RemoveFile("..", "escape.txt"); err == nil || err.GetId() != ErrPathEscapeId {
		t.Errorf("RemoveFile(..): err = %v, want %s", err, ErrPathEscapeId)
	}

	// Nothing landed outside the storage directory.
	filepath.Walk(base, func(p string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() {
			if rel, _ := filepath.Rel(root, p); strings.HasPrefix(rel, "..") {
				t.Errorf("file written outside the storage directory: %s", p)
			}
		}
		return nil
	})

	// A name with ".." inside it that stays in the directory is still fine.
	f, err := upload("a..b.txt")
	if err != nil {
		t.Fatalf("name with dots: %v", err)
	}
	r, err := be.Reader(f, 0)
	if err != nil {
		t.Fatalf("read back: %v", err)
	}
	defer r.Close()
	if b, _ := io.ReadAll(r); string(b) != "poc" {
		t.Errorf("read back %q, want poc", b)
	}
}
