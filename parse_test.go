package jsonlite_test

import (
	"fmt"
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

	if val.Len() != 5 {
		t.Fatalf("expected 5 fields, got %d", val.Len())
	}

	// Check "tags" array
	tagsVal := val.Lookup("tags")
	if tagsVal == nil {
		t.Fatal("tags field not found")
	}
	if tagsVal.Kind() != jsonlite.Array {
		t.Fatalf("tags: expected Array, got %v", tagsVal.Kind())
	}
	if tagsVal.Len() != 3 {
		t.Fatalf("tags: expected length 3, got %d", tagsVal.Len())
	}

	// Check "metadata" object
	metadataVal := val.Lookup("metadata")
	if metadataVal == nil {
		t.Fatal("metadata field not found")
	}
	if metadataVal.Kind() != jsonlite.Object {
		t.Fatalf("metadata: expected Object, got %v", metadataVal.Kind())
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

func BenchmarkTokenize(b *testing.B) {
	benchmarks := []struct {
		name  string
		input string
	}{
		{
			name:  "short_string",
			input: `"hello"`,
		},
		{
			name:  "medium_string",
			input: `"hello world foo bar"`,
		},
		{
			name:  "long_string",
			input: `"The quick brown fox jumps over the lazy dog and runs away"`,
		},
		{
			name:  "string_with_escapes",
			input: `"hello \"world\" foo\nbar"`,
		},
		{
			name:  "many_short_strings",
			input: `["a","b","c","d","e","f","g","h","i","j"]`,
		},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			b.SetBytes(int64(len(bm.input)))
			for b.Loop() {
				tok := jsonlite.Tokenize(bm.input)
				for {
					_, ok := tok.Next()
					if !ok {
						break
					}
				}
			}
		})
	}
}

const cloudLoggingPayload = `{
        "logName": "projects/test-project/logs/test-log",
        "insertId": "test-insert-id",
        "timestamp": "2024-01-15T10:30:00Z",
        "receiveTimestamp": "2024-01-15T10:30:01Z",
        "severity": "INFO",
        "textPayload": "test log message",
        "resource": {
            "type": "gce_instance",
            "labels": {
                "instance_id": "1234567890",
                "zone": "us-central1-a"
            }
        },
        "labels": {
            "env": "test"
        },
        "httpRequest": {
            "requestMethod": "GET",
            "requestUrl": "https://example.com/api",
            "requestSize": 1024,
            "status": 200,
            "responseSize": 2048,
            "userAgent": "Mozilla/5.0",
            "remoteIp": "192.168.1.1",
            "serverIp": "10.0.0.1",
            "referer": "https://example.com",
            "latency": "0.5s",
            "cacheLookup": true,
            "cacheHit": false,
            "protocol": "HTTP/1.1"
        },
        "trace": "projects/test-project/traces/1234567890abcdef",
        "spanId": "abcdef1234567890",
        "traceSampled": true,
        "operation": {
            "id": "op-123",
            "producer": "test-producer",
            "first": true,
            "last": false
        },
        "sourceLocation": {
            "file": "test.go",
            "line": 42,
            "function": "TestFunction"
        },
        "split": {
            "uid": "split-123",
            "index": 1,
            "totalSplits": 3
        }
    }`

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
		{
			name:  "CloudLogging",
			input: cloudLoggingPayload,
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

func TestParseMaxDepth(t *testing.T) {
	t.Run("root object stored as unparsed when max depth is zero", func(t *testing.T) {
		// maxDepth=0 means even root object is unparsed
		val, err := jsonlite.ParseMaxDepth(`{"a":1}`, 0)
		if err != nil {
			t.Fatal(err)
		}
		if val.Kind() != jsonlite.Object {
			t.Errorf("expected Object, got %v", val.Kind())
		}
		// Accessing should trigger lazy parse
		v := val.Lookup("a")
		if v == nil {
			t.Error("expected to find 'a'")
		} else if v.Int() != 1 {
			t.Errorf("expected 1, got %d", v.Int())
		}
	})

	t.Run("first level parsed and second level unparsed when max depth is one", func(t *testing.T) {
		// maxDepth=1 means first level is parsed, second is not
		val, err := jsonlite.ParseMaxDepth(`{"a":{"b":2}}`, 1)
		if err != nil {
			t.Fatal(err)
		}
		// First level should be parsed
		a := val.Lookup("a")
		if a == nil {
			t.Fatal("expected to find 'a'")
		}
		// Second level should be unparsed (lazy)
		b := a.Lookup("b")
		if b == nil {
			t.Error("expected to find 'b'")
		} else if b.Int() != 2 {
			t.Errorf("expected 2, got %d", b.Int())
		}
	})

	t.Run("two levels parsed and third level unparsed when max depth is two", func(t *testing.T) {
		// maxDepth=2 means two levels parsed, third is not
		val, err := jsonlite.ParseMaxDepth(`{"a":{"b":{"c":3}}}`, 2)
		if err != nil {
			t.Fatal(err)
		}
		// First and second levels should be parsed
		a := val.Lookup("a")
		if a == nil {
			t.Fatal("expected to find 'a'")
		}
		b := a.Lookup("b")
		if b == nil {
			t.Fatal("expected to find 'b'")
		}
		// Third level should be unparsed (lazy)
		c := b.Lookup("c")
		if c == nil {
			t.Error("expected to find 'c'")
		} else if c.Int() != 3 {
			t.Errorf("expected 3, got %d", c.Int())
		}
	})

	t.Run("arrays do not increment the depth counter", func(t *testing.T) {
		// Arrays should not increment depth
		val, err := jsonlite.ParseMaxDepth(`{"a":[{"b":1}]}`, 1)
		if err != nil {
			t.Fatal(err)
		}
		// First level object is parsed
		a := val.Lookup("a")
		if a == nil {
			t.Fatal("expected to find 'a'")
		}
		// Array doesn't increment depth, so nested object is still depth 1
		var found bool
		for elem := range a.Array {
			b := elem.Lookup("b")
			if b != nil && b.Int() == 1 {
				found = true
				break
			}
		}
		if !found {
			t.Error("expected to find b=1 in array")
		}
	})

	t.Run("empty object at max depth can be iterated", func(t *testing.T) {
		val, err := jsonlite.ParseMaxDepth(`{"a":{}}`, 1)
		if err != nil {
			t.Fatal(err)
		}
		a := val.Lookup("a")
		if a == nil {
			t.Fatal("expected to find 'a'")
		}
		// Should be able to iterate over empty object
		count := 0
		for range a.Object {
			count++
		}
		if count != 0 {
			t.Errorf("expected 0 fields, got %d", count)
		}
	})

	t.Run("calling len on unparsed value triggers parsing", func(t *testing.T) {
		val, err := jsonlite.ParseMaxDepth(`{"a":{"x":1,"y":2,"z":3}}`, 1)
		if err != nil {
			t.Fatal(err)
		}
		a := val.Lookup("a")
		if a == nil {
			t.Fatal("expected to find 'a'")
		}
		// Len should trigger parsing
		if a.Len() != 3 {
			t.Errorf("expected length 3, got %d", a.Len())
		}
	})

	t.Run("calling JSON on unparsed value returns cached content without parsing", func(t *testing.T) {
		val, err := jsonlite.ParseMaxDepth(`{"a":{"b":1}}`, 1)
		if err != nil {
			t.Fatal(err)
		}
		a := val.Lookup("a")
		if a == nil {
			t.Fatal("expected to find 'a'")
		}
		// JSON() should return cached JSON without parsing
		json := a.JSON()
		if json != `{"b":1}` {
			t.Errorf("expected {\"b\":1}, got %s", json)
		}
	})

	t.Run("calling compact on unparsed value triggers parsing", func(t *testing.T) {
		val, err := jsonlite.ParseMaxDepth(`{"a": {"b": 1}}`, 1)
		if err != nil {
			t.Fatal(err)
		}
		a := val.Lookup("a")
		if a == nil {
			t.Fatal("expected to find 'a'")
		}
		// Compact should parse first, then compact
		compact := string(a.Compact(nil))
		if compact != `{"b":1}` {
			t.Errorf("expected {\"b\":1}, got %s", compact)
		}
	})
}

func TestLazyParsingCorrectness(t *testing.T) {
	// Complex nested structure
	input := `{
		"users": [
			{
				"name": "Alice",
				"profile": {
					"age": 30,
					"settings": {
						"theme": "dark"
					}
				}
			},
			{
				"name": "Bob",
				"profile": {
					"age": 25,
					"settings": {
						"theme": "light"
					}
				}
			}
		]
	}`

	// Parse with different depths and verify same results
	depths := []int{0, 1, 2, 3, 1000}
	var results []string

	for _, depth := range depths {
		val, err := jsonlite.ParseMaxDepth(input, depth)
		if err != nil {
			t.Fatalf("depth %d: %v", depth, err)
		}

		users := val.Lookup("users")
		if users == nil {
			t.Fatalf("depth %d: expected to find 'users'", depth)
		}

		var buf []byte
		for user := range users.Array {
			name := user.Lookup("name")
			profile := user.Lookup("profile")
			age := profile.Lookup("age")
			settings := profile.Lookup("settings")
			theme := settings.Lookup("theme")

			buf = append(buf, name.String()...)
			buf = append(buf, ':')
			buf = append(buf, []byte(fmt.Sprintf("%d", age.Int()))...)
			buf = append(buf, ':')
			buf = append(buf, theme.String()...)
			buf = append(buf, ';')
		}

		results = append(results, string(buf))
	}

	// All results should be identical
	expected := results[0]
	for i, result := range results {
		if result != expected {
			t.Errorf("depth %d produced different result:\nexpected: %s\ngot: %s", depths[i], expected, result)
		}
	}
}

func BenchmarkParseMaxDepth(b *testing.B) {
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
		{
			name:  "CloudLogging",
			input: cloudLoggingPayload,
		},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			b.ReportAllocs()
			b.SetBytes(int64(len(bm.input)))
			for b.Loop() {
				_, err := jsonlite.ParseMaxDepth(bm.input, 1)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
