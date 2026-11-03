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

// Local storage is the service's own: a local profile from the database
// serves and removes the files it has but stores no new ones.
func TestProfileBackendStoreLocalReadOnly(t *testing.T) {
	root := t.TempDir()
	profile := &model.FileBackendProfile{
		DomainRecord: model.DomainRecord{Id: 7},
		Name:         "tenant local",
		Type:         model.FileDriverLocal,
		Properties:   model.StringInterface{"directory": root, "path_pattern": "$DOMAIN"},
	}

	// A file stored before the rule, written as the old code did.
	old, err := NewBackendStore(profile, nil)
	if err != nil {
		t.Fatal(err)
	}
	f := &model.File{DomainId: 1, BaseFile: model.BaseFile{Name: "old.wav"}}
	if _, err := old.Write(strings.NewReader("old"), f); err != nil {
		t.Fatal(err)
	}

	be, err := NewProfileBackendStore(profile, nil)
	if err != nil {
		t.Fatal(err)
	}
	if be.Name() != "tenant local" || be.Id() == nil || *be.Id() != 7 {
		t.Errorf("name/id = %q/%v, want the profile's", be.Name(), be.Id())
	}

	nf := &model.File{DomainId: 1, BaseFile: model.BaseFile{Name: "new.wav"}}
	if _, err := be.Write(strings.NewReader("new"), nf); err == nil || err.GetId() != ErrReadOnlyId {
		t.Errorf("Write: err = %v, want %s", err, ErrReadOnlyId)
	}
	if _, err := os.Stat(filepath.Join(root, "1", "new.wav")); err == nil {
		t.Error("Write stored a file")
	}
	if err := be.CopyTo(f, func(s string) string { return s + "/copy" }); err == nil || err.GetId() != ErrReadOnlyId {
		t.Errorf("CopyTo: err = %v, want %s", err, ErrReadOnlyId)
	}

	r, err := be.Reader(f, 0)
	if err != nil {
		t.Fatalf("Reader: %v", err)
	}
	b, _ := io.ReadAll(r)
	r.Close()
	if string(b) != "old" {
		t.Errorf("Reader = %q, want old", b)
	}
	if err := be.Remove(f); err != nil {
		t.Errorf("Remove: %v", err)
	}
}

func TestProfileBackendStoreNotLocal(t *testing.T) {
	be, err := NewProfileBackendStore(&model.FileBackendProfile{
		Type:       model.FileDriverS3,
		Properties: model.StringInterface{"bucket_name": "b", "region": "us-east-1", "endpoint": "http://127.0.0.1:1", "key_id": "k", "access_key": "s"},
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := be.(readOnlyBackend); ok {
		t.Error("an s3 profile is read-only")
	}
}
