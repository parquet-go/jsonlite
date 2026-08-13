//go:build goexperiment.simd && amd64

package jsonlite

import "testing"

// BenchmarkStage1Kernels measures each structural indexer implementation
// directly, independent of CPU dispatch.
func BenchmarkStage1Kernels(b *testing.B) {
	kernels := []struct {
		name string
		fn   func(string, []uint32) ([]uint32, stage1Flags, error)
	}{
		{"avx2", structuralIndexAVX2},
		{"portable", structuralIndexPortable},
	}
	index := make([]uint32, 0, 1024)
	for _, k := range kernels {
		b.Run(k.name, func(b *testing.B) {
			b.SetBytes(int64(len(internalCloudLoggingPayload)))
			for b.Loop() {
				var err error
				index, _, err = k.fn(internalCloudLoggingPayload, index[:0])
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
