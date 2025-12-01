package jsonlite_test

import (
	"encoding/json"
	"testing"

	"github.com/parquet-go/jsonlite"
)

func TestValid(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		// Valid JSON values
		{"null", `null`, true},
		{"true", `true`, true},
		{"false", `false`, true},
		{"zero", `0`, true},
		{"positive int", `42`, true},
		{"negative int", `-42`, true},
		{"float", `3.14`, true},
		{"negative float", `-3.14`, true},
		{"exponent", `1e10`, true},
		{"negative exponent", `1.5e-10`, true},
		{"positive exponent", `1E+5`, true},
		{"empty string", `""`, true},
		{"simple string", `"hello"`, true},
		{"string with spaces", `"hello world"`, true},
		{"string with newline escape", `"hello\nworld"`, true},
		{"string with tab escape", `"hello\tworld"`, true},
		{"string with quote escape", `"hello\"world"`, true},
		{"string with backslash escape", `"hello\\world"`, true},
		{"string with unicode escape", `"\u0048\u0065\u006c\u006c\u006f"`, true},
		{"string with emoji unicode", `"\ud83d\ude00"`, true},
		{"empty array", `[]`, true},
		{"array with one element", `[1]`, true},
		{"array with multiple elements", `[1,2,3]`, true},
		{"array with mixed types", `[1,"two",true,null]`, true},
		{"nested arrays", `[[1,2],[3,4]]`, true},
		{"deeply nested array", `[[[1]]]`, true},
		{"empty object", `{}`, true},
		{"object with one field", `{"a":1}`, true},
		{"object with multiple fields", `{"a":1,"b":2}`, true},
		{"nested object", `{"a":{"b":1}}`, true},
		{"object with array", `{"arr":[1,2,3]}`, true},
		{"complex nested", `{"users":[{"name":"Alice"},{"name":"Bob"}]}`, true},
		{"whitespace around", `  { "a" : 1 }  `, true},
		{"tabs and newlines", "{\n\t\"a\":\t1\n}", true},

		// Invalid JSON
		{"empty", ``, false},
		{"just whitespace", `   `, false},
		{"unclosed object", `{`, false},
		{"unclosed array", `[`, false},
		{"extra closing brace", `{}}`, false},
		{"extra closing bracket", `[]]`, false},
		{"mismatched braces", `{]`, false},
		{"mismatched brackets", `[}`, false},
		{"missing value in object", `{"a"}`, false},
		{"missing colon", `{"a"1}`, false},
		{"missing key value", `{:1}`, false},
		{"trailing comma in array", `[1,]`, false},
		{"leading comma in array", `[,1]`, false},
		{"trailing comma in object", `{"a":1,}`, false},
		{"leading comma in object", `{,"a":1}`, false},
		{"unclosed string", `"unclosed`, false},
		{"single quotes", `'hello'`, false},
		{"incomplete true", `tru`, false},
		{"incomplete false", `fals`, false},
		{"incomplete null", `nul`, false},
		{"NaN", `NaN`, false},
		{"Infinity", `Infinity`, false},
		{"undefined", `undefined`, false},
		{"leading zeros", `01`, false},
		{"trailing decimal", `1.`, false},
		{"leading decimal", `.1`, false},
		{"multiple values", `1 2`, false},
		{"object then extra", `{} extra`, false},
		{"invalid escape", `"\x00"`, false},
		{"unescaped control char", "\"\x00\"", false},
		{"unescaped newline", "\"\n\"", false},
		{"incomplete unicode escape", `"\u00"`, false},
		{"invalid unicode escape", `"\uXXXX"`, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := jsonlite.Valid(tt.input)
			if got != tt.want {
				t.Errorf("Valid(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestValidMatchesStdlib(t *testing.T) {
	// Test that Valid matches encoding/json.Valid for various inputs
	tests := []string{
		// Valid JSON
		`null`, `true`, `false`,
		`0`, `1`, `-1`, `123`, `3.14`, `-3.14`, `1e10`, `1.5e-10`,
		`""`, `"hello"`, `"hello world"`,
		`[]`, `[1]`, `[1,2,3]`,
		`{}`, `{"a":1}`, `{"a":1,"b":2}`,
		`{"nested":{"a":[1,2,3]}}`,
		// Invalid JSON
		``, `{`, `}`, `[`, `]`,
		`{]`, `[}`,
		`{"a"}`, `{"a":}`, `{:1}`,
		`[,]`, `[1,]`, `[,1]`,
		`{"a":1,}`, `{,"a":1}`,
		`"unclosed`,
		`tru`, `fals`, `nul`,
		`NaN`, `Infinity`, `undefined`,
		`01`, `1.`, `.1`,
	}

	for _, input := range tests {
		stdValid := json.Valid([]byte(input))
		ourValid := jsonlite.Valid(input)
		if stdValid != ourValid {
			t.Errorf("Valid(%q): stdlib=%v, jsonlite=%v", input, stdValid, ourValid)
		}
	}
}

func FuzzValid(f *testing.F) {
	// Add seed corpus
	seeds := []string{
		`null`, `true`, `false`,
		`0`, `123`, `3.14`, `1e10`,
		`""`, `"hello"`, `"with\nnewline"`,
		`[]`, `[1,2,3]`,
		`{}`, `{"a":1}`,
		``, `{`, `}`, `[`, `]`,
		`[1,]`, `{"a":1,}`,
	}

	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, data string) {
		// Valid should match encoding/json.Valid
		stdValid := json.Valid([]byte(data))
		ourValid := jsonlite.Valid(data)

		if stdValid != ourValid {
			t.Errorf("Valid(%q): stdlib=%v, jsonlite=%v", data, stdValid, ourValid)
		}
	})
}

var benchmarkInputs = []struct {
	name  string
	input string
}{
	{"null", `null`},
	{"bool", `true`},
	{"number", `12345.6789`},
	{"string_short", `"hello"`},
	{"string_long", `"The quick brown fox jumps over the lazy dog and runs away into the forest"`},
	{"string_escapes", `"hello \"world\" with\nnewlines\tand\ttabs"`},
	{"array_small", `[1,2,3,4,5]`},
	{"array_medium", `[1,2,3,4,5,6,7,8,9,10,11,12,13,14,15,16,17,18,19,20]`},
	{"object_small", `{"a":1,"b":2,"c":3}`},
	{"object_medium", `{"name":"test","age":42,"active":true,"score":98.6,"tags":["a","b","c"]}`},
	{"nested", `{"users":[{"id":1,"name":"Alice","email":"alice@example.com"},{"id":2,"name":"Bob","email":"bob@example.com"}],"meta":{"total":2,"page":1}}`},
}

func BenchmarkValid(b *testing.B) {
	for _, bm := range benchmarkInputs {
		b.Run(bm.name, func(b *testing.B) {
			b.SetBytes(int64(len(bm.input)))
			b.ReportAllocs()
			for b.Loop() {
				jsonlite.Valid(bm.input)
			}
		})
	}
}

func BenchmarkValidStdlib(b *testing.B) {
	for _, bm := range benchmarkInputs {
		b.Run(bm.name, func(b *testing.B) {
			data := []byte(bm.input)
			b.SetBytes(int64(len(bm.input)))
			b.ReportAllocs()
			for b.Loop() {
				json.Valid(data)
			}
		})
	}
}
