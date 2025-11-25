package jsonlite_test

import (
	"testing"

	"github.com/parquet-go/jsonlite"
)

func TestParseNull(t *testing.T) {
	val, err := jsonlite.Parse("null")
	if err != nil {
		t.Fatal(err)
	}
	if val.Kind() != jsonlite.Null {
		t.Errorf("expected Null, got %v", val.Kind())
	}
}

func TestParseBool(t *testing.T) {
	tests := []struct {
		input    string
		expected jsonlite.Kind
	}{
		{"true", jsonlite.True},
		{"false", jsonlite.False},
	}

	for _, tt := range tests {
		val, err := jsonlite.Parse(tt.input)
		if err != nil {
			t.Fatalf("parse %q: %v", tt.input, err)
		}
		if val.Kind() != tt.expected {
			t.Errorf("%q: expected %v, got %v", tt.input, tt.expected, val.Kind())
		}
	}
}

func TestParseNumber(t *testing.T) {
	tests := []string{
		"42",
		"-42",
		"0",
		"9223372036854775807",  // large int
		"18446744073709551615", // very large int
		"3.14",
		"-3.14",
		"1e10",
		"1.5e-10",
	}

	for _, input := range tests {
		val, err := jsonlite.Parse(input)
		if err != nil {
			t.Fatalf("parse %q: %v (type: %T)", input, err, err)
		}
		if val.Kind() != jsonlite.Number {
			t.Errorf("%q: expected Number, got %v", input, val.Kind())
		}
	}
}

func TestParseString(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`"hello"`, "hello"},
		{`""`, ""},
		{`"hello world"`, "hello world"},
		{`"with\nnewline"`, "with\nnewline"},
		{`"with\ttab"`, "with\ttab"},
		{`"with\"quote"`, `with"quote`},
		{`"unicode: \u0048\u0065\u006c\u006c\u006f"`, "unicode: Hello"},
	}

	for _, tt := range tests {
		val, err := jsonlite.Parse(tt.input)
		if err != nil {
			t.Fatalf("parse %q: %v", tt.input, err)
		}
		if val.Kind() != jsonlite.String {
			t.Errorf("%q: expected String, got %v", tt.input, val.Kind())
		}
		if val.String() != tt.expected {
			t.Errorf("%q: expected %q, got %q", tt.input, tt.expected, val.String())
		}
	}
}

func TestParseArray(t *testing.T) {
	tests := []struct {
		input    string
		expected int
	}{
		{"[]", 0},
		{"[1]", 1},
		{"[1,2,3]", 3},
		{`["a","b","c"]`, 3},
		{"[1,2,3,4,5,6,7,8,9,10]", 10},
	}

	for _, tt := range tests {
		val, err := jsonlite.Parse(tt.input)
		if err != nil {
			t.Fatalf("parse %q: %v", tt.input, err)
		}
		if val.Kind() != jsonlite.Array {
			t.Errorf("%q: expected Array, got %v", tt.input, val.Kind())
		}
		if val.Len() != tt.expected {
			t.Errorf("%q: expected length %d, got %d", tt.input, tt.expected, val.Len())
		}
	}
}

func TestParseObject(t *testing.T) {
	tests := []struct {
		input    string
		expected int
	}{
		{"{}", 0},
		{`{"a":1}`, 1},
		{`{"a":1,"b":2}`, 2},
		{`{"a":1,"b":2,"c":3}`, 3},
	}

	for _, tt := range tests {
		val, err := jsonlite.Parse(tt.input)
		if err != nil {
			t.Fatalf("parse %q: %v", tt.input, err)
		}
		if val.Kind() != jsonlite.Object {
			t.Errorf("%q: expected Object, got %v", tt.input, val.Kind())
		}
		if val.Len() != tt.expected {
			t.Errorf("%q: expected length %d, got %d", tt.input, tt.expected, val.Len())
		}
	}
}

func TestParseNested(t *testing.T) {
	input := `{
		"name": "test",
		"age": 42,
		"active": true,
		"tags": ["a", "b", "c"],
		"metadata": {
			"created": "2024-01-01",
			"updated": null
		}
	}`

	val, err := jsonlite.Parse(input)
	if err != nil {
		t.Fatal(err)
	}

	if val.Kind() != jsonlite.Object {
		t.Fatalf("expected Object, got %v", val.Kind())
	}

	fields := val.Object()
	if len(fields) != 5 {
		t.Fatalf("expected 5 fields, got %d", len(fields))
	}

	// Check "tags" array
	var tagsField *jsonlite.Field
	for i := range fields {
		if fields[i].Key == "tags" {
			tagsField = &fields[i]
			break
		}
	}
	if tagsField == nil {
		t.Fatal("tags field not found")
	}
	if tagsField.Val.Kind() != jsonlite.Array {
		t.Fatalf("tags: expected Array, got %v", tagsField.Val.Kind())
	}
	if tagsField.Val.Len() != 3 {
		t.Fatalf("tags: expected length 3, got %d", tagsField.Val.Len())
	}

	// Check "metadata" object
	var metadataField *jsonlite.Field
	for i := range fields {
		if fields[i].Key == "metadata" {
			metadataField = &fields[i]
			break
		}
	}
	if metadataField == nil {
		t.Fatal("metadata field not found")
	}
	if metadataField.Val.Kind() != jsonlite.Object {
		t.Fatalf("metadata: expected Object, got %v", metadataField.Val.Kind())
	}
}

func TestSerializeRoundTrip(t *testing.T) {
	tests := []string{
		"null",
		"true",
		"false",
		"42",
		"3.14",
		`"hello"`,
		"[]",
		"[1,2,3]",
		"{}",
		`{"a":1}`,
		`{"a":1,"b":"hello","c":true,"d":null}`,
		`[1,"two",true,null,{"nested":"object"}]`,
		`{"array":[1,2,3],"object":{"nested":true}}`,
	}

	for _, input := range tests {
		val, err := jsonlite.Parse(input)
		if err != nil {
			t.Fatalf("parse %q: %v", input, err)
		}

		serialized := val.Append(nil)

		val2, err := jsonlite.Parse(string(serialized))
		if err != nil {
			t.Fatalf("parse serialized %q: %v", serialized, err)
		}

		if !valuesEqual(*val, *val2) {
			t.Errorf("round-trip failed for %q\ngot: %s", input, serialized)
		}
	}
}

func valuesEqual(a, b jsonlite.Value) bool {
	if a.Kind() != b.Kind() {
		return false
	}

	switch a.Kind() {
	case jsonlite.Null, jsonlite.True, jsonlite.False:
		return true
	case jsonlite.Number:
		return a.Float() == b.Float()
	case jsonlite.String:
		return a.String() == b.String()
	case jsonlite.Array:
		aArr := a.Array()
		bArr := b.Array()
		if len(aArr) != len(bArr) {
			return false
		}
		for i := range aArr {
			if !valuesEqual(aArr[i], bArr[i]) {
				return false
			}
		}
		return true
	case jsonlite.Object:
		aObj := a.Object()
		bObj := b.Object()
		if len(aObj) != len(bObj) {
			return false
		}
		// Note: field order might differ, so we need to match by key
		for _, af := range aObj {
			found := false
			for _, bf := range bObj {
				if af.Key == bf.Key {
					if !valuesEqual(af.Val, bf.Val) {
						return false
					}
					found = true
					break
				}
			}
			if !found {
				return false
			}
		}
		return true
	}
	return false
}

func TestAccessors(t *testing.T) {
	input := `{"num1":42,"num2":18446744073709551615,"num3":3.14,"string":"hello","bool":true,"null":null,"array":[1,2],"object":{"a":1}}`
	val, err := jsonlite.Parse(input)
	if err != nil {
		t.Fatal(err)
	}

	fields := val.Object()
	fieldMap := make(map[string]jsonlite.Value)
	for _, f := range fields {
		fieldMap[f.Key] = f.Val
	}

	// Test number accessor - all numbers are float64
	if num1Val, ok := fieldMap["num1"]; ok {
		if num1Val.Kind() != jsonlite.Number {
			t.Errorf("num1: expected Number, got %v", num1Val.Kind())
		}
		if num1Val.Float() != 42 {
			t.Errorf("num1: expected 42, got %f", num1Val.Float())
		}
	} else {
		t.Error("num1 field not found")
	}

	// Test large number
	if num2Val, ok := fieldMap["num2"]; ok {
		if num2Val.Kind() != jsonlite.Number {
			t.Errorf("num2: expected Number, got %v", num2Val.Kind())
		}
	} else {
		t.Error("num2 field not found")
	}

	// Test float
	if num3Val, ok := fieldMap["num3"]; ok {
		if num3Val.Kind() != jsonlite.Number {
			t.Errorf("num3: expected Number, got %v", num3Val.Kind())
		}
		if num3Val.Float() != 3.14 {
			t.Errorf("num3: expected 3.14, got %f", num3Val.Float())
		}
	} else {
		t.Error("num3 field not found")
	}

	// Test string accessor
	if strVal, ok := fieldMap["string"]; ok {
		if strVal.Kind() != jsonlite.String {
			t.Errorf("string: expected String, got %v", strVal.Kind())
		}
		if strVal.String() != "hello" {
			t.Errorf("string: expected 'hello', got %q", strVal.String())
		}
	} else {
		t.Error("string field not found")
	}

	// Test bool accessor
	if boolVal, ok := fieldMap["bool"]; ok {
		if boolVal.Kind() != jsonlite.True {
			t.Errorf("bool: expected True, got %v", boolVal.Kind())
		}
	} else {
		t.Error("bool field not found")
	}

	// Test null
	if nullVal, ok := fieldMap["null"]; ok {
		if nullVal.Kind() != jsonlite.Null {
			t.Errorf("null: expected Null, got %v", nullVal.Kind())
		}
	} else {
		t.Error("null field not found")
	}

	// Test array accessor
	if arrayVal, ok := fieldMap["array"]; ok {
		if arrayVal.Kind() != jsonlite.Array {
			t.Errorf("array: expected Array, got %v", arrayVal.Kind())
		}
		arr := arrayVal.Array()
		if len(arr) != 2 {
			t.Errorf("array: expected length 2, got %d", len(arr))
		}
	} else {
		t.Error("array field not found")
	}

	// Test object accessor
	if objVal, ok := fieldMap["object"]; ok {
		if objVal.Kind() != jsonlite.Object {
			t.Errorf("object: expected Object, got %v", objVal.Kind())
		}
		obj := objVal.Object()
		if len(obj) != 1 {
			t.Errorf("object: expected length 1, got %d", len(obj))
		}
	} else {
		t.Error("object field not found")
	}
}

func TestParseError(t *testing.T) {
	tests := []string{
		"{",
		`{"unclosed": "string}`,
	}

	for _, input := range tests {
		_, err := jsonlite.Parse(input)
		if err == nil {
			t.Errorf("expected error for %q, got nil", input)
		}
	}
}

func TestLookup(t *testing.T) {
	input := `{"a":1,"b":"hello","c":true}`
	val, err := jsonlite.Parse(input)
	if err != nil {
		t.Fatal(err)
	}

	// Test existing keys
	if v := val.Lookup("a"); v == nil {
		t.Error("expected to find key 'a'")
	} else if v.Kind() != jsonlite.Number {
		t.Errorf("expected Number for 'a', got %v", v.Kind())
	}

	if v := val.Lookup("b"); v == nil {
		t.Error("expected to find key 'b'")
	} else if v.String() != "hello" {
		t.Errorf("expected 'hello' for 'b', got %q", v.String())
	}

	// Test non-existing key
	if v := val.Lookup("nonexistent"); v != nil {
		t.Error("expected nil for non-existing key")
	}
}

func TestNumberType(t *testing.T) {
	tests := []struct {
		input    string
		expected jsonlite.NumberType
	}{
		{"42", jsonlite.Uint},
		{"-42", jsonlite.Int},
		{"0", jsonlite.Uint},
		{"3.14", jsonlite.Float},
		{"-3.14", jsonlite.Float},
		{"1e10", jsonlite.Float},
	}

	for _, tt := range tests {
		val, err := jsonlite.Parse(tt.input)
		if err != nil {
			t.Fatalf("parse %q: %v", tt.input, err)
		}
		if val.NumberType() != tt.expected {
			t.Errorf("%q: expected %v, got %v", tt.input, tt.expected, val.NumberType())
		}
	}
}

func TestNumber(t *testing.T) {
	val, err := jsonlite.Parse("42")
	if err != nil {
		t.Fatal(err)
	}
	num := val.Number()
	if string(num) != "42" {
		t.Errorf("expected \"42\", got %q", num)
	}
	i, err := num.Int64()
	if err != nil {
		t.Fatal(err)
	}
	if i != 42 {
		t.Errorf("expected 42, got %d", i)
	}
}

func TestNumberTypeOf(t *testing.T) {
	tests := []struct {
		input    string
		expected jsonlite.NumberType
	}{
		{"42", jsonlite.Uint},
		{"-42", jsonlite.Int},
		{"0", jsonlite.Uint},
		{"3.14", jsonlite.Float},
		{"-3.14", jsonlite.Float},
		{"1e10", jsonlite.Float},
		{"", jsonlite.Float},
	}

	for _, tt := range tests {
		if got := jsonlite.NumberTypeOf(tt.input); got != tt.expected {
			t.Errorf("NumberTypeOf(%q): expected %v, got %v", tt.input, tt.expected, got)
		}
	}
}

func BenchmarkParse(b *testing.B) {
	benchmarks := []struct {
		name  string
		input string
	}{
		{
			name:  "Small",
			input: `{"name":"test","age":42,"active":true}`,
		},
		{
			name: "Medium",
			input: `{
				"name": "test",
				"age": 42,
				"active": true,
				"tags": ["a", "b", "c"],
				"metadata": {
					"created": "2024-01-01",
					"updated": null
				}
			}`,
		},
		{
			name: "Large",
			input: `{
				"users": [
					{"id":1,"name":"Alice","email":"alice@example.com","active":true},
					{"id":2,"name":"Bob","email":"bob@example.com","active":false},
					{"id":3,"name":"Charlie","email":"charlie@example.com","active":true}
				],
				"metadata": {
					"total": 3,
					"page": 1,
					"perPage": 10,
					"filters": {
						"status": "active",
						"role": ["admin", "user"],
						"createdAfter": "2024-01-01T00:00:00Z"
					}
				},
				"stats": {
					"activeUsers": 2,
					"inactiveUsers": 1,
					"averageAge": 28.5,
					"tags": ["production", "verified", "premium"]
				}
			}`,
		},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			b.ReportAllocs()
			b.SetBytes(int64(len(bm.input)))
			for b.Loop() {
				_, err := jsonlite.Parse(bm.input)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func TestStringReturnsJSON(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"null", "null"},
		{"true", "true"},
		{"false", "false"},
		{"[1,2,3]", "[1,2,3]"},
		{`{"a":1}`, `{"a":1}`},
	}

	for _, tt := range tests {
		val, err := jsonlite.Parse(tt.input)
		if err != nil {
			t.Fatalf("parse %q: %v", tt.input, err)
		}
		got := val.String()
		if got != tt.expected {
			t.Errorf("String() for %q: got %q, want %q", tt.input, got, tt.expected)
		}
	}
}

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

// Test safety checks - these should panic when called on wrong types
func TestValueSafetyChecks(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		operation func(*jsonlite.Value)
	}{
		{
			name:  "IntOnString",
			input: `"hello"`,
			operation: func(v *jsonlite.Value) {
				_ = v.Int()
			},
		},
		{
			name:  "IntOnArray",
			input: "[1,2,3]",
			operation: func(v *jsonlite.Value) {
				_ = v.Int()
			},
		},
		{
			name:  "FloatOnString",
			input: `"hello"`,
			operation: func(v *jsonlite.Value) {
				_ = v.Float()
			},
		},
		{
			name:  "UintOnObject",
			input: `{"a":1}`,
			operation: func(v *jsonlite.Value) {
				_ = v.Uint()
			},
		},
		{
			name:  "ArrayOnString",
			input: `"hello"`,
			operation: func(v *jsonlite.Value) {
				_ = v.Array()
			},
		},
		{
			name:  "ArrayOnNumber",
			input: "42",
			operation: func(v *jsonlite.Value) {
				_ = v.Array()
			},
		},
		{
			name:  "ObjectOnString",
			input: `"hello"`,
			operation: func(v *jsonlite.Value) {
				_ = v.Object()
			},
		},
		{
			name:  "ObjectOnArray",
			input: "[1,2,3]",
			operation: func(v *jsonlite.Value) {
				_ = v.Object()
			},
		},
		{
			name:  "LookupOnArray",
			input: "[1,2,3]",
			operation: func(v *jsonlite.Value) {
				_ = v.Lookup("key")
			},
		},
		{
			name:  "LookupOnString",
			input: `"hello"`,
			operation: func(v *jsonlite.Value) {
				_ = v.Lookup("key")
			},
		},
		{
			name:  "NumberOnString",
			input: `"hello"`,
			operation: func(v *jsonlite.Value) {
				_ = v.Number()
			},
		},
		{
			name:  "NumberOnArray",
			input: "[1,2,3]",
			operation: func(v *jsonlite.Value) {
				_ = v.Number()
			},
		},
		{
			name:  "LenOnNull",
			input: "null",
			operation: func(v *jsonlite.Value) {
				_ = v.Len()
			},
		},
		{
			name:  "LenOnTrue",
			input: "true",
			operation: func(v *jsonlite.Value) {
				_ = v.Len()
			},
		},
		{
			name:  "LenOnFalse",
			input: "false",
			operation: func(v *jsonlite.Value) {
				_ = v.Len()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			val, err := jsonlite.Parse(tt.input)
			if err != nil {
				t.Fatalf("parse %q: %v", tt.input, err)
			}

			defer func() {
				if r := recover(); r == nil {
					t.Errorf("expected panic for %s, but didn't get one", tt.name)
				}
			}()

			tt.operation(val)
		})
	}
}

// Test that valid operations don't panic
func TestValueValidOperations(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		operation func(*jsonlite.Value)
	}{
		{
			name:  "StringOnString",
			input: `"hello"`,
			operation: func(v *jsonlite.Value) {
				_ = v.String()
			},
		},
		{
			name:  "StringOnNumber",
			input: "42",
			operation: func(v *jsonlite.Value) {
				_ = v.String()
			},
		},
		{
			name:  "IntOnNumber",
			input: "42",
			operation: func(v *jsonlite.Value) {
				_ = v.Int()
			},
		},
		{
			name:  "UintOnNumber",
			input: "42",
			operation: func(v *jsonlite.Value) {
				_ = v.Uint()
			},
		},
		{
			name:  "FloatOnNumber",
			input: "3.14",
			operation: func(v *jsonlite.Value) {
				_ = v.Float()
			},
		},
		{
			name:  "ArrayOnArray",
			input: "[1,2,3]",
			operation: func(v *jsonlite.Value) {
				_ = v.Array()
			},
		},
		{
			name:  "ObjectOnObject",
			input: `{"a":1}`,
			operation: func(v *jsonlite.Value) {
				_ = v.Object()
			},
		},
		{
			name:  "LookupOnObject",
			input: `{"a":1}`,
			operation: func(v *jsonlite.Value) {
				_ = v.Lookup("a")
			},
		},
		{
			name:  "NumberOnNumber",
			input: "42",
			operation: func(v *jsonlite.Value) {
				_ = v.Number()
			},
		},
		{
			name:  "LenOnString",
			input: `"hello"`,
			operation: func(v *jsonlite.Value) {
				_ = v.Len()
			},
		},
		{
			name:  "LenOnNumber",
			input: "42",
			operation: func(v *jsonlite.Value) {
				_ = v.Len()
			},
		},
		{
			name:  "LenOnArray",
			input: "[1,2,3]",
			operation: func(v *jsonlite.Value) {
				_ = v.Len()
			},
		},
		{
			name:  "LenOnObject",
			input: `{"a":1}`,
			operation: func(v *jsonlite.Value) {
				_ = v.Len()
			},
		},
		{
			name:  "StringOnArray",
			input: "[1,2,3]",
			operation: func(v *jsonlite.Value) {
				_ = v.String() // Should return JSON, not panic
			},
		},
		{
			name:  "StringOnObject",
			input: `{"a":1}`,
			operation: func(v *jsonlite.Value) {
				_ = v.String() // Should return JSON, not panic
			},
		},
		{
			name:  "StringOnNull",
			input: "null",
			operation: func(v *jsonlite.Value) {
				_ = v.String() // Should return "null", not panic
			},
		},
		{
			name:  "StringOnTrue",
			input: "true",
			operation: func(v *jsonlite.Value) {
				_ = v.String() // Should return "true", not panic
			},
		},
		{
			name:  "StringOnFalse",
			input: "false",
			operation: func(v *jsonlite.Value) {
				_ = v.String() // Should return "false", not panic
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			val, err := jsonlite.Parse(tt.input)
			if err != nil {
				t.Fatalf("parse %q: %v", tt.input, err)
			}

			defer func() {
				if r := recover(); r != nil {
					t.Errorf("unexpected panic for %s: %v", tt.name, r)
				}
			}()

			tt.operation(val)
		})
	}
}
