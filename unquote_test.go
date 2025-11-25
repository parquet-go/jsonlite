package jsonlite_test

import (
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
			name:     "EmptyString",
			input:    `""`,
			expected: "",
		},
		{
			name:     "SimpleString",
			input:    `"hello"`,
			expected: "hello",
		},
		{
			name:     "StringWithSpaces",
			input:    `"hello world"`,
			expected: "hello world",
		},
		{
			name:     "EscapedQuote",
			input:    `"say \"hello\""`,
			expected: `say "hello"`,
		},
		{
			name:     "EscapedBackslash",
			input:    `"path\\to\\file"`,
			expected: `path\to\file`,
		},
		{
			name:     "EscapedSlash",
			input:    `"a\/b"`,
			expected: "a/b",
		},
		{
			name:     "EscapedBackspace",
			input:    `"a\bb"`,
			expected: "a\bb",
		},
		{
			name:     "EscapedFormfeed",
			input:    `"a\fb"`,
			expected: "a\fb",
		},
		{
			name:     "EscapedNewline",
			input:    `"line1\nline2"`,
			expected: "line1\nline2",
		},
		{
			name:     "EscapedCarriageReturn",
			input:    `"line1\rline2"`,
			expected: "line1\rline2",
		},
		{
			name:     "EscapedTab",
			input:    `"col1\tcol2"`,
			expected: "col1\tcol2",
		},
		{
			name:     "UnicodeNull",
			input:    `"\u0000"`,
			expected: "\u0000",
		},
		{
			name:     "UnicodeASCII",
			input:    `"\u0041"`,
			expected: "A",
		},
		{
			name:     "UnicodeMultiByte",
			input:    `"\u4e2d\u6587"`,
			expected: "中文",
		},
		{
			name:     "UnicodeMax",
			input:    `"\uffff"`,
			expected: "\uffff",
		},
		{
			name:     "MixedEscapes",
			input:    `"line1\nline2\ttab\u0041end"`,
			expected: "line1\nline2\ttabAend",
		},
		{
			name:     "MultipleQuotes",
			input:    `"\"quote1\" and \"quote2\""`,
			expected: `"quote1" and "quote2"`,
		},
		{
			name:     "AllSingleCharEscapes",
			input:    `"\"\\\//\b\f\n\r\t"`,
			expected: "\"\\//\b\f\n\r\t",
		},
		{
			name:     "OnlyEscapedChars",
			input:    `"\n\t"`,
			expected: "\n\t",
		},
		{
			name:     "LongString",
			input:    `"The quick brown fox jumps over the lazy dog"`,
			expected: "The quick brown fox jumps over the lazy dog",
		},
		{
			name:     "StringWithNumbers",
			input:    `"test123"`,
			expected: "test123",
		},
		{
			name:     "JSONValue",
			input:    `"{\"key\":\"value\"}"`,
			expected: `{"key":"value"}`,
		},
		{
			name:     "UnicodeLowercaseHex",
			input:    `"\u00e9"`,
			expected: "é",
		},
		{
			name:     "UnicodeEmoji",
			input:    `"\ud83d\ude00"`,
			expected: "😀",
		},
		{
			name:     "UnicodeSurrogatePairHeart",
			input:    `"\ud83d\udc96"`,
			expected: "💖",
		},
		{
			name:     "UnicodeSurrogatePairRocket",
			input:    `"\ud83d\ude80"`,
			expected: "🚀",
		},
		{
			name:     "MultipleEmojis",
			input:    `"\ud83d\ude00\ud83d\udc96"`,
			expected: "😀💖",
		},
		{
			name:     "EmojiWithText",
			input:    `"Hello \ud83d\udc4b World"`,
			expected: "Hello 👋 World",
		},
		{
			name:     "ConsecutiveEscapes",
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
			name:  "NoQuotes",
			input: "hello",
		},
		{
			name:  "OnlyOpeningQuote",
			input: `"hello`,
		},
		{
			name:  "OnlyClosingQuote",
			input: `hello"`,
		},
		{
			name:  "SingleQuote",
			input: `"`,
		},
		{
			name:  "EmptyInput",
			input: "",
		},
		{
			name:  "TrailingBackslash",
			input: `"hello\`,
		},
		{
			name:  "TrailingBackslashBeforeQuote",
			input: `"hello\"`,
		},
		{
			name:  "InvalidEscapeX",
			input: `"hello\x"`,
		},
		{
			name:  "InvalidEscapeV",
			input: `"hello\v"`,
		},
		{
			name:  "InvalidEscapeA",
			input: `"hello\a"`,
		},
		{
			name:  "InvalidEscapeDigit",
			input: `"hello\0"`,
		},
		{
			name:  "IncompleteUnicode3Chars",
			input: `"\u041"`,
		},
		{
			name:  "IncompleteUnicode2Chars",
			input: `"\u04"`,
		},
		{
			name:  "IncompleteUnicode1Char",
			input: `"\u0"`,
		},
		{
			name:  "IncompleteUnicodeNoChars",
			input: `"\u"`,
		},
		{
			name:  "InvalidUnicodeHexG",
			input: `"\u00GG"`,
		},
		{
			name:  "InvalidUnicodeHexSpace",
			input: `"\u00 0"`,
		},
		{
			name:  "InvalidUnicodeHexMinus",
			input: `"\u-001"`,
		},
		{
			name:  "UnicodeAtEnd",
			input: `"hello\u123"`,
		},
		{
			name:  "BackslashAtVeryEnd",
			input: `"test\`,
		},
		{
			name:  "OnlyBackslash",
			input: `"\"`,
		},
		{
			name:  "UnterminatedString",
			input: `"hello world`,
		},
		{
			name:  "WrongQuoteType",
			input: "'hello'",
		},
		{
			name:  "HighSurrogateWithoutLow",
			input: `"\ud83d"`,
		},
		{
			name:  "HighSurrogateWithText",
			input: `"\ud83dtext"`,
		},
		{
			name:  "HighSurrogateWithNormalUnicode",
			input: `"\ud83d\u0041"`,
		},
		{
			name:  "LowSurrogateWithoutHigh",
			input: `"\ude00"`,
		},
		{
			name:  "LowSurrogateAlone",
			input: `"\udc96"`,
		},
		{
			name:  "HighSurrogateWithInvalidLow",
			input: `"\ud83d\uffff"`,
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
