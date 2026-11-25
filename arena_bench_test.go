package jsonlite

import (
	"strings"
	"testing"
)

// parseSink keeps a parsed value reachable so the compiler cannot
// stack-allocate the tree and drop the arena's bytes out of the measurement.
var parseSink *Value

// BenchmarkParsePerRow parses one document per iteration, which is how
// parquet-go's BenchmarkWriteJSONShredded uses Parse: once per row, on a
// record whose only nesting is a 7-field http object inside a 4-field parent.
// small, medium and large differ in string length, not in the number of
// fields, so they want the same amount of field storage.
func BenchmarkParsePerRow(b *testing.B) {
	httpJSON := `"http":{"method":"GET","url":"https://example.com/api/v1/items",` +
		`"status":200,"latency_ms":12.5,"user_agent":"Mozilla/5.0 (compatible)",` +
		`"remote_ip":"192.168.1.100","cache_hit":false}`
	docs := []struct {
		name string
		json string
	}{
		{
			"small",
			`{"severity":"INFO","message":"request served","trace":"","http":{"method":"GET","url":"/","status":200,"latency_ms":1,"user_agent":"","remote_ip":"","cache_hit":true}}`,
		},
		{
			"medium",
			`{"severity":"INFO","message":"` + strings.Repeat("request served ", 8) +
				`","trace":"projects/test-project/traces/1234567890abcdef",` + httpJSON + `}`,
		},
		{
			"large",
			`{"severity":"WARNING","message":"` + strings.Repeat("upstream latency above threshold ", 24) +
				`","trace":"projects/test-project/traces/1234567890abcdef",` + httpJSON + `}`,
		},
	}
	for _, d := range docs {
		b.Run(d.name, func(b *testing.B) {
			b.ReportAllocs()
			b.SetBytes(int64(len(d.json)))
			for b.Loop() {
				v, err := Parse(d.json)
				if err != nil {
					b.Fatal(err)
				}
				parseSink = v
			}
		})
	}
}
