package jsonlite_test

import (
	"encoding/json"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/parquet-go/jsonlite"
)

func FuzzParse(f *testing.F) {
	// Add seed corpus with various JSON patterns
	seeds := []string{
		// Basic values
		`null`,
		`true`,
		`false`,
		`0`,
		`1`,
		`-1`,
		`123`,
		`-456`,
		`3.14`,
		`-2.718`,
		`1e10`,
		`1.5e-10`,
		`""`,
		`"hello"`,
		`"hello world"`,
		`"with\nnewline"`,
		`"with\ttab"`,
		`"with\"quote"`,
		`"unicode: \u0048\u0065\u006c\u006c\u006f"`,
		`"\ud83d\ude00"`,

		// Arrays
		`[]`,
		`[1]`,
		`[1,2,3]`,
		`["a","b","c"]`,
		`[1,"two",true,null]`,
		`[[1,2],[3,4]]`,
		`[[[1]]]`,

		// Objects
		`{}`,
		`{"a":1}`,
		`{"a":1,"b":2}`,
		`{"name":"test","age":42}`,
		`{"nested":{"a":1}}`,
		`{"array":[1,2,3]}`,

		// Complex nested structures
		`{"users":[{"name":"Alice"},{"name":"Bob"}]}`,
		`[{"a":1},{"b":2}]`,
		`{"a":{"b":{"c":{"d":1}}}}`,
		`[[[[1]]]]`,

		// Edge cases
		`{"":1}`,
		`{"a":""}`,
		`[null,null,null]`,
		`{"a":null,"b":null}`,

		// Invalid JSON (should error gracefully)
		``,
		`{`,
		`}`,
		`[`,
		`]`,
		`{]`,
		`[}`,
		`{"a"}`,
		`{"a":}`,
		`{:1}`,
		`[,]`,
		`[1,]`,
		`[,1]`,
		`{"a":1,}`,
		`{,"a":1}`,
		`"unclosed`,
		`'single quotes'`,
		`tru`,
		`fals`,
		`nul`,
		`NaN`,
		`Infinity`,
		`undefined`,
	}

	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, data string) {
		// Test that Parse doesn't panic
		val, err := jsonlite.Parse(data)

		// If we successfully parsed, verify basic properties
		if err == nil && val != nil {
			// Verify Kind is valid
			switch val.Kind() {
			case jsonlite.Null, jsonlite.True, jsonlite.False,
				jsonlite.Number, jsonlite.String, jsonlite.Object, jsonlite.Array:
				// Valid kind
			default:
				t.Errorf("invalid Kind: %v", val.Kind())
			}

			// Verify we can call String() without panic
			_ = val.String()

			// Verify we can call Append without panic
			_ = val.Append(nil)
		}
	})
}

func FuzzParseMatchesStdlib(f *testing.F) {
	// Add seed corpus with valid JSON only
	seeds := []string{
		`null`,
		`true`,
		`false`,
		`0`,
		`123`,
		`3.14`,
		`""`,
		`"hello"`,
		`[]`,
		`[1,2,3]`,
		`{}`,
		`{"a":1}`,
		`{"a":1,"b":"hello","c":true,"d":null}`,
		`[1,"two",true,null]`,
		`{"nested":{"a":[1,2,3]}}`,
	}

	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, data string) {
		// Parse with standard library
		var stdVal any
		stdErr := json.Unmarshal([]byte(data), &stdVal)

		// Parse with jsonlite
		val, err := jsonlite.Parse(data)

		// Both should agree on validity
		if (stdErr == nil) != (err == nil) {
			// Allow jsonlite to be stricter (reject more)
			// but not more lenient (accept invalid JSON)
			if stdErr != nil && err == nil {
				// Exception: jsonlite accepts valid JSON numbers that overflow float64
				// (stdlib rejects these with "cannot unmarshal number X into Go value of type float64")
				if strings.Contains(stdErr.Error(), "cannot unmarshal number") {
					return
				}
				t.Errorf("jsonlite accepted invalid JSON that stdlib rejected: %q", data)
			}
		}

		// If both succeeded, verify the parsed structure matches
		if stdErr == nil && err == nil && val != nil {
			compareValues(t, data, stdVal, val)
		}
	})
}

func compareValues(t *testing.T, input string, std any, val *jsonlite.Value) {
	t.Helper()

	switch v := std.(type) {
	case nil:
		if val.Kind() != jsonlite.Null {
			t.Errorf("input %q: expected Null, got %v", input, val.Kind())
		}
	case bool:
		if v {
			if val.Kind() != jsonlite.True {
				t.Errorf("input %q: expected True, got %v", input, val.Kind())
			}
		} else {
			if val.Kind() != jsonlite.False {
				t.Errorf("input %q: expected False, got %v", input, val.Kind())
			}
		}
	case float64:
		if val.Kind() != jsonlite.Number {
			t.Errorf("input %q: expected Number, got %v", input, val.Kind())
		}
	case string:
		if val.Kind() != jsonlite.String {
			t.Errorf("input %q: expected String, got %v", input, val.Kind())
		} else if val.String() != v {
			// Skip comparison for invalid UTF-8 - stdlib normalizes to replacement char,
			// jsonlite preserves raw bytes (intentional difference)
			if utf8.ValidString(input) {
				t.Errorf("input %q: string mismatch: got %q, want %q", input, val.String(), v)
			}
		}
	case []any:
		if val.Kind() != jsonlite.Array {
			t.Errorf("input %q: expected Array, got %v", input, val.Kind())
		} else {
			arr := val.Array()
			if len(arr) != len(v) {
				t.Errorf("input %q: array length mismatch: got %d, want %d", input, len(arr), len(v))
			} else {
				for i := range v {
					compareValues(t, input, v[i], &arr[i])
				}
			}
		}
	case map[string]any:
		if val.Kind() != jsonlite.Object {
			t.Errorf("input %q: expected Object, got %v", input, val.Kind())
		} else {
			fields := val.Object()
			// Skip detailed comparison if input has invalid UTF-8
			// stdlib normalizes keys, jsonlite preserves raw bytes
			if !utf8.ValidString(input) {
				return
			}
			// Note: jsonlite keeps duplicate keys, stdlib uses last-wins
			// Skip comparison if there are duplicate keys (different behavior)
			if len(fields) != len(v) {
				return
			}
			for _, f := range fields {
				stdField, ok := v[f.Key]
				if !ok {
					t.Errorf("input %q: unexpected field %q", input, f.Key)
				} else {
					compareValues(t, input, stdField, &f.Val)
				}
			}
		}
	}
}

func FuzzIterator(f *testing.F) {
	// Add seed corpus
	seeds := []string{
		`null`,
		`true`,
		`false`,
		`42`,
		`"hello"`,
		`[]`,
		`[1,2,3]`,
		`{}`,
		`{"a":1}`,
		`{"a":1,"b":2,"c":3}`,
		`[{"a":1},{"b":2}]`,
		`{"users":[{"name":"Alice"},{"name":"Bob"}]}`,
	}

	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, data string) {
		// First check if it's valid JSON using stdlib
		var stdVal any
		if json.Unmarshal([]byte(data), &stdVal) != nil {
			return // Skip invalid JSON for iterator tests
		}

		// Test basic iteration doesn't panic
		iter := jsonlite.Iterate(data)
		count := 0
		for iter.Next() {
			count++
			_ = iter.Kind()
			_ = iter.Key()
			_ = iter.Depth()

			// Limit iterations to prevent infinite loops on malformed input
			if count > 10000 {
				t.Fatalf("too many iterations")
			}
		}

		// Check for errors
		if iter.Err() != nil {
			t.Errorf("iterator error on valid JSON %q: %v", data, iter.Err())
		}
	})
}

func FuzzIteratorValue(f *testing.F) {
	// Add seed corpus
	seeds := []string{
		`null`,
		`true`,
		`false`,
		`42`,
		`"hello"`,
		`[]`,
		`[1,2,3]`,
		`{}`,
		`{"a":1}`,
		`[{"a":1},{"b":2}]`,
	}

	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, data string) {
		// First check if it's valid JSON using stdlib
		var stdVal any
		if json.Unmarshal([]byte(data), &stdVal) != nil {
			return // Skip invalid JSON
		}

		// Test that Iterator.Value() matches Parse()
		iter := jsonlite.Iterate(data)
		if !iter.Next() {
			// Empty or invalid - check Parse agrees
			parsed, err := jsonlite.Parse(data)
			if err == nil && parsed != nil {
				t.Errorf("Parse succeeded but Iterator.Next() returned false for %q", data)
			}
			return
		}

		iterVal, iterErr := iter.Value()
		parsedVal, parsedErr := jsonlite.Parse(data)

		// Both should agree on success/failure
		if (iterErr == nil) != (parsedErr == nil) {
			t.Errorf("Iterator.Value() and Parse() disagree on %q: iter=%v, parse=%v", data, iterErr, parsedErr)
			return
		}

		if iterErr == nil && parsedErr == nil {
			// Compare the values
			iterStr := iterVal.String()
			parsedStr := parsedVal.String()
			if iterStr != parsedStr {
				t.Errorf("Iterator.Value() and Parse() produced different results for %q:\niter:  %s\nparse: %s", data, iterStr, parsedStr)
			}
		}
	})
}

func FuzzIteratorObjectArray(f *testing.F) {
	// Add seed corpus with objects and arrays
	seeds := []string{
		`{}`,
		`{"a":1}`,
		`{"a":1,"b":2,"c":3}`,
		`[]`,
		`[1]`,
		`[1,2,3]`,
		`{"nested":{"a":1}}`,
		`[{"a":1},{"b":2}]`,
		`{"arr":[1,2,3]}`,
		`{"users":[{"name":"Alice","age":30},{"name":"Bob","age":25}]}`,
	}

	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, data string) {
		// First check if it's valid JSON using stdlib
		var stdVal any
		if json.Unmarshal([]byte(data), &stdVal) != nil {
			return // Skip invalid JSON
		}

		iter := jsonlite.Iterate(data)
		if !iter.Next() {
			return
		}

		// Test Object() and Array() sequences don't panic
		// Use a recursive helper to properly consume nested values
		var consumeValue func() error
		consumeValue = func() error {
			switch iter.Kind() {
			case jsonlite.Object:
				for key, err := range iter.Object() {
					if err != nil {
						return err
					}
					_ = key
					if err := consumeValue(); err != nil {
						return err
					}
				}
			case jsonlite.Array:
				for idx, err := range iter.Array() {
					if err != nil {
						return err
					}
					_ = idx
					if err := consumeValue(); err != nil {
						return err
					}
				}
			default:
				// Scalar value - just consume it
				_, err := iter.Value()
				return err
			}
			return nil
		}

		if err := consumeValue(); err != nil {
			t.Errorf("error consuming valid JSON %q: %v", data, err)
		}
	})
}

func FuzzTokenizer(f *testing.F) {
	// Add seed corpus
	seeds := []string{
		``,
		`null`,
		`true`,
		`false`,
		`123`,
		`"hello"`,
		`[]`,
		`{}`,
		`[1,2,3]`,
		`{"a":1}`,
		`  {  "a"  :  1  }  `,
		"\t\n\r ",
	}

	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, data string) {
		// Test that tokenizer doesn't panic
		tok := jsonlite.Tokenize(data)
		count := 0
		for {
			token, ok := tok.Next()
			if !ok {
				break
			}
			_ = token
			count++
			if count > 100000 {
				t.Fatalf("too many tokens")
			}
		}
	})
}
