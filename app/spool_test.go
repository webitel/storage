package app

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSpoolFileNeverFailsTheUpload(t *testing.T) {
	f, err := os.Create(filepath.Join(t.TempDir(), "spool"))
	if err != nil {
		t.Fatal(err)
	}

	s := &spoolFile{f: f}

	if n, err := s.Write([]byte("abc")); n != 3 || err != nil || s.err != nil {
		t.Fatalf("write = %d, %v (spool err %v)", n, err, s.err)
	}

	_ = f.Close()

	if n, err := s.Write([]byte("def")); n != 3 || err != nil {
		t.Fatalf("write after close = %d, %v; want 3, nil", n, err)
	}

	if s.err == nil {
		t.Fatal("spool error not recorded")
	}
}
