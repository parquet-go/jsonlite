package jsonlite_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/parquet-go/jsonlite"
)

// smallObject builds a single flat object with n fields and realistic
// snake_case key names, sized to sit around the smallObjectFields threshold.
func smallObject(n int) (string, []string) {
	names := []string{
		"id",
		"name",
		"email",
		"created_at",
		"updated_at",
		"status",
		"account_id",
		"region",
		"tier",
		"owner",
	}
	keys := make([]string, n)
	parts := make([]string, n)
	for i := range n {
		keys[i] = names[i%len(names)]
		if i >= len(names) {
			keys[i] = fmt.Sprintf("%s_%d", names[i%len(names)], i)
		}
		parts[i] = fmt.Sprintf(`"%s":%d`, keys[i], i)
	}
	return "{" + strings.Join(parts, ",") + "}", keys
}

// BenchmarkThreshParse isolates the parse-side cost of the threshold: with the
// index it pays one allocation plus n hashes, without it pays nothing.
func BenchmarkThreshParse(b *testing.B) {
	for _, n := range []int{1, 2, 3, 4, 6, 8, 12} {
		input, _ := smallObject(n)
		b.Run(fmt.Sprintf("fields=%d", n), func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				if _, err := jsonlite.Parse(input); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// BenchmarkThreshLookup isolates the lookup-side benefit: hash+memchr versus a
// linear key scan. Cycles through every key so the result is the average over
// field positions rather than a best or worst case.
func BenchmarkThreshLookup(b *testing.B) {
	for _, n := range []int{1, 2, 3, 4, 6, 8, 12} {
		input, keys := smallObject(n)
		v, err := jsonlite.Parse(input)
		if err != nil {
			b.Fatal(err)
		}
		b.Run(fmt.Sprintf("fields=%d", n), func(b *testing.B) {
			b.ReportAllocs()
			i := 0
			for b.Loop() {
				if v.Lookup(keys[i]) == nil {
					b.Fatal("missing key")
				}
				if i++; i == len(keys) {
					i = 0
				}
			}
		})
	}
}

// BenchmarkThreshParseLookup is the decisive one: indexing is a cost paid once
// at parse and recovered over subsequent lookups, so the threshold is only
// meaningful relative to how many lookups a parsed object receives. Each
// iteration parses once and performs `lookups` lookups.
func BenchmarkThreshParseLookup(b *testing.B) {
	for _, n := range []int{2, 4, 6, 8, 12} {
		input, keys := smallObject(n)
		for _, lookups := range []int{1, 2, 4, 8, 16} {
			b.Run(fmt.Sprintf("fields=%d/lookups=%d", n, lookups), func(b *testing.B) {
				b.ReportAllocs()
				for b.Loop() {
					v, err := jsonlite.Parse(input)
					if err != nil {
						b.Fatal(err)
					}
					for j := range lookups {
						if v.Lookup(keys[j%len(keys)]) == nil {
							b.Fatal("missing key")
						}
					}
				}
			})
		}
	}
}
