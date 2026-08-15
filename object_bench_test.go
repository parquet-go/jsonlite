package jsonlite_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/parquet-go/jsonlite"
)

// wideObjects builds a payload of `count` objects each holding `fields`
// members, so every object clears smallObjectFields and gets a hash index.
// This isolates the parse-time hashing cost far better than the mixed
// CloudLogging payload, where only two of the objects are hashed at all.
func wideObjects(count, fields int, keyLen int) string {
	var b strings.Builder
	b.WriteByte('[')
	for i := range count {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteByte('{')
		for j := range fields {
			if j > 0 {
				b.WriteByte(',')
			}
			key := fmt.Sprintf("k%0*d", keyLen-1, j)
			fmt.Fprintf(&b, `"%s":%d`, key, j)
		}
		b.WriteByte('}')
	}
	b.WriteByte(']')
	return b.String()
}

func BenchmarkParseWideObjects(b *testing.B) {
	for _, tc := range []struct{ fields, keyLen int }{
		{9, 8}, {16, 8}, {16, 24}, {64, 8}, {64, 24},
	} {
		input := wideObjects(64, tc.fields, tc.keyLen)
		b.Run(fmt.Sprintf("fields=%d/keylen=%d", tc.fields, tc.keyLen), func(b *testing.B) {
			b.ReportAllocs()
			b.SetBytes(int64(len(input)))
			for b.Loop() {
				if _, err := jsonlite.Parse(input); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
