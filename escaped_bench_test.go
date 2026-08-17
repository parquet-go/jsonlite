package jsonlite_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/parquet-go/jsonlite"
)

// BenchmarkEscapedScan compares the word-at-a-time and vector scans across
// the string lengths a JSON document actually contains: keys and short values
// are tens of bytes, text fields hundreds.
func BenchmarkEscapedScan(b *testing.B) {
	b.Logf("vector scan built in: %v", jsonlite.EscapedHasWide())
	for _, n := range []int{16, 32, 64, 128, 256, 1024, 4096} {
		s := strings.Repeat("abcdefghijklmnopqrstuvwxyz0123456789", n/36+2)[:n]
		b.Run(fmt.Sprintf("len=%04d/dispatch", n), func(b *testing.B) {
			b.SetBytes(int64(n))
			for b.Loop() {
				if jsonlite.Escaped(s) {
					b.Fatal("unexpected")
				}
			}
		})
		if n >= 64 {
			b.Run(fmt.Sprintf("len=%04d/wide", n), func(b *testing.B) {
				b.SetBytes(int64(n))
				for b.Loop() {
					if jsonlite.EscapedWide(s) {
						b.Fatal("unexpected")
					}
				}
			})
		}
	}
}
