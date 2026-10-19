package utils

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

const (
	mediaProbeQueue   = 64
	mediaProbeTimeout = 30 * time.Second
	mediaProbeStderr  = 1 << 10
)

var errMediaProbeTimeout = errors.New("ffprobe: timed out")

// MediaProbe tees an upload into ffprobe to read the media duration. Write
// never blocks or fails: chunks go through a bounded queue and are dropped
// once ffprobe falls behind, so a stalled or crashed probe cannot hold up the
// upload it observes.
type MediaProbe struct {
	cmd     *exec.Cmd
	stdout  bytes.Buffer
	stderr  limitedBuffer
	queue   chan []byte
	fed     chan struct{}
	timeout time.Duration
	broken  bool
}

func NewMediaProbe() (*MediaProbe, error) {
	return startMediaProbe(exec.Command("ffprobe",
		"-v", "error",
		"-show_entries", "format=duration",
		"-of", "default=noprint_wrappers=1:nokey=1",
		"-i", "pipe:0",
	), mediaProbeTimeout)
}

func startMediaProbe(cmd *exec.Cmd, timeout time.Duration) (*MediaProbe, error) {
	p := &MediaProbe{
		cmd:     cmd,
		queue:   make(chan []byte, mediaProbeQueue),
		fed:     make(chan struct{}),
		timeout: timeout,
	}
	p.cmd.Stdout = &p.stdout
	p.stderr.limit = mediaProbeStderr
	p.cmd.Stderr = &p.stderr

	stdin, err := p.cmd.StdinPipe()
	if err != nil {
		return nil, err
	}

	if err = p.cmd.Start(); err != nil {
		return nil, err
	}

	go p.feed(stdin)

	return p, nil
}

func (p *MediaProbe) feed(stdin io.WriteCloser) {
	defer close(p.fed)
	defer stdin.Close()

	for chunk := range p.queue {
		if _, err := stdin.Write(chunk); err != nil {
			// ffprobe exited (often early, once it has the header); drain the rest.
			for range p.queue {
			}

			return
		}
	}
}

func (p *MediaProbe) Write(b []byte) (int, error) {
	if p.broken {
		return len(b), nil
	}

	select {
	case p.queue <- bytes.Clone(b):
	default:
		p.broken = true
	}

	return len(b), nil
}

// Duration ends the input and waits for ffprobe; call it exactly once.
func (p *MediaProbe) Duration() (time.Duration, error) {
	close(p.queue)

	done := make(chan error, 1)
	go func() {
		<-p.fed
		done <- p.cmd.Wait()
	}()

	select {
	case err := <-done:
		if err != nil {
			return 0, fmt.Errorf("ffprobe: %w: %s", err, strings.TrimSpace(p.stderr.String()))
		}
	case <-time.After(p.timeout):
		_ = p.cmd.Process.Kill()
		<-done

		return 0, errMediaProbeTimeout
	}

	return parseDuration(p.stdout.String())
}

// ProbeDurationFromFile reads a seekable file, so unlike the piped MediaProbe it
// never misses an index (moov) at the end of the file.
func ProbeDurationFromFile(path string) (time.Duration, error) {
	ctx, cancel := context.WithTimeout(context.Background(), mediaProbeTimeout)
	defer cancel()

	var stdout bytes.Buffer
	stderr := limitedBuffer{limit: mediaProbeStderr}

	cmd := exec.CommandContext(ctx, "ffprobe",
		"-v", "error",
		"-show_entries", "format=duration",
		"-of", "default=noprint_wrappers=1:nokey=1",
		path,
	)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if ctx.Err() != nil {
			return 0, errMediaProbeTimeout
		}

		return 0, fmt.Errorf("ffprobe: %w: %s", err, strings.TrimSpace(stderr.String()))
	}

	return parseDuration(stdout.String())
}

func parseDuration(raw string) (time.Duration, error) {
	out := strings.TrimSpace(raw)

	secs, err := strconv.ParseFloat(out, 64)
	if err != nil || secs <= 0 {
		return 0, fmt.Errorf("ffprobe: unreadable duration %q", out)
	}

	return time.Duration(secs * float64(time.Second)), nil
}

func IsSupportMediaProbe(mimeType string) bool {
	return strings.HasPrefix(mimeType, "video/") || strings.HasPrefix(mimeType, "audio/")
}

type limitedBuffer struct {
	bytes.Buffer
	limit int
}

func (b *limitedBuffer) Write(p []byte) (int, error) {
	if room := b.limit - b.Len(); room > 0 {
		b.Buffer.Write(p[:min(len(p), room)])
	}

	return len(p), nil
}
