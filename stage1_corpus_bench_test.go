package jsonlite

import (
	"fmt"
	"testing"
)

// BenchmarkScanCorpus compares the two ways jsonlite finds structure in a
// document, over the same inputs:
//
//   - tokenizer: nextToken, the scalar scanner that Tokenizer and Iterator
//     drive one token at a time.
//   - stage1: structuralIndex, the block-at-a-time indexer, vectorized on
//     amd64 under GOEXPERIMENT=simd and scalar everywhere else.
//
// Iterator is built on the tokenizer and never touches stage 1, so the gap
// between these two is the headroom available to a stage-1-backed Iterator.
// Reading it on a machine where simdStage1() is false measures the scalar
// indexer and understates the gap.
func BenchmarkScanCorpus(b *testing.B) {
	b.Logf("simdStage1()=%v", simdStage1())
	for _, d := range benchCorpus() {
		b.Run(d.Name+"/tokenizer", func(b *testing.B) {
			b.ReportAllocs()
			b.SetBytes(int64(len(d.JSON)))
			for b.Loop() {
				s := d.JSON
				for {
					_, rest, ok := nextToken(s)
					if !ok {
						break
					}
					s = rest
				}
			}
		})
		b.Run(d.Name+"/stage1", func(b *testing.B) {
			b.ReportAllocs()
			b.SetBytes(int64(len(d.JSON)))
			index := make([]uint32, 0, 1024)
			for b.Loop() {
				var err error
				index, _, err = structuralIndex(d.JSON, index[:0])
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// BenchmarkParsePathsCorpus compares the two parse implementations directly,
// bypassing the length threshold in ParseMaxDepth so both run on every
// document. The crossover this reveals is what indexedParseThreshold encodes.
func BenchmarkParsePathsCorpus(b *testing.B) {
	for _, d := range benchCorpus() {
		b.Run(d.Name+"/scalar", func(b *testing.B) {
			b.ReportAllocs()
			b.SetBytes(int64(len(d.JSON)))
			var p *parser // acquired from the pool on first container
			for b.Loop() {
				if _, _, err := p.parseValue(d.JSON, DefaultMaxDepth); err != nil {
					b.Fatal(err)
				}
			}
		})
		b.Run(d.Name+"/indexed", func(b *testing.B) {
			b.ReportAllocs()
			b.SetBytes(int64(len(d.JSON)))
			for b.Loop() {
				if _, err := parseIndexed(d.JSON, DefaultMaxDepth); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// BenchmarkThresholdSweep locates the document size at which the indexed
// parser overtakes the scalar one, which is what indexedParseThreshold
// encodes. The threshold only matters where simdStage1() is true; measured
// with it false, the indexed path is handicapped and the crossover reads far
// higher than it is in a build that can use the vector indexer.
func BenchmarkThresholdSweep(b *testing.B) {
	for _, size := range []int{64, 128, 192, 256, 384, 512, 768, 1024, 2048} {
		input := benchSizedRecord(size)
		b.Run(fmt.Sprintf("size=%04d/scalar", size), func(b *testing.B) {
			b.SetBytes(int64(len(input)))
			var p *parser // acquired from the pool on first container
			for b.Loop() {
				if _, _, err := p.parseValue(input, DefaultMaxDepth); err != nil {
					b.Fatal(err)
				}
			}
		})
		b.Run(fmt.Sprintf("size=%04d/indexed", size), func(b *testing.B) {
			b.SetBytes(int64(len(input)))
			for b.Loop() {
				if _, err := parseIndexed(input, DefaultMaxDepth); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// BenchmarkParseSized goes through Parse rather than calling a path directly,
// so it measures the dispatch indexedParseThreshold controls end to end.
func BenchmarkParseSized(b *testing.B) {
	for _, size := range []int{192, 256, 384, 512} {
		input := benchSizedRecord(size)
		b.Run(fmt.Sprintf("size=%04d", size), func(b *testing.B) {
			b.SetBytes(int64(len(input)))
			for b.Loop() {
				if _, err := Parse(input); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
