package utils

import (
	"fmt"
	"github.com/pkg/errors"
	"io"
	"os/exec"
	"strings"
)

type Thumbnail struct {
	io.Writer
	io.Closer
	l      int
	stdin  io.WriteCloser
	stdout io.ReadCloser
	cmd    *exec.Cmd
	end    bool
}

func (r *Thumbnail) Write(p []byte) (nn int, err error) {
	if r.end {
		return len(p), nil // TODO wait if io.EOF
	}
	nn, err = r.stdin.Write(p)
	r.l += nn
	if err != nil {
		r.end = true
		return nn, nil
	}

	return
}

func (r *Thumbnail) Reader() io.Reader {
	return r.stdout
}

func (r *Thumbnail) Close() (err error) {
	fmt.Println("send ", r.l)

	err = r.stdin.Close() // close the stdin, or ffmpeg will wait forever
	if err != nil {
		return err
	}

	err = r.cmd.Wait() // wait until ffmpeg finish
	if err != nil {
		return err
	}

	return nil
}

func mimeCmdArgs(mime string) []string {
	if strings.HasPrefix(mime, "image/") {
		return []string{
			"-i", "pipe:0",
			"-f", "image2pipe",
			"-vcodec", "png",
			"-vf",
			"scale=128:-1",
			"pipe:1",
		}
	} else if strings.HasPrefix(mime, "video/") {
		return []string{
			"-err_detect", "ignore_err",
			//"-f", "mp4", // Вказуємо формат вхідного файлу
			"-i", "pipe:0", // Використання pipe:0 для отримання даних з io.Reader
			"-ss", "00:00:01", // Затримка 2 секунди
			"-vframes", "1", // Захопити лише 1 кадр
			"-f", "image2pipe", // Вивід у форматі image2pipe
			"-vcodec", "png", // Виведення у форматі PNG
			"-pix_fmt", "rgba", // Формат пікселів
			"pipe:1", // pipe:1 для виводу у io.Writer
		}
	}

	return nil
}

func NewThumbnail(mime string) (*Thumbnail, error) {

	cmdArgs := mimeCmdArgs(mime)
	if cmdArgs == nil {
		return nil, errors.New("not supported")
	}

	cmd := exec.Command("ffmpeg", cmdArgs...)
	//cmd.Stderr = os.Stderr // bind log stream to stderr

	stdin, _ := cmd.StdinPipe() // Open stdin pipe
	//fh, _ := os.OpenFile("/Users/ihor/work/storage/bin/"+model.NewId()[:6]+".png",
	//	os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	stdout, _ := cmd.StdoutPipe() // Open stout pipe
	//cmd.Stdout = fh
	cmd.Start()

	return &Thumbnail{
		stdin:  stdin,
		stdout: stdout,
		cmd:    cmd,
	}, nil
}
