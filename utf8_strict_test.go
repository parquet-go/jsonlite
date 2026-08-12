package jsonlite

import (
	"strings"
	"testing"
	stdutf8 "unicode/utf8"
)

// TestParseRejectsInvalidUTF8 verifies that Parse refuses malformed UTF-8 on
// both the classic path (small documents) and the indexed path (large
// documents, when the vectorized stage 1 is active).
func TestParseRejectsInvalidUTF8(t *testing.T) {
	pad := `"` + strings.Repeat("x", 600) + `",` // forces the indexed path
	cases := []string{
		"\"\xff\"",
		"\"\xc0\xaf\"",     // overlong
		"\"\xed\xa0\x80\"", // surrogate
		"\"a\x80b\"",       // stray continuation
		"\"" + strings.Repeat("y", 80) + "\xf0\x90\"", // truncated 4-byte
	}
	for _, bad := range cases {
		small := `{"k":` + bad + `}`
		if _, err := Parse(small); err == nil {
			t.Errorf("small doc %q: invalid UTF-8 accepted", small)
		}
		large := `{"pad":[` + pad + bad + `]}`
		if _, err := Parse(large); err == nil {
			t.Errorf("large doc with %q: invalid UTF-8 accepted", bad)
		}
	}
	// Valid unicode must still parse on both paths.
	good := `{"msg":"héllo wörld 🎉 日本語"}`
	if _, err := Parse(good); err != nil {
		t.Errorf("small valid unicode rejected: %v", err)
	}
	largeGood := `{"pad":[` + pad + `"日本語のログメッセージ 🎉"]}`
	if _, err := Parse(largeGood); err != nil {
		t.Errorf("large valid unicode rejected: %v", err)
	}
}

// TestParseSeqRejectsInvalidUTF8 verifies strictness for sequences.
func TestParseSeqRejectsInvalidUTF8(t *testing.T) {
	for v, err := range ParseSeq("{\"a\":1}\n{\"b\":\"\xff\"}") {
		if err == nil {
			t.Fatalf("expected error, got value %v", v.JSON())
		}
		break
	}
}

// FuzzParseStrictUTF8 asserts the strictness invariant: Parse never succeeds
// on input that is not valid UTF-8, and never rejects valid UTF-8 that the
// classic grammar accepts (cross-checked via parseIndexed).
func FuzzParseStrictUTF8(f *testing.F) {
	f.Add(`{"k":"héllo"}`)
	f.Add("\"\xff\"")
	f.Add(`{"pad":"` + strings.Repeat("x", 600) + `"}`)
	f.Fuzz(func(t *testing.T, input string) {
		v, err := Parse(input)
		if err == nil && !stdutf8.ValidString(input) {
			t.Fatalf("Parse accepted invalid UTF-8: %q", input)
		}
		vi, erri := parseIndexed(input, DefaultMaxDepth)
		if (err == nil) != (erri == nil) {
			t.Fatalf("input %q: Parse err=%v, parseIndexed err=%v", input, err, erri)
		}
		if err == nil && v.JSON() != vi.JSON() {
			t.Fatalf("input %q: JSON mismatch", input)
		}
	})
}
