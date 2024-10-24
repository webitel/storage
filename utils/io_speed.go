package utils

import (
	"fmt"
	"github.com/juju/ratelimit"
	"io"
	"time"
)

type speedCalc struct {
	count      int64 // may have large (2GB+) files - so don't use int
	start, end time.Time
}

type speedReader struct {
	r io.Reader
	speedCalc
}

type speedWriter struct {
	w io.Writer
	speedCalc
}

func SpeedLimitReader(src io.Reader, speed int64) io.Reader {
	bucket := ratelimit.NewBucketWithRate(float64(speed)*1024, speed*1024)
	return &speedReader{
		r: ratelimit.Reader(src, bucket),
	}
}

func SpeedLimitWriter(src io.Writer, speed int64) io.Writer {
	bucket := ratelimit.NewBucketWithRate(float64(speed)*1024, speed*1024)
	return &speedWriter{
		w: ratelimit.Writer(src, bucket),
	}
}

func (r *speedReader) Read(b []byte) (n int, err error) {
	if r.start.IsZero() {
		r.start = time.Now()
	}

	n, err = r.r.Read(b) // underlying io.Reader read

	r.count += int64(n)

	if err == io.EOF {
		r.end = time.Now()
	}

	return
}

func (r *speedWriter) Write(b []byte) (n int, err error) {
	if r.start.IsZero() {
		r.start = time.Now()
	}

	n, err = r.w.Write(b) // underlying io.Writer write

	r.count += int64(n)

	if err == io.EOF {
		r.end = time.Now()
	}

	return
}

func (r *speedCalc) Rate() (n int64, d time.Duration) {
	end := r.end
	if end.IsZero() {
		end = time.Now()
	}
	return r.count, end.Sub(r.start)
}

func (r *speedCalc) String() string {
	n, d := r.Rate()
	return fmt.Sprintf("%.0f kB/s", float64(n/1000)/(d.Seconds()))
}
