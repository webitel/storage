package utils

import (
	"testing"
)

func BenchmarkThumbnail(b *testing.B) {
	for i := 0; i < b.N; i++ {
		imageTest(&testing.T{})
	}
}
