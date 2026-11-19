package mimex_test

import (
	"bytes"
	"fmt"
	"io"
	"net/http"

	"github.com/webitel/storage/utils/mimex"
)

func ExampleDetect() {
	jpegBytes := []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 0x4A, 0x46, 0x49, 0x46}

	mimeType, ext := mimex.Detect(jpegBytes)

	fmt.Println("MIME:", mimeType)
	fmt.Println("Extension:", ext)

	// Output:
	// MIME: image/jpeg
	// Extension: .jpg
}

func ExampleMatches() {
	isWave := mimex.Matches("audio/wav", "audio/wave")

	isPng := mimex.Matches("image/png", "audio/wav")

	fmt.Println(isWave)
	fmt.Println(isPng)

	// Output:
	// true
	// false
}

func ExampleIsGeneric() {
	fmt.Println(mimex.IsGeneric("application/octet-stream"))
	fmt.Println(mimex.IsGeneric("binary/octet-stream"))
	fmt.Println(mimex.IsGeneric("image/jpeg"))

	// Output:
	// true
	// true
	// false
}

func ExampleIsAudio() {
	wavBytes := []byte("RIFF____WAVEfmt ")
	jpegBytes := []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 0x4A, 0x46, 0x49, 0x46}

	fmt.Println(mimex.IsAudio(wavBytes))
	fmt.Println(mimex.IsAudio(jpegBytes))

	// Output:
	// true
	// false
}

func ExampleResolveURLUploadMIME() {
	body := io.NopCloser(bytes.NewReader([]byte("RIFF____WAVEfmt audio stream data")))
	res := &http.Response{
		Header: http.Header{
			"Content-Type": []string{"application/octet-stream"},
		},
		Body: body,
	}

	bodyRead, resolvedMime, err := mimex.ResolveURLUploadMIME(res, "audio/wav")
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	defer bodyRead.Close()

	fmt.Println("Resolved MIME:", resolvedMime)

	// Output:
	// Resolved MIME: audio/wav
}
