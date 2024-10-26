package utils

import (
	"io"
	"os"
	"testing"
)

type writer struct {
}

func (r *writer) Write(buf []byte) (int, error) {
	return len(buf), nil
}

func TestThumbnail(t *testing.T) {
	imageTest(t)
}

func imageTest(t *testing.T) {
	//src, err := os.Open("/Users/ihor/work/storage/1/2.mp4")
	src, err := os.Open("/Users/ihor/work/storage/1/1.jpg")
	if err != nil {
		t.Fatal(err)
	}
	//dst, err := os.OpenFile(allPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)

	th, err := NewThumbnail("image/", "")
	if err != nil {
		t.Fatal(err)
	}
	png := io.TeeReader(src, th)

	_, err = io.Copy(&writer{}, png)
	if err != nil {
		t.Fatal(err)
	}
	err = th.Close()
	if err != nil {
		t.Fatal(err)
	}
}
