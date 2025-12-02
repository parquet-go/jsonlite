package jsonlite_test

import (
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
