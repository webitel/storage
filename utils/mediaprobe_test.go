package utils

import (
	"bytes"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func requireFFmpeg(t *testing.T) {
	t.Helper()

	for _, bin := range []string{"ffmpeg", "ffprobe"} {
		if _, err := exec.LookPath(bin); err != nil {
			t.Skipf("%s not installed", bin)
		}
	}
}

func makeVideo(t *testing.T, seconds string, faststart bool) []byte {
	t.Helper()

	out := filepath.Join(t.TempDir(), "v.mp4")
	args := []string{"-v", "error", "-y", "-f", "lavfi", "-i", "testsrc=duration=" + seconds + ":size=160x120:rate=10",
		"-c:v", "libx264", "-pix_fmt", "yuv420p"}
	if faststart {
		args = append(args, "-movflags", "+faststart")
	}

	if err := exec.Command("ffmpeg", append(args, out)...).Run(); err != nil {
		t.Skipf("ffmpeg cannot encode test video: %v", err)
	}

	b, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}

	return b
}

func TestMediaProbeDuration(t *testing.T) {
	requireFFmpeg(t)

	for name, faststart := range map[string]bool{"moov at end": false, "faststart": true} {
		t.Run(name, func(t *testing.T) {
			video := makeVideo(t, "3", faststart)

			probe, err := NewMediaProbe()
			if err != nil {
				t.Fatal(err)
			}

			var sink bytes.Buffer
			if _, err = io.Copy(&sink, io.TeeReader(bytes.NewReader(video), probe)); err != nil {
				t.Fatal(err)
			}

			if !bytes.Equal(sink.Bytes(), video) {
				t.Fatal("tee altered the uploaded bytes")
			}

			d, err := probe.Duration()
			if err != nil {
				t.Fatal(err)
			}

			if d < 2900*time.Millisecond || d > 3100*time.Millisecond {
				t.Fatalf("duration = %s, want ~3s", d)
			}
		})
	}
}

func TestMediaProbeGarbageDoesNotBreakUpload(t *testing.T) {
	requireFFmpeg(t)

	probe, err := NewMediaProbe()
	if err != nil {
		t.Fatal(err)
	}

	garbage := bytes.Repeat([]byte("not a video"), 64<<10)

	n, err := io.Copy(io.Discard, io.TeeReader(bytes.NewReader(garbage), probe))
	if err != nil || n != int64(len(garbage)) {
		t.Fatalf("copy = %d, %v; want %d, nil", n, err, len(garbage))
	}

	if _, err = probe.Duration(); err == nil {
		t.Fatal("expected an error for non-media input")
	}
}

func TestIsSupportMediaProbe(t *testing.T) {
	for mime, want := range map[string]bool{
		"video/mp4":       true,
		"audio/mpeg":      true,
		"image/png":       false,
		"application/pdf": false,
		"":                false,
	} {
		if got := IsSupportMediaProbe(mime); got != want {
			t.Errorf("IsSupportMediaProbe(%q) = %v, want %v", mime, got, want)
		}
	}
}

func TestMediaProbeStalledNeverBlocksUpload(t *testing.T) {
	sleep, err := exec.LookPath("sleep")
	if err != nil {
		t.Skip("sleep not available")
	}

	// sleep never reads stdin, like an ffprobe that hangs.
	probe, err := startMediaProbe(exec.Command(sleep, "30"), 200*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}

	upload := bytes.Repeat([]byte{1}, 32<<20)

	done := make(chan struct{})
	go func() {
		defer close(done)
		_, _ = io.Copy(io.Discard, io.TeeReader(bytes.NewReader(upload), probe))
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("upload blocked on a stalled probe")
	}

	if _, err = probe.Duration(); !errors.Is(err, errMediaProbeTimeout) {
		t.Fatalf("Duration err = %v, want timeout", err)
	}
}
