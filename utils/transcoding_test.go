package utils

import (
	"os"
	"testing"
)

func TestTranscoding(t *testing.T) {
	testVP9(t)
}

func testVP9(t *testing.T) {
	f, err := os.Open("../test_data/stream.vp9")
	if err != nil {
		panic(err.Error())
	}
	defer f.Close()

	os.Remove("../test_data/output")

	out, err := os.Create("../test_data/output")
	if err != nil {
		panic(err.Error())
	}
	defer out.Close()

	tr, err := NewTranscoding(f, out)
	if err != nil {
		panic(err)
	}
	defer tr.Close()
	err = tr.Start()
	if err != nil {
		panic(err)
	}
}
