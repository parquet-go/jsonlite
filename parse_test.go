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
