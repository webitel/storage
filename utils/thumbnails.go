package utils

import (
	"errors"
	"io"
	"os/exec"
	"strings"
)

const (
	ThumbnailScale = "128:-1"
)

type Thumbnail struct {
	scale string
	io.Writer
	io.Closer
	l        int64
	stdin    io.WriteCloser
	stdout   io.ReadCloser
	cmd      *exec.Cmd
	end      bool
	UserData interface{}
}

func NewThumbnail(mime string, scale string) (*Thumbnail, error) {
	if scale == "" {
		scale = ThumbnailScale
	}
	scale = "scale=" + scale
	cmdArgs := mimeCmdArgs(mime, scale)
	if cmdArgs == nil {
		return nil, errors.New("not supported")
	}

	cmd := exec.Command("ffmpeg", cmdArgs...)
	//cmd.Stderr = os.Stderr // bind log stream to stderr

	stdin, _ := cmd.StdinPipe()   // Open stdin pipe
	stdout, _ := cmd.StdoutPipe() // Open stout pipe
	cmd.Start()

	return &Thumbnail{
		scale:  scale,
		stdin:  stdin,
		stdout: stdout,
		cmd:    cmd,
	}, nil
}

func (t *Thumbnail) Write(p []byte) (nn int, err error) {
	if t.end {
		return len(p), nil // TODO wait if io.EOF
	}
	nn, err = t.stdin.Write(p)
	t.l += int64(nn)
	if err != nil {
		t.end = true
		return nn, nil
	}

	return
}

func (t *Thumbnail) Reader() io.Reader {
	return t.stdout
}

func (t *Thumbnail) Close() (err error) {
	err = t.stdin.Close() // close the stdin, or ffmpeg will wait forever
	if err != nil {
		return err
	}

	err = t.cmd.Wait() // wait until ffmpeg finish
	if err != nil {
		return err
	}

	return nil
}

func (t *Thumbnail) StopWriter() {
	t.stdin.Close()
}

func (t *Thumbnail) Size() int64 {
	return t.l
}

func (t *Thumbnail) Scale() string {
	return t.scale
}

func mimeCmdArgs(mime string, scale string) []string {
	if strings.HasPrefix(mime, "image/") {
		return []string{
			"-i", "pipe:0",
			"-f", "image2pipe",
			"-vcodec", "png",
			"-pix_fmt", "rgba", // Формат пікселів
			"-threads", "1",
			"-vf", scale,
			"pipe:1",
		}
	} else if strings.HasPrefix(mime, "video/") {
		return []string{
			"-err_detect", "ignore_err",
			//"-f", "mp4", // Вказуємо формат вхідного файлу
			"-i", "pipe:0", // Використання pipe:0 для отримання даних з io.Reader
			//"-ss", "00:00:01", // Затримка 2 секунди
			"-vframes", "1", // Захопити лише 1 кадр
			"-f", "image2pipe", // Вивід у форматі image2pipe
			"-vcodec", "png", // Виведення у форматі PNG
			"-pix_fmt", "rgba", // Формат пікселів
			//"-threads", "1",
			"-vf", scale,
			"pipe:1", // pipe:1 для виводу у io.Writer
		}
	}

	return nil
}

func IsSupportThumbnail(mimeType string) bool {
	return strings.HasPrefix(mimeType, "video/") || strings.HasPrefix(mimeType, "image/")
}
