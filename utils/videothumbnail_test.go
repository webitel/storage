package utils

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestVideoThumbnailFromFile(t *testing.T) {
	requireFFmpeg(t)

	for name, faststart := range map[string]bool{"moov at end": false, "faststart": true} {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "v.mp4")
			if err := os.WriteFile(path, makeVideo(t, "2", faststart), 0o600); err != nil {
				t.Fatal(err)
			}

			png, scale, err := VideoThumbnailFromFile(path, "")
			if err != nil {
				t.Fatal(err)
			}

			if !bytes.HasPrefix(png, []byte("\x89PNG")) {
				t.Fatalf("not a PNG (%d bytes)", len(png))
			}

			if scale != "scale="+ThumbnailScale {
				t.Errorf("scale = %q", scale)
			}
		})
	}
}

func TestVideoThumbnailFromFileRejectsNonVideo(t *testing.T) {
	requireFFmpeg(t)

	path := filepath.Join(t.TempDir(), "x.mp4")
	if err := os.WriteFile(path, []byte("<html>not a video</html>"), 0o600); err != nil {
		t.Fatal(err)
	}

	if _, _, err := VideoThumbnailFromFile(path, ""); err == nil {
		t.Fatal("expected an error for non-video input")
	}
}
