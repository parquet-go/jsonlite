//go:build goexperiment.simd && amd64 && go1.27

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

// TestStructuralIndexAVX2MatchesPortable compares the AVX2 kernel itself,
// not the AVX2 dispatch in structuralIndex, with the scalar indexer.
func TestStructuralIndexAVX2MatchesPortable(t *testing.T) {
	if !simdStage1() {
		t.Skip("AVX2 not available")
	}
	for _, input := range structuralMatchInputs(t) {
		assertStructuralMatch(t, input, structuralIndexAVX2)
	}
}
