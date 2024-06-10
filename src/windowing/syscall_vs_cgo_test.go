package windowing

import (
	"testing"
)

func BenchmarkSyscall(b *testing.B) {
	for i := 0; i < b.N; i++ {
		syscallTest()
	}
}

func BenchmarkCgo(b *testing.B) {
	for i := 0; i < b.N; i++ {
		cgoTest()
	}
}
