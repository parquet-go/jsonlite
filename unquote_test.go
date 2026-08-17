package jsonlite_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/parquet-go/jsonlite"
)

func TestUnquoteValid(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty string is unquoted correctly",
			input:    `""`,
			expected: "",
		},
		{
			name:     "simple string is unquoted correctly",
			input:    `"hello"`,
			expected: "hello",
		},
		{
			name:     "string with spaces is unquoted correctly",
			input:    `"hello world"`,
			expected: "hello world",
		},
		{
			name:     "escaped quote is unquoted correctly",
			input:    `"say \"hello\""`,
			expected: `say "hello"`,
		},
		{
			name:     "escaped backslash is unquoted correctly",
			input:    `"path\\to\\file"`,
			expected: `path\to\file`,
		},
		{
			name:     "escaped slash is unquoted correctly",
			input:    `"a\/b"`,
			expected: "a/b",
		},
		{
			name:     "escaped backspace is unquoted correctly",
			input:    `"a\bb"`,
			expected: "a\bb",
		},
		{
			name:     "escaped formfeed is unquoted correctly",
			input:    `"a\fb"`,
			expected: "a\fb",
		},
		{
			name:     "escaped newline is unquoted correctly",
			input:    `"line1\nline2"`,
			expected: "line1\nline2",
		},
		{
			name:     "escaped carriage return is unquoted correctly",
			input:    `"line1\rline2"`,
			expected: "line1\rline2",
		},
		{
			name:     "escaped tab is unquoted correctly",
			input:    `"col1\tcol2"`,
			expected: "col1\tcol2",
		},
		{
			name:     "unicode null character is unquoted correctly",
			input:    `"\u0000"`,
			expected: "\u0000",
		},
		{
			name:     "unicode ascii character is unquoted correctly",
			input:    `"\u0041"`,
			expected: "A",
		},
		{
			name:     "unicode multibyte characters are unquoted correctly",
			input:    `"\u4e2d\u6587"`,
			expected: "中文",
		},
		{
			name:     "unicode max value is unquoted correctly",
			input:    `"\uffff"`,
			expected: "\uffff",
		},
		{
			name:     "mixed escapes are unquoted correctly",
			input:    `"line1\nline2\ttab\u0041end"`,
			expected: "line1\nline2\ttabAend",
		},
		{
			name:     "multiple quotes are unquoted correctly",
			input:    `"\"quote1\" and \"quote2\""`,
			expected: `"quote1" and "quote2"`,
		},
		{
			name:     "all single character escapes are unquoted correctly",
			input:    `"\"\\\//\b\f\n\r\t"`,
			expected: "\"\\//\b\f\n\r\t",
		},
		{
			name:     "string with only escaped characters is unquoted correctly",
			input:    `"\n\t"`,
			expected: "\n\t",
		},
		{
			name:     "long string is unquoted correctly",
			input:    `"The quick brown fox jumps over the lazy dog"`,
			expected: "The quick brown fox jumps over the lazy dog",
		},
		{
			name:     "string with numbers is unquoted correctly",
			input:    `"test123"`,
			expected: "test123",
		},
		{
			name:     "json value is unquoted correctly",
			input:    `"{\"key\":\"value\"}"`,
			expected: `{"key":"value"}`,
		},
		{
			name:     "unicode lowercase hex is unquoted correctly",
			input:    `"\u00e9"`,
			expected: "é",
		},
		{
			name:     "unicode emoji is unquoted correctly",
			input:    `"\ud83d\ude00"`,
			expected: "😀",
		},
		{
			name:     "unicode surrogate pair heart is unquoted correctly",
			input:    `"\ud83d\udc96"`,
			expected: "💖",
		},
		{
			name:     "unicode surrogate pair rocket is unquoted correctly",
			input:    `"\ud83d\ude80"`,
			expected: "🚀",
		},
		{
			name:     "multiple emojis are unquoted correctly",
			input:    `"\ud83d\ude00\ud83d\udc96"`,
			expected: "😀💖",
		},
		{
			name:     "emoji with text is unquoted correctly",
			input:    `"Hello \ud83d\udc4b World"`,
			expected: "Hello 👋 World",
		},
		{
			name:     "consecutive escapes are unquoted correctly",
			input:    `"\\\\n"`,
			expected: `\\n`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := jsonlite.Unquote(tt.input)
			if err != nil {
				t.Errorf("Unquote(%q) unexpected error: %v", tt.input, err)
			}
			if got != tt.expected {
				t.Errorf("Unquote(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestUnquoteInvalid(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "string without quotes returns error",
			input: "hello",
		},
		{
			name:  "string with only opening quote returns error",
			input: `"hello`,
		},
		{
			name:  "string with only closing quote returns error",
			input: `hello"`,
		},
		{
			name:  "single quote character returns error",
			input: `"`,
		},
		{
			name:  "empty input returns error",
			input: "",
		},
		{
			name:  "trailing backslash returns error",
			input: `"hello\`,
		},
		{
			name:  "trailing backslash before quote returns error",
			input: `"hello\"`,
		},
		{
			name:  "invalid escape sequence with x returns error",
			input: `"hello\x"`,
		},
		{
			name:  "invalid escape sequence with v returns error",
			input: `"hello\v"`,
		},
		{
			name:  "invalid escape sequence with a returns error",
			input: `"hello\a"`,
		},
		{
			name:  "invalid escape sequence with digit returns error",
			input: `"hello\0"`,
		},
		{
			name:  "incomplete unicode sequence with three chars returns error",
			input: `"\u041"`,
		},
		{
			name:  "incomplete unicode sequence with two chars returns error",
			input: `"\u04"`,
		},
		{
			name:  "incomplete unicode sequence with one char returns error",
			input: `"\u0"`,
		},
		{
			name:  "incomplete unicode sequence with no chars returns error",
			input: `"\u"`,
		},
		{
			name:  "invalid unicode hex character g returns error",
			input: `"\u00GG"`,
		},
		{
			name:  "invalid unicode hex with space returns error",
			input: `"\u00 0"`,
		},
		{
			name:  "invalid unicode hex with minus returns error",
			input: `"\u-001"`,
		},
		{
			name:  "incomplete unicode sequence at end returns error",
			input: `"hello\u123"`,
		},
		{
			name:  "backslash at very end returns error",
			input: `"test\`,
		},
		{
			name:  "only backslash and quote returns error",
			input: `"\"`,
		},
		{
			name:  "unterminated string returns error",
			input: `"hello world`,
		},
		{
			name:  "single quotes instead of double quotes returns error",
			input: "'hello'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := jsonlite.Unquote(tt.input)
			if err == nil {
				t.Errorf("Unquote(%q) expected error, got %q", tt.input, got)
			}
		})
	}
}

func BenchmarkUnquote(b *testing.B) {
	inputs := []struct {
		name  string
		value string
	}{
		{"empty", `""`},
		{"short", `"hello"`},
		{"medium", `"The quick brown fox jumps over the lazy dog"`},
		{"with_escapes", `"Hello, \"World\"!\nHow are you?"`},
		{"unicode", `"Hello \u0048\u0065\u006c\u006c\u006f"`},
		{"long_no_escape", func() string {
			s := make([]byte, 1024)
			for i := range s {
				s[i] = 'a'
			}
			return `"` + string(s) + `"`
		}()},
		{"long_with_escape", func() string {
			s := make([]byte, 1024)
			for i := range s {
				if i%100 == 0 {
					s[i] = '\\'
				} else if i%100 == 1 {
					s[i] = 'n'
				} else {
					s[i] = 'a'
				}
			}
			return `"` + string(s) + `"`
		}()},
	}

	for _, input := range inputs {
		b.Run(input.name, func(b *testing.B) {
			b.SetBytes(int64(len(input.value)))
			for i := 0; i < b.N; i++ {
				_, _ = jsonlite.Unquote(input.value)
			}
		})
	}
}

func BenchmarkAppendUnquote(b *testing.B) {
	inputs := []struct {
		name  string
		value string
	}{
		{"empty", `""`},
		{"short", `"hello"`},
		{"medium", `"The quick brown fox jumps over the lazy dog"`},
		{"with_escapes", `"Hello, \"World\"!\nHow are you?"`},
		{"unicode", `"Hello \u0048\u0065\u006c\u006c\u006f"`},
		{"long_no_escape", func() string {
			s := make([]byte, 1024)
			for i := range s {
				s[i] = 'a'
			}
			return `"` + string(s) + `"`
		}()},
		{"long_with_escape", func() string {
			s := make([]byte, 1024)
			for i := range s {
				if i%100 == 0 {
					s[i] = '\\'
				} else if i%100 == 1 {
					s[i] = 'n'
				} else {
					s[i] = 'a'
				}
			}
			return `"` + string(s) + `"`
		}()},
	}

	for _, input := range inputs {
		b.Run(input.name, func(b *testing.B) {
			buf := make([]byte, 0, 2048)
			b.SetBytes(int64(len(input.value)))
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				buf, _ = jsonlite.AppendUnquote(buf[:0], input.value)
			}
		})
	}
}

// TestUnquoteUnpairedSurrogates pins the behaviour of surrogate escapes that
// do not form a pair. They are not an error: RFC 8259 leaves the handling to
// the implementation, and jsonlite follows encoding/json in decoding them to
// the replacement character.
//
// These cases previously returned an error from Unquote. Since Value.String
// has no error to return and discarded it, that error surfaced as a silently
// truncated string -- `"\ud800x"` read back as "" rather than "�x" --
// which is why the behaviour changed rather than the error being propagated.
func TestUnquoteUnpairedSurrogates(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"lone high surrogate", `"\ud83d"`, "�"},
		{"high surrogate then text", `"\ud83dtext"`, "�text"},
		{"high surrogate then non-surrogate escape", `"\ud83dA"`, "�A"},
		{"lone low surrogate", `"\ude00"`, "�"},
		{"low surrogate alone", `"\udc96"`, "�"},
		{"high surrogate then out-of-range escape", `"\ud83d￿"`, "�￿"},
		{"two high surrogates", `"\ud800\ud800"`, "��"},
		{"high surrogate then newline escape", `"\ud800\n"`, "�\n"},
		{"unpaired surrogate between text", `"a\ud800b"`, "a�b"},
		{"valid pair still decodes", `"😀"`, "\U0001F600"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := jsonlite.Unquote(tt.input)
			if err != nil {
				t.Fatalf("Unquote(%q) returned error: %v", tt.input, err)
			}
			if got != tt.want {
				t.Errorf("Unquote(%q) = %q, want %q", tt.input, got, tt.want)
			}
			// The whole point is agreeing with encoding/json here.
			var std string
			if err := json.Unmarshal([]byte(tt.input), &std); err != nil {
				t.Fatalf("encoding/json rejected %q: %v", tt.input, err)
			}
			if got != std {
				t.Errorf("Unquote(%q) = %q, encoding/json = %q", tt.input, got, std)
			}
		})
	}
}

// TestEscapedAgreesWithByteScan pins the word-at-a-time scan against a plain
// byte loop. The unrolled path folds four words together before testing, so a
// mistake would only show at particular lengths and offsets: this walks every
// length across both loop tiers and puts the escape at every position.
func TestEscapedAgreesWithByteScan(t *testing.T) {
	base := "abcdefghijklmnopqrstuvwxyz0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ!@#$%^&*()_+-="
	for n := 0; n <= 72; n++ {
		clean := base[:n]
		if got, want := jsonlite.Escaped(clean), referenceEscaped(clean); got != want {
			t.Errorf("len %d clean: got %v want %v", n, got, want)
		}
		for pos := range n {
			for _, bad := range []byte{'\\', 0x00, 0x1f, '\t', '\n'} {
				b := []byte(clean)
				b[pos] = bad
				s := string(b)
				if got, want := jsonlite.Escaped(s), referenceEscaped(s); got != want {
					t.Errorf("len %d pos %d byte %#x: got %v want %v", n, pos, bad, got, want)
				}
			}
		}
		// High bytes must not trigger the slow path.
		if n >= 4 {
			b := []byte(clean)
			b[n/2] = 0xC3
			b[n/2+1] = 0xA9
			s := string(b)
			if got, want := jsonlite.Escaped(s), referenceEscaped(s); got != want {
				t.Errorf("len %d utf8: got %v want %v", n, got, want)
			}
		}
	}
}

// TestEscapedWideAgreesWithPortable pins the vector scan against the
// word-at-a-time one. It only does real work on a build where the vector
// implementation exists, but it compiles and passes everywhere.
func TestEscapedWideAgreesWithPortable(t *testing.T) {
	if !jsonlite.EscapedHasWide() {
		t.Skip("vector escape scan not built in")
	}
	base := strings.Repeat("abcdefghijklmnopqrstuvwxyz0123456789", 8)
	for n := 0; n <= 200; n++ {
		clean := base[:n]
		if got, want := jsonlite.EscapedWide(clean), referenceEscaped(clean); got != want {
			t.Fatalf("len %d clean: wide=%v reference=%v", n, got, want)
		}
		for _, pos := range []int{0, 1, 31, 32, 33, 63, 64, 65, n - 1} {
			if pos < 0 || pos >= n {
				continue
			}
			for _, bad := range []byte{'\\', 0x00, 0x1f, '\n', 0xC3, 0xFF} {
				b := []byte(clean)
				b[pos] = bad
				s := string(b)
				if got, want := jsonlite.EscapedWide(s), referenceEscaped(s); got != want {
					t.Fatalf("len %d pos %d byte %#x: wide=%v reference=%v", n, pos, bad, got, want)
				}
			}
		}
	}
}

// referenceEscaped is the obvious byte loop the scans must agree with.
func referenceEscaped(s string) bool {
	for i := range len(s) {
		if c := s[i]; c < 0x20 || c == '\\' {
			return true
		}
	}
	return false
}
