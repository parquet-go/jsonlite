package jsonlite

import (
	"strings"
	"testing"
)

// TestParseIsByteTransparent verifies that Parse treats input as opaque
// bytes: string values that are not valid UTF-8 are accepted and preserved
// as-is, on both parse paths.
func TestParseIsByteTransparent(t *testing.T) {
	pad := `"` + strings.Repeat("x", 600) + `",` // forces the indexed path
	for _, doc := range []string{
		"{\"k\":\"\xff\"}",
		`{"pad":[` + pad + "\"a\x80b\"]}",
	} {
		v, err := Parse(doc)
		if err != nil {
			t.Errorf("Parse(%q) = %v, want success (byte-transparent)", doc, err)
			continue
		}
		if got := string(v.Compact(nil)); !strings.Contains(got, "\xff") && !strings.Contains(got, "\x80") {
			t.Errorf("Parse(%q) did not preserve raw bytes: %q", doc, got)
		}
	}
}
