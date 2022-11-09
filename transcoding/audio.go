package main

// #cgo LDFLAGS: -lavcodec -lavformat -lavutil -lavdevice -lswresample
//#include "audio.c"
import "C"
import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"reflect"
	"runtime"
	"runtime/cgo"
	"unsafe"
)

/*
	TODO
	1. Read from memory https://stackoverflow.com/questions/19785254/play-a-video-from-memorystream-using-ffmpeg
*/

type transcoding struct {
	rate int
	w    io.WriteCloser
}

func New(w io.WriteCloser, rate int) *transcoding {
	return &transcoding{
		rate: rate,
		w:    w,
	}
}

func (t *transcoding) Write(data []byte) (int, error) {
	return t.w.Write(data)
}

//export goFrameHandler
func goFrameHandler(ctx C.uintptr_t, samples *C.uint8_t, clen C.int, cchannels C.int) {
	if clen == 0 || cchannels == 0 {
		return
	}

	h := cgo.Handle(ctx)
	v := h.Value()
	tr, ok := v.(*transcoding)
	if !ok {
		panic(1)
	}

	//res := make([][]float32, cchannels, cchannels)
	bufLen := (int)(clen)
	//length := (int)(clen / cchannels)
	//channels := (int)(cchannels)

	var list []C.uint8_t
	sliceHeader := (*reflect.SliceHeader)(unsafe.Pointer(&list))
	sliceHeader.Cap = bufLen
	sliceHeader.Len = bufLen
	sliceHeader.Data = uintptr(unsafe.Pointer(samples))

	//for i := 0; i < (int)(channels); i++ {
	//	res[i] = make([]float32, length, length)
	//}
	//
	//for k, sample := range list {
	//	res[k%channels][(k-(k%channels))/channels] = (float32)(sample)
	//}

	buf := new(bytes.Buffer)
	binary.Write(buf, binary.LittleEndian, list)
	tr.Write(buf.Bytes())
	fmt.Println("W ", len(buf.Bytes()))
}

func (e *transcoding) Decode(ctx context.Context, path string) {
	h := cgo.NewHandle(e)
	C.transcode(C.uintptr_t(h), C.CString(path), C.int(e.rate))
	h.Delete()
	e.w.Close()
}

func init() {
	C.initTranscoding()
}

func main() {

	gfile, err := os.OpenFile("./out.pcm", os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		panic(err.Error())
	}

	t := New(gfile, 16000)

	t.Decode(context.TODO(), "https://dev.webitel.com/api/storage/recordings/69904/stream?access_token=ira4d61g4jb1jm7647eysbhohr")

	gfile.Close()
	runtime.GC()
}
