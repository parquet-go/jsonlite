package jsonlite

import (
	"strconv"
	"testing"
)

func TestQuote(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "empty string",
			input: "",
			want:  `""`,
		},
		{
			name:  "simple string",
			input: "hello",
			want:  `"hello"`,
		},
		{
			name:  "string with spaces",
			input: "hello world",
			want:  `"hello world"`,
		},
		{
			name:  "string with quote",
			input: `say "hello"`,
			want:  `"say \"hello\""`,
		},
		{
			name:  "string with backslash",
			input: `path\to\file`,
			want:  `"path\\to\\file"`,
		},
		{
			name:  "string with newline",
			input: "line1\nline2",
			want:  `"line1\nline2"`,
		},
		{
			name:  "string with tab",
			input: "col1\tcol2",
			want:  `"col1\tcol2"`,
		},
		{
			name:  "string with carriage return",
			input: "line1\rline2",
			want:  `"line1\rline2"`,
		},
		{
			name:  "string with backspace",
			input: "back\bspace",
			want:  `"back\bspace"`,
		},
		{
			name:  "string with form feed",
			input: "form\ffeed",
			want:  `"form\ffeed"`,
		},
		{
			name:  "control character NUL",
			input: "hello\x00world",
			want:  `"hello\u0000world"`,
		},
		{
			name:  "control character SOH",
			input: "hello\x01world",
			want:  `"hello\u0001world"`,
		},
		{
			name:  "control character US",
			input: "hello\x1fworld",
			want:  `"hello\u001fworld"`,
		},
		{
			name:  "DEL character (0x7F)",
			input: "hello\x7fworld",
			want:  "\"hello\x7fworld\"", // DEL is valid unescaped in JSON (only 0x00-0x1F must be escaped)
		},
		{
			name:  "non-ASCII byte",
			input: "hello\x80world",
			want:  `"hello\u0080world"`,
		},
		{
			name:  "UTF-8 multibyte",
			input: "café",
			want:  `"caf\u00c3\u00a9"`, // UTF-8 bytes of é are 0xC3 0xA9
		},
		{
			name:  "all escape types",
			input: "\"\\/\b\f\n\r\t",
			want:  `"\"\\/\b\f\n\r\t"`,
		},
		{
			name:  "mixed content",
			input: "Hello, \"World\"!\nHow are you?",
			want:  `"Hello, \"World\"!\nHow are you?"`,
		},
		{
			name:  "long string without escapes",
			input: "abcdefghijklmnopqrstuvwxyz0123456789",
			want:  `"abcdefghijklmnopqrstuvwxyz0123456789"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Quote(tt.input)
			if got != tt.want {
				t.Errorf("Quote(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestAppendQuote(t *testing.T) {
	// Test that AppendQuote correctly appends to existing buffer
	prefix := []byte("prefix:")
	result := AppendQuote(prefix, "hello")
	want := `prefix:"hello"`
	if string(result) != want {
		t.Errorf("AppendQuote with prefix = %q, want %q", string(result), want)
	}
}

func TestQuoteMatchesStrconv(t *testing.T) {
	// Test that Quote produces the same output as strconv.Quote for valid ASCII strings
	// (excluding high bytes where behavior differs)
	inputs := []string{
		"",
		"hello",
		"hello world",
		"line1\nline2",
		"col1\tcol2",
		`path\to\file`,
		`say "hello"`,
		"\b\f\n\r\t",
	}

	for _, input := range inputs {
		got := Quote(input)
		want := strconv.Quote(input)
		if got != want {
			t.Errorf("Quote(%q) = %q, strconv.Quote = %q", input, got, want)
		}
	}
}

func TestQuoteRoundTrip(t *testing.T) {
	// Test that Quote/Unquote round-trip correctly
	inputs := []string{
		"",
		"hello",
		"hello world",
		"line1\nline2",
		"col1\tcol2",
		`path\to\file`,
		`say "hello"`,
		"\b\f\n\r\t",
		"control\x00char",
		"control\x1fchar",
	}

	for _, input := range inputs {
		quoted := Quote(input)
		unquoted, err := Unquote(quoted)
		if err != nil {
			t.Errorf("Unquote(Quote(%q)) error: %v", input, err)
			continue
		}
		if unquoted != input {
			t.Errorf("Round-trip failed: input=%q, quoted=%q, unquoted=%q", input, quoted, unquoted)
		}
	}
}

func TestQuoteUnquoteMultipleLevels(t *testing.T) {
	// Test that applying Quote multiple times produces different outputs each time,
	// and applying Unquote the same number of times recovers the original.
	inputs := []string{
		`{"name":"test","value":42}`,
		`{"message":"Hello, \"World\"!"}`,
		`{"path":"C:\\Users\\test"}`,
		`{"multiline":"line1\nline2\nline3"}`,
		`{"nested":{"a":1,"b":2}}`,
		`["item1","item2","item3"]`,
		`{"special":"\t\r\n"}`,
	}

	for _, original := range inputs {
		t.Run(original[:min(20, len(original))], func(t *testing.T) {
			const levels = 5

			// Apply Quote multiple times, verify each level is different
			values := make([]string, levels+1)
			values[0] = original

			for i := 1; i <= levels; i++ {
				values[i] = Quote(values[i-1])
				if values[i] == values[i-1] {
					t.Errorf("Quote level %d produced same output as level %d: %q", i, i-1, values[i])
				}
			}

			// Apply Unquote the same number of times
			current := values[levels]
			for i := levels; i >= 1; i-- {
				unquoted, err := Unquote(current)
				if err != nil {
					t.Fatalf("Unquote at level %d failed: %v (input: %q)", i, err, current)
				}
				if unquoted != values[i-1] {
					t.Errorf("Unquote at level %d: got %q, want %q", i, unquoted, values[i-1])
				}
				current = unquoted
			}

			// Verify we're back to the original
			if current != original {
				t.Errorf("After %d Quote/Unquote cycles: got %q, want %q", levels, current, original)
			}
		})
	}
}

func TestEscapeIndex(t *testing.T) {
	tests := []struct {
		input string
		want  int
	}{
		{"", -1},
		{"hello", -1},
		{"hello world", -1},
		{`"`, 0},
		{`hello"`, 5},
		{`\`, 0},
		{`hello\`, 5},
		{"\n", 0},
		{"hello\n", 5},
		{"\x00", 0},
		{"hello\x00", 5},
		{"\x1f", 0},
		{"hello\x1f", 5},
		{"\x7f", -1}, // DEL (0x7F) doesn't need escaping in JSON
		{"hello\x7f", -1},
		{"\x80", 0},
		{"hello\x80", 5},
		// Test with string longer than 8 bytes to exercise SIMD path
		{"abcdefghijklmnop", -1},
		{"abcdefgh\"jklmnop", 8},
		{"abcdefghijklmno\"", 15},
	}

	for _, tt := range tests {
		got := escapeIndex(tt.input)
		if got != tt.want {
			t.Errorf("escapeIndex(%q) = %d, want %d", tt.input, got, tt.want)
		}
	}
}

func BenchmarkQuote(b *testing.B) {
	inputs := []struct {
		name  string
		value string
	}{
		{"empty", ""},
		{"short", "hello"},
		{"medium", "The quick brown fox jumps over the lazy dog"},
		{"with_escapes", "Hello, \"World\"!\nHow are you?"},
		{"long", func() string {
			s := make([]byte, 1024)
			for i := range s {
				s[i] = 'a'
			}
			return string(s)
		}()},
	}

	for _, input := range inputs {
		b.Run(input.name, func(b *testing.B) {
			b.SetBytes(int64(len(input.value)))
			for i := 0; i < b.N; i++ {
				_ = Quote(input.value)
			}
		})
	}
}

func BenchmarkAppendQuote(b *testing.B) {
	inputs := []struct {
		name  string
		value string
	}{
		{"empty", ""},
		{"short", "hello"},
		{"medium", "The quick brown fox jumps over the lazy dog"},
		{"with_escapes", "Hello, \"World\"!\nHow are you?"},
		{"long_no_escape", func() string {
			s := make([]byte, 1024)
			for i := range s {
				s[i] = 'a'
			}
			return string(s)
		}()},
		{"long_with_escape", func() string {
			s := make([]byte, 1024)
			for i := range s {
				if i%100 == 0 {
					s[i] = '\n'
				} else {
					s[i] = 'a'
				}
			}
			return string(s)
		}()},
	}

	for _, input := range inputs {
		b.Run(input.name, func(b *testing.B) {
			buf := make([]byte, 0, 2048)
			b.SetBytes(int64(len(input.value)))
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				buf = AppendQuote(buf[:0], input.value)
			}
		})
	}
}
