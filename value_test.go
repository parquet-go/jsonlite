package jsonlite_test

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/parquet-go/jsonlite"
)

func TestAccessors(t *testing.T) {
	input := `{"num1":42,"num2":18446744073709551615,"num3":3.14,"string":"hello","bool":true,"null":null,"array":[1,2],"object":{"a":1}}`
	val, err := jsonlite.Parse(input)
	if err != nil {
		t.Fatal(err)
	}

	// Test number accessor - all numbers are float64
	if num1Val := val.Lookup("num1"); num1Val != nil {
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
	if num2Val := val.Lookup("num2"); num2Val != nil {
		if num2Val.Kind() != jsonlite.Number {
			t.Errorf("num2: expected Number, got %v", num2Val.Kind())
		}
	} else {
		t.Error("num2 field not found")
	}

	// Test float
	if num3Val := val.Lookup("num3"); num3Val != nil {
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
	if strVal := val.Lookup("string"); strVal != nil {
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
	if boolVal := val.Lookup("bool"); boolVal != nil {
		if boolVal.Kind() != jsonlite.True {
			t.Errorf("bool: expected True, got %v", boolVal.Kind())
		}
	} else {
		t.Error("bool field not found")
	}

	// Test null
	if nullVal := val.Lookup("null"); nullVal != nil {
		if nullVal.Kind() != jsonlite.Null {
			t.Errorf("null: expected Null, got %v", nullVal.Kind())
		}
	} else {
		t.Error("null field not found")
	}

	// Test array accessor
	if arrayVal := val.Lookup("array"); arrayVal != nil {
		if arrayVal.Kind() != jsonlite.Array {
			t.Errorf("array: expected Array, got %v", arrayVal.Kind())
		}
		if arrayVal.Len() != 2 {
			t.Errorf("array: expected length 2, got %d", arrayVal.Len())
		}
	} else {
		t.Error("array field not found")
	}

	// Test object accessor
	if objVal := val.Lookup("object"); objVal != nil {
		if objVal.Kind() != jsonlite.Object {
			t.Errorf("object: expected Object, got %v", objVal.Kind())
		}
		if objVal.Len() != 1 {
			t.Errorf("object: expected length 1, got %d", objVal.Len())
		}
	} else {
		t.Error("object field not found")
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

func TestLookupComprehensive(t *testing.T) {
	t.Run("EmptyObject", func(t *testing.T) {
		val, err := jsonlite.Parse(`{}`)
		if err != nil {
			t.Fatal(err)
		}
		if v := val.Lookup("any"); v != nil {
			t.Error("expected nil for lookup in empty object")
		}
	})

	t.Run("EmptyStringKey", func(t *testing.T) {
		val, err := jsonlite.Parse(`{"":42,"a":1}`)
		if err != nil {
			t.Fatal(err)
		}
		v := val.Lookup("")
		if v == nil {
			t.Fatal("expected to find empty string key")
		}
		if v.Int() != 42 {
			t.Errorf("expected 42 for empty key, got %d", v.Int())
		}
		// Ensure other keys still work
		if v := val.Lookup("a"); v == nil || v.Int() != 1 {
			t.Error("expected to find key 'a' with value 1")
		}
	})

	t.Run("PrefixKeys", func(t *testing.T) {
		val, err := jsonlite.Parse(`{"a":1,"ab":2,"abc":3,"abcd":4}`)
		if err != nil {
			t.Fatal(err)
		}
		tests := []struct {
			key   string
			value int64
		}{
			{"a", 1},
			{"ab", 2},
			{"abc", 3},
			{"abcd", 4},
		}
		for _, tt := range tests {
			v := val.Lookup(tt.key)
			if v == nil {
				t.Errorf("expected to find key %q", tt.key)
			} else if v.Int() != tt.value {
				t.Errorf("key %q: expected %d, got %d", tt.key, tt.value, v.Int())
			}
		}
		// Ensure non-existent prefix doesn't match
		if v := val.Lookup("abcde"); v != nil {
			t.Error("expected nil for non-existent key 'abcde'")
		}
	})

	t.Run("LargeObject", func(t *testing.T) {
		// Create object with 300 fields to guarantee hash collisions
		// (only 256 possible hash values)
		var sb strings.Builder
		sb.WriteString("{")
		for i := 0; i < 300; i++ {
			if i > 0 {
				sb.WriteString(",")
			}
			sb.WriteString(fmt.Sprintf(`"key%d":%d`, i, i))
		}
		sb.WriteString("}")

		val, err := jsonlite.Parse(sb.String())
		if err != nil {
			t.Fatal(err)
		}

		// Test that all keys can be found despite collisions
		for i := 0; i < 300; i++ {
			key := fmt.Sprintf("key%d", i)
			v := val.Lookup(key)
			if v == nil {
				t.Errorf("expected to find key %q", key)
			} else if v.Int() != int64(i) {
				t.Errorf("key %q: expected %d, got %d", key, i, v.Int())
			}
		}

		// Test non-existent key
		if v := val.Lookup("key300"); v != nil {
			t.Error("expected nil for non-existent key 'key300'")
		}
	})

	t.Run("HashCollisions", func(t *testing.T) {
		// Generate keys and find ones that hash to the same value
		// We'll build an object with keys that we know will collide
		keysByHash := make(map[byte][]string)

		// Generate candidate keys
		for i := 0; i < 1000; i++ {
			key := fmt.Sprintf("k%d", i)
			// We need to compute the hash the same way Lookup does
			// Since we don't have access to hashseed, we'll just create
			// a diverse set of keys and trust that some will collide
			keysByHash[byte(i%256)] = append(keysByHash[byte(i%256)], key)
		}

		// Build an object with multiple keys per hash bucket
		var sb strings.Builder
		sb.WriteString("{")
		first := true
		count := 0
		for _, keys := range keysByHash {
			if len(keys) >= 3 && count < 10 {
				// Use first 3 keys from this bucket to force collisions
				for j := 0; j < 3; j++ {
					if !first {
						sb.WriteString(",")
					}
					first = false
					sb.WriteString(fmt.Sprintf(`"%s":%d`, keys[j], j))
					count++
				}
			}
			if count >= 30 {
				break
			}
		}
		sb.WriteString("}")

		val, err := jsonlite.Parse(sb.String())
		if err != nil {
			t.Fatal(err)
		}

		// Verify all keys can still be found
		for k, v := range val.Object() {
			found := val.Lookup(k)
			if found == nil {
				t.Errorf("failed to lookup key %q that exists in object", k)
			}
			if found.JSON() != v.JSON() {
				t.Errorf("key %q: lookup returned different value", k)
			}
		}
	})

	t.Run("SimilarKeys", func(t *testing.T) {
		// Keys that differ by one character
		val, err := jsonlite.Parse(`{"user":1,"usar":2,"used":3}`)
		if err != nil {
			t.Fatal(err)
		}

		tests := []struct {
			key   string
			value int64
		}{
			{"user", 1},
			{"usar", 2},
			{"used", 3},
		}

		for _, tt := range tests {
			v := val.Lookup(tt.key)
			if v == nil {
				t.Errorf("expected to find key %q", tt.key)
			} else if v.Int() != tt.value {
				t.Errorf("key %q: expected %d, got %d", tt.key, tt.value, v.Int())
			}
		}
	})
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

func TestStringReturnsJSON(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"null", "<nil>"},
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

func TestAppend(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string // Expected output should match input exactly
	}{
		{"null", "null", "null"},
		{"true", "true", "true"},
		{"false", "false", "false"},
		{"number", "42", "42"},
		{"float", "3.14", "3.14"},
		{"string", `"hello"`, `"hello"`},
		{"empty array", "[]", "[]"},
		{"array", "[1,2,3]", "[1,2,3]"},
		{"array with spaces", "[ 1 , 2 , 3 ]", "[ 1 , 2 , 3 ]"},
		{"empty object", "{}", "{}"},
		{"object", `{"a":1}`, `{"a":1}`},
		{"object with spaces", `{ "a" : 1 }`, `{ "a" : 1 }`},
		{"nested", `{"array":[1,2,3],"object":{"nested":true}}`, `{"array":[1,2,3],"object":{"nested":true}}`},
		{"nested with whitespace", `{ "array" : [ 1 , 2 ] }`, `{ "array" : [ 1 , 2 ] }`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			val, err := jsonlite.Parse(tt.input)
			if err != nil {
				t.Fatalf("parse failed: %v", err)
			}

			result := string(val.Append(nil))
			if result != tt.expected {
				t.Errorf("Append() = %q, want %q", result, tt.expected)
			}

			// Verify the output is valid JSON
			_, err = jsonlite.Parse(result)
			if err != nil {
				t.Errorf("Append() produced invalid JSON: %v", err)
			}
		})
	}
}

func TestCompact(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string // Expected compacted output
	}{
		{"null", "null", "null"},
		{"true", "true", "true"},
		{"false", "false", "false"},
		{"number", "42", "42"},
		{"float", "3.14", "3.14"},
		{"string", `"hello"`, `"hello"`},
		{"empty array", "[]", "[]"},
		{"array", "[1,2,3]", "[1,2,3]"},
		{"array with spaces", "[ 1 , 2 , 3 ]", "[1,2,3]"},
		{"empty object", "{}", "{}"},
		{"object", `{"a":1}`, `{"a":1}`},
		{"object with spaces", `{ "a" : 1 }`, `{"a":1}`},
		{"nested", `{"array":[1,2,3],"object":{"nested":true}}`, `{"array":[1,2,3],"object":{"nested":true}}`},
		{"nested with whitespace", `{ "array" : [ 1 , 2 ] }`, `{"array":[1,2]}`},
		{"complex object", `{ "a" : 1 , "b" : "hello" , "c" : true }`, `{"a":1,"b":"hello","c":true}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			val, err := jsonlite.Parse(tt.input)
			if err != nil {
				t.Fatalf("parse failed: %v", err)
			}

			result := string(val.Compact(nil))
			if result != tt.expected {
				t.Errorf("Compact() = %q, want %q", result, tt.expected)
			}

			// Verify the output is valid JSON
			_, err = jsonlite.Parse(result)
			if err != nil {
				t.Errorf("Compact() produced invalid JSON: %v", err)
			}
		})
	}
}

func TestAppendVsCompact(t *testing.T) {
	// Test that Append preserves formatting while Compact removes it
	input := `{ "array" : [ 1 , 2 , 3 ] , "object" : { "nested" : true } }`

	val, err := jsonlite.Parse(input)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	appended := string(val.Append(nil))
	compacted := string(val.Compact(nil))

	// Append should preserve the original formatting
	if appended != input {
		t.Errorf("Append() did not preserve formatting:\ngot:  %q\nwant: %q", appended, input)
	}

	// Compact should remove whitespace
	expectedCompact := `{"array":[1,2,3],"object":{"nested":true}}`
	if compacted != expectedCompact {
		t.Errorf("Compact() did not remove whitespace:\ngot:  %q\nwant: %q", compacted, expectedCompact)
	}

	// Both should produce semantically equivalent values
	val1, _ := jsonlite.Parse(appended)
	val2, _ := jsonlite.Parse(compacted)
	if !valuesEqual(*val1, *val2) {
		t.Error("Append() and Compact() produced semantically different values")
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
		if a.Len() != b.Len() {
			return false
		}
		// Collect array elements
		var aElems, bElems []*jsonlite.Value
		for v := range a.Array() {
			aElems = append(aElems, v)
		}
		for v := range b.Array() {
			bElems = append(bElems, v)
		}
		for i := range aElems {
			if !valuesEqual(*aElems[i], *bElems[i]) {
				return false
			}
		}
		return true
	case jsonlite.Object:
		if a.Len() != b.Len() {
			return false
		}
		// Check each key in a exists in b with equal value
		for key, aVal := range a.Object() {
			bVal := b.Lookup(key)
			if bVal == nil {
				return false
			}
			if !valuesEqual(*aVal, *bVal) {
				return false
			}
		}
		return true
	}
	return false
}

// Test safety checks - these should panic when called on wrong types
func TestValueSafetyChecks(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		operation func(*jsonlite.Value)
	}{
		{
			name:  "calling int on string value panics",
			input: `"hello"`,
			operation: func(v *jsonlite.Value) {
				_ = v.Int()
			},
		},
		{
			name:  "calling int on array value panics",
			input: "[1,2,3]",
			operation: func(v *jsonlite.Value) {
				_ = v.Int()
			},
		},
		{
			name:  "calling float on string value panics",
			input: `"hello"`,
			operation: func(v *jsonlite.Value) {
				_ = v.Float()
			},
		},
		{
			name:  "calling uint on object value panics",
			input: `{"a":1}`,
			operation: func(v *jsonlite.Value) {
				_ = v.Uint()
			},
		},
		{
			name:  "calling array on string value panics",
			input: `"hello"`,
			operation: func(v *jsonlite.Value) {
				_ = v.Array()
			},
		},
		{
			name:  "calling array on number value panics",
			input: "42",
			operation: func(v *jsonlite.Value) {
				_ = v.Array()
			},
		},
		{
			name:  "calling object on string value panics",
			input: `"hello"`,
			operation: func(v *jsonlite.Value) {
				_ = v.Object()
			},
		},
		{
			name:  "calling object on array value panics",
			input: "[1,2,3]",
			operation: func(v *jsonlite.Value) {
				_ = v.Object()
			},
		},
		{
			name:  "calling lookup on array value panics",
			input: "[1,2,3]",
			operation: func(v *jsonlite.Value) {
				_ = v.Lookup("key")
			},
		},
		{
			name:  "calling lookup on string value panics",
			input: `"hello"`,
			operation: func(v *jsonlite.Value) {
				_ = v.Lookup("key")
			},
		},
		{
			name:  "calling number on string value panics",
			input: `"hello"`,
			operation: func(v *jsonlite.Value) {
				_ = v.Number()
			},
		},
		{
			name:  "calling number on array value panics",
			input: "[1,2,3]",
			operation: func(v *jsonlite.Value) {
				_ = v.Number()
			},
		},
		{
			name:  "calling len on null value panics",
			input: "null",
			operation: func(v *jsonlite.Value) {
				_ = v.Len()
			},
		},
		{
			name:  "calling len on true value panics",
			input: "true",
			operation: func(v *jsonlite.Value) {
				_ = v.Len()
			},
		},
		{
			name:  "calling len on false value panics",
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
			name:  "calling string on string value succeeds",
			input: `"hello"`,
			operation: func(v *jsonlite.Value) {
				_ = v.String()
			},
		},
		{
			name:  "calling string on number value succeeds",
			input: "42",
			operation: func(v *jsonlite.Value) {
				_ = v.String()
			},
		},
		{
			name:  "calling int on number value succeeds",
			input: "42",
			operation: func(v *jsonlite.Value) {
				_ = v.Int()
			},
		},
		{
			name:  "calling uint on number value succeeds",
			input: "42",
			operation: func(v *jsonlite.Value) {
				_ = v.Uint()
			},
		},
		{
			name:  "calling float on number value succeeds",
			input: "3.14",
			operation: func(v *jsonlite.Value) {
				_ = v.Float()
			},
		},
		{
			name:  "calling array on array value succeeds",
			input: "[1,2,3]",
			operation: func(v *jsonlite.Value) {
				_ = v.Array()
			},
		},
		{
			name:  "calling object on object value succeeds",
			input: `{"a":1}`,
			operation: func(v *jsonlite.Value) {
				_ = v.Object()
			},
		},
		{
			name:  "calling lookup on object value succeeds",
			input: `{"a":1}`,
			operation: func(v *jsonlite.Value) {
				_ = v.Lookup("a")
			},
		},
		{
			name:  "calling number on number value succeeds",
			input: "42",
			operation: func(v *jsonlite.Value) {
				_ = v.Number()
			},
		},
		{
			name:  "calling len on string value succeeds",
			input: `"hello"`,
			operation: func(v *jsonlite.Value) {
				_ = v.Len()
			},
		},
		{
			name:  "calling len on number value succeeds",
			input: "42",
			operation: func(v *jsonlite.Value) {
				_ = v.Len()
			},
		},
		{
			name:  "calling len on array value succeeds",
			input: "[1,2,3]",
			operation: func(v *jsonlite.Value) {
				_ = v.Len()
			},
		},
		{
			name:  "calling len on object value succeeds",
			input: `{"a":1}`,
			operation: func(v *jsonlite.Value) {
				_ = v.Len()
			},
		},
		{
			name:  "calling string on array value returns json",
			input: "[1,2,3]",
			operation: func(v *jsonlite.Value) {
				_ = v.String() // Should return JSON, not panic
			},
		},
		{
			name:  "calling string on object value returns json",
			input: `{"a":1}`,
			operation: func(v *jsonlite.Value) {
				_ = v.String() // Should return JSON, not panic
			},
		},
		{
			name:  "calling string on null value returns string",
			input: "null",
			operation: func(v *jsonlite.Value) {
				_ = v.String() // Should return "null", not panic
			},
		},
		{
			name:  "calling string on true value returns string",
			input: "true",
			operation: func(v *jsonlite.Value) {
				_ = v.String() // Should return "true", not panic
			},
		},
		{
			name:  "calling string on false value returns string",
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

func TestAsBool(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		// nil is tested separately
		{"null", false},
		{"true", true},
		{"false", false},
		{"0", false},
		{"0.0", false},
		{"-0", false},
		{"1", true},
		{"-1", true},
		{"3.14", true},
		{`""`, false},
		{`"hello"`, true},
		{"[]", false},
		{"[1]", true},
		{"{}", false},
		{`{"a":1}`, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			val, err := jsonlite.Parse(tt.input)
			if err != nil {
				t.Fatalf("parse %q: %v", tt.input, err)
			}
			if got := jsonlite.AsBool(val); got != tt.expected {
				t.Errorf("AsBool(%q) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}

	if got := jsonlite.AsBool(nil); got != false {
		t.Errorf("AsBool(nil) = %v, want false", got)
	}
}

func TestAsString(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"null", ""},
		{"true", "true"},
		{"false", "false"},
		{"42", "42"},
		{"3.14", "3.14"},
		{`"hello"`, "hello"},
		{"[]", "[]"},
		{"[1,2]", "[1,2]"},
		{"{}", "{}"},
		{`{"a":1}`, `{"a":1}`},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			val, err := jsonlite.Parse(tt.input)
			if err != nil {
				t.Fatalf("parse %q: %v", tt.input, err)
			}
			if got := jsonlite.AsString(val); got != tt.expected {
				t.Errorf("AsString(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}

	if got := jsonlite.AsString(nil); got != "" {
		t.Errorf("AsString(nil) = %q, want \"\"", got)
	}
}

func TestAsInt(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
	}{
		{"null", 0},
		{"true", 1},
		{"false", 0},
		{"42", 42},
		{"-42", -42},
		{"3.14", 3},
		{"-3.99", -3},
		{`"123"`, 123},
		{`"-456"`, -456},
		{`"3.14"`, 3},
		{`"hello"`, 0},
		{"[]", 0},
		{"{}", 0},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			val, err := jsonlite.Parse(tt.input)
			if err != nil {
				t.Fatalf("parse %q: %v", tt.input, err)
			}
			if got := jsonlite.AsInt(val); got != tt.expected {
				t.Errorf("AsInt(%q) = %d, want %d", tt.input, got, tt.expected)
			}
		})
	}

	if got := jsonlite.AsInt(nil); got != 0 {
		t.Errorf("AsInt(nil) = %d, want 0", got)
	}
}

func TestAsUint(t *testing.T) {
	tests := []struct {
		input    string
		expected uint64
	}{
		{"null", 0},
		{"true", 1},
		{"false", 0},
		{"42", 42},
		{"-42", 0},
		{"3.14", 3},
		{"-3.14", 0},
		{`"123"`, 123},
		{`"-456"`, 0},
		{`"hello"`, 0},
		{"[]", 0},
		{"{}", 0},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			val, err := jsonlite.Parse(tt.input)
			if err != nil {
				t.Fatalf("parse %q: %v", tt.input, err)
			}
			if got := jsonlite.AsUint(val); got != tt.expected {
				t.Errorf("AsUint(%q) = %d, want %d", tt.input, got, tt.expected)
			}
		})
	}

	if got := jsonlite.AsUint(nil); got != 0 {
		t.Errorf("AsUint(nil) = %d, want 0", got)
	}
}

func TestAsFloat(t *testing.T) {
	tests := []struct {
		input    string
		expected float64
	}{
		{"null", 0},
		{"true", 1},
		{"false", 0},
		{"42", 42},
		{"-42", -42},
		{"3.14", 3.14},
		{`"3.14"`, 3.14},
		{`"hello"`, 0},
		{"[]", 0},
		{"{}", 0},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			val, err := jsonlite.Parse(tt.input)
			if err != nil {
				t.Fatalf("parse %q: %v", tt.input, err)
			}
			if got := jsonlite.AsFloat(val); got != tt.expected {
				t.Errorf("AsFloat(%q) = %f, want %f", tt.input, got, tt.expected)
			}
		})
	}

	if got := jsonlite.AsFloat(nil); got != 0 {
		t.Errorf("AsFloat(nil) = %f, want 0", got)
	}
}

func TestAsDuration(t *testing.T) {
	tests := []struct {
		input    string
		expected time.Duration
	}{
		{"null", 0},
		{"true", time.Second},
		{"false", 0},
		{"1", time.Second},
		{"1.5", 1500 * time.Millisecond},
		{"0.001", time.Millisecond},
		{`"1s"`, time.Second},
		{`"500ms"`, 500 * time.Millisecond},
		{`"1h30m"`, 90 * time.Minute},
		{`"invalid"`, 0},
		{"[]", 0},
		{"{}", 0},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			val, err := jsonlite.Parse(tt.input)
			if err != nil {
				t.Fatalf("parse %q: %v", tt.input, err)
			}
			if got := jsonlite.AsDuration(val); got != tt.expected {
				t.Errorf("AsDuration(%q) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}

	if got := jsonlite.AsDuration(nil); got != 0 {
		t.Errorf("AsDuration(nil) = %v, want 0", got)
	}
}

func TestAsTime(t *testing.T) {
	refTime := time.Date(2024, 6, 15, 12, 30, 45, 0, time.UTC)

	tests := []struct {
		name     string
		input    string
		expected time.Time
	}{
		{"null", "null", time.Time{}},
		{"true", "true", time.Time{}},
		{"false", "false", time.Time{}},
		{"unix_epoch", "0", time.Unix(0, 0).UTC()},
		{"unix_timestamp", "1718454645", refTime},
		{"unix_with_fraction", "1718454645.5", time.Date(2024, 6, 15, 12, 30, 45, 500000000, time.UTC)},
		{"rfc3339", `"2024-06-15T12:30:45Z"`, refTime},
		{"invalid_string", `"not a time"`, time.Time{}},
		{"array", "[]", time.Time{}},
		{"object", "{}", time.Time{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			val, err := jsonlite.Parse(tt.input)
			if err != nil {
				t.Fatalf("parse %q: %v", tt.input, err)
			}
			got := jsonlite.AsTime(val)
			if !got.Equal(tt.expected) {
				t.Errorf("AsTime(%q) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}

	if got := jsonlite.AsTime(nil); !got.IsZero() {
		t.Errorf("AsTime(nil) = %v, want zero time", got)
	}
}

func BenchmarkLookup(b *testing.B) {
	sizes := []int{1, 10, 25, 100}

	for _, size := range sizes {
		// Generate object with 'size' fields
		// Use keys that sort alphabetically: field_000, field_001, etc.
		fields := make([]string, size)
		for i := 0; i < size; i++ {
			fields[i] = fmt.Sprintf(`"field_%03d":%d`, i, i)
		}
		json := "{" + strings.Join(fields, ",") + "}"

		val, err := jsonlite.Parse(json)
		if err != nil {
			b.Fatalf("parse failed: %v", err)
		}

		// Benchmark looking up first field (best case - early in sorted list)
		b.Run(fmt.Sprintf("First_%dfields", size), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				result := val.Lookup("field_000")
				if result == nil {
					b.Fatal("expected to find field_000")
				}
			}
		})

		// Benchmark looking up middle field
		if size > 1 {
			middleKey := fmt.Sprintf("field_%03d", size/2)
			b.Run(fmt.Sprintf("Middle_%dfields", size), func(b *testing.B) {
				b.ReportAllocs()
				for i := 0; i < b.N; i++ {
					result := val.Lookup(middleKey)
					if result == nil {
						b.Fatalf("expected to find %s", middleKey)
					}
				}
			})
		}

		// Benchmark looking up last field (worst case - late in sorted list)
		lastKey := fmt.Sprintf("field_%03d", size-1)
		b.Run(fmt.Sprintf("Last_%dfields", size), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				result := val.Lookup(lastKey)
				if result == nil {
					b.Fatalf("expected to find %s", lastKey)
				}
			}
		})

		// Benchmark looking up non-existent field
		b.Run(fmt.Sprintf("NotFound_%dfields", size), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				result := val.Lookup("nonexistent")
				if result != nil {
					b.Fatal("expected nil for nonexistent field")
				}
			}
		})
	}
}

func BenchmarkAppendVsCompact(b *testing.B) {
	tests := []struct {
		name string
		json string
	}{
		{
			name: "SimpleObject",
			json: `{"a":1,"b":"hello","c":true}`,
		},
		{
			name: "SimpleArray",
			json: `[1,2,3,4,5,6,7,8,9,10]`,
		},
		{
			name: "ObjectWithWhitespace",
			json: `{ "a" : 1 , "b" : "hello" , "c" : true , "d" : null }`,
		},
		{
			name: "NestedObject",
			json: `{"user":{"name":"John","age":30,"address":{"street":"Main St","city":"NYC"}}}`,
		},
		{
			name: "NestedArray",
			json: `[[1,2,3],[4,5,6],[7,8,9]]`,
		},
		{
			name: "LargeObject",
			json: func() string {
				fields := make([]string, 100)
				for i := 0; i < 100; i++ {
					fields[i] = fmt.Sprintf(`"field_%d":%d`, i, i)
				}
				return "{" + strings.Join(fields, ",") + "}"
			}(),
		},
		{
			name: "LargeArray",
			json: func() string {
				elements := make([]string, 100)
				for i := 0; i < 100; i++ {
					elements[i] = fmt.Sprintf("%d", i)
				}
				return "[" + strings.Join(elements, ",") + "]"
			}(),
		},
		{
			name: "DeeplyNested",
			json: `{"a":{"b":{"c":{"d":{"e":{"f":{"g":{"h":{"i":{"j":"deep"}}}}}}}}}}`,
		},
	}

	for _, tt := range tests {
		val, err := jsonlite.Parse(tt.json)
		if err != nil {
			b.Fatalf("%s: parse failed: %v", tt.name, err)
		}

		// Benchmark Append (uses cached JSON - O(1))
		b.Run(tt.name+"/Append", func(b *testing.B) {
			b.SetBytes(int64(len(tt.json)))
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				result := val.Append(nil)
				if len(result) == 0 {
					b.Fatal("Append returned empty result")
				}
			}
		})

		// Benchmark Compact (recursive reconstruction - O(n))
		b.Run(tt.name+"/Compact", func(b *testing.B) {
			// For compact, measure output size not input size
			compacted := val.Compact(nil)
			b.SetBytes(int64(len(compacted)))
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				result := val.Compact(nil)
				if len(result) == 0 {
					b.Fatal("Compact returned empty result")
				}
			}
		})
	}
}
