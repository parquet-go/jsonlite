package jsonlite_test

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"github.com/parquet-go/jsonlite"
)

// Primitive type tests

func TestAs_bool(t *testing.T) {
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
			if got := jsonlite.As[bool](val); got != tt.expected {
				t.Errorf("As[bool](%q) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}

	if got := jsonlite.As[bool](nil); got != false {
		t.Errorf("As[bool](nil) = %v, want false", got)
	}
}

func TestAs_string(t *testing.T) {
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
			if got := jsonlite.As[string](val); got != tt.expected {
				t.Errorf("As[string](%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}

	if got := jsonlite.As[string](nil); got != "" {
		t.Errorf("As[string](nil) = %q, want \"\"", got)
	}
}

func TestAs_int64(t *testing.T) {
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
			if got := jsonlite.As[int64](val); got != tt.expected {
				t.Errorf("As[int64](%q) = %d, want %d", tt.input, got, tt.expected)
			}
		})
	}

	if got := jsonlite.As[int64](nil); got != 0 {
		t.Errorf("As[int64](nil) = %d, want 0", got)
	}
}

func TestAs_uint64(t *testing.T) {
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
			if got := jsonlite.As[uint64](val); got != tt.expected {
				t.Errorf("As[uint64](%q) = %d, want %d", tt.input, got, tt.expected)
			}
		})
	}

	if got := jsonlite.As[uint64](nil); got != 0 {
		t.Errorf("As[uint64](nil) = %d, want 0", got)
	}
}

func TestAs_float64(t *testing.T) {
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
			if got := jsonlite.As[float64](val); got != tt.expected {
				t.Errorf("As[float64](%q) = %f, want %f", tt.input, got, tt.expected)
			}
		})
	}

	if got := jsonlite.As[float64](nil); got != 0 {
		t.Errorf("As[float64](nil) = %f, want 0", got)
	}
}

func TestAs_Duration(t *testing.T) {
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
			if got := jsonlite.As[time.Duration](val); got != tt.expected {
				t.Errorf("As[time.Duration](%q) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}

	if got := jsonlite.As[time.Duration](nil); got != 0 {
		t.Errorf("As[time.Duration](nil) = %v, want 0", got)
	}
}

func TestAs_Time(t *testing.T) {
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
			got := jsonlite.As[time.Time](val)
			if !got.Equal(tt.expected) {
				t.Errorf("As[time.Time](%q) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}

	if got := jsonlite.As[time.Time](nil); !got.IsZero() {
		t.Errorf("As[time.Time](nil) = %v, want zero time", got)
	}
}

func TestAs_jsonNumber(t *testing.T) {
	tests := []struct {
		input    string
		expected json.Number
	}{
		{"null", ""},
		{"true", ""},
		{"false", ""},
		{"42", "42"},
		{"-42", "-42"},
		{"3.14", "3.14"},
		{`"hello"`, ""},
		{"[]", ""},
		{"{}", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			val, err := jsonlite.Parse(tt.input)
			if err != nil {
				t.Fatalf("parse %q: %v", tt.input, err)
			}
			if got := jsonlite.As[json.Number](val); got != tt.expected {
				t.Errorf("As[json.Number](%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}

	if got := jsonlite.As[json.Number](nil); got != "" {
		t.Errorf("As[json.Number](nil) = %q, want \"\"", got)
	}
}

// Slice type tests

func TestAs_sliceBool(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []bool
	}{
		{"nil_array", "null", nil},
		{"empty_array", "[]", []bool{}},
		{"bool_array", "[true, false, true]", []bool{true, false, true}},
		{"mixed_array", `[1, 0, "hello", "", null]`, []bool{true, false, true, false, false}},
		{"non_array_string", `"not an array"`, nil},
		{"non_array_object", `{"a":1}`, nil},
		{"non_array_number", "42", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			val, err := jsonlite.Parse(tt.input)
			if err != nil {
				t.Fatalf("parse %q: %v", tt.input, err)
			}
			got := jsonlite.As[[]bool](val)
			if !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("As[[]bool](%q) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}

	if got := jsonlite.As[[]bool](nil); got != nil {
		t.Errorf("As[[]bool](nil) = %v, want nil", got)
	}
}

func TestAs_sliceInt64(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []int64
	}{
		{"nil_array", "null", nil},
		{"empty_array", "[]", []int64{}},
		{"int_array", "[1, 2, 3]", []int64{1, 2, 3}},
		{"mixed_array", `[42, "123", 3.14, true]`, []int64{42, 123, 3, 1}},
		{"non_array", `"not an array"`, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			val, err := jsonlite.Parse(tt.input)
			if err != nil {
				t.Fatalf("parse %q: %v", tt.input, err)
			}
			got := jsonlite.As[[]int64](val)
			if !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("As[[]int64](%q) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}

func TestAs_sliceUint64(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []uint64
	}{
		{"nil_array", "null", nil},
		{"empty_array", "[]", []uint64{}},
		{"uint_array", "[1, 2, 3]", []uint64{1, 2, 3}},
		{"with_negative", `[42, -1, 3.14]`, []uint64{42, 0, 3}},
		{"non_array", "42", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			val, err := jsonlite.Parse(tt.input)
			if err != nil {
				t.Fatalf("parse %q: %v", tt.input, err)
			}
			got := jsonlite.As[[]uint64](val)
			if !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("As[[]uint64](%q) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}

func TestAs_sliceFloat64(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []float64
	}{
		{"nil_array", "null", nil},
		{"empty_array", "[]", []float64{}},
		{"float_array", "[1.5, 2.7, 3.14]", []float64{1.5, 2.7, 3.14}},
		{"mixed_array", `[42, "3.14", true]`, []float64{42, 3.14, 1}},
		{"non_array", "{}", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			val, err := jsonlite.Parse(tt.input)
			if err != nil {
				t.Fatalf("parse %q: %v", tt.input, err)
			}
			got := jsonlite.As[[]float64](val)
			if !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("As[[]float64](%q) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}

func TestAs_sliceString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{"nil_array", "null", nil},
		{"empty_array", "[]", []string{}},
		{"string_array", `["a", "b", "c"]`, []string{"a", "b", "c"}},
		{"mixed_array", `["hello", 42, true, null]`, []string{"hello", "42", "true", ""}},
		{"non_array", "123", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			val, err := jsonlite.Parse(tt.input)
			if err != nil {
				t.Fatalf("parse %q: %v", tt.input, err)
			}
			got := jsonlite.As[[]string](val)
			if !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("As[[]string](%q) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}

func TestAs_sliceDuration(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []time.Duration
	}{
		{"nil_array", "null", nil},
		{"empty_array", "[]", []time.Duration{}},
		{"duration_array", `["1s", "500ms", "1h"]`, []time.Duration{time.Second, 500 * time.Millisecond, time.Hour}},
		{"number_array", "[1, 0.5, 2]", []time.Duration{time.Second, 500 * time.Millisecond, 2 * time.Second}},
		{"non_array", "true", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			val, err := jsonlite.Parse(tt.input)
			if err != nil {
				t.Fatalf("parse %q: %v", tt.input, err)
			}
			got := jsonlite.As[[]time.Duration](val)
			if !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("As[[]time.Duration](%q) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}

// Map type tests

func TestAs_mapBool(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected map[string]bool
	}{
		{"nil_object", "null", nil},
		{"empty_object", "{}", map[string]bool{}},
		{"bool_object", `{"a":true,"b":false}`, map[string]bool{"a": true, "b": false}},
		{"mixed_object", `{"x":1,"y":"","z":null}`, map[string]bool{"x": true, "y": false, "z": false}},
		{"non_object_array", "[]", nil},
		{"non_object_string", `"text"`, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			val, err := jsonlite.Parse(tt.input)
			if err != nil {
				t.Fatalf("parse %q: %v", tt.input, err)
			}
			got := jsonlite.As[map[string]bool](val)
			if !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("As[map[string]bool](%q) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}

	if got := jsonlite.As[map[string]bool](nil); got != nil {
		t.Errorf("As[map[string]bool](nil) = %v, want nil", got)
	}
}

func TestAs_mapInt64(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected map[string]int64
	}{
		{"nil_object", "null", nil},
		{"empty_object", "{}", map[string]int64{}},
		{"int_object", `{"a":1,"b":2,"c":3}`, map[string]int64{"a": 1, "b": 2, "c": 3}},
		{"mixed_object", `{"x":"42","y":3.14,"z":true}`, map[string]int64{"x": 42, "y": 3, "z": 1}},
		{"non_object", "[1,2,3]", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			val, err := jsonlite.Parse(tt.input)
			if err != nil {
				t.Fatalf("parse %q: %v", tt.input, err)
			}
			got := jsonlite.As[map[string]int64](val)
			if !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("As[map[string]int64](%q) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}

func TestAs_mapString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected map[string]string
	}{
		{"nil_object", "null", nil},
		{"empty_object", "{}", map[string]string{}},
		{"string_object", `{"a":"hello","b":"world"}`, map[string]string{"a": "hello", "b": "world"}},
		{"mixed_object", `{"x":42,"y":true,"z":null}`, map[string]string{"x": "42", "y": "true", "z": ""}},
		{"non_object", "42", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			val, err := jsonlite.Parse(tt.input)
			if err != nil {
				t.Fatalf("parse %q: %v", tt.input, err)
			}
			got := jsonlite.As[map[string]string](val)
			if !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("As[map[string]string](%q) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}

// any type tests

func TestAs_any(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected any
	}{
		{"nil", "null", nil},
		{"true", "true", true},
		{"false", "false", false},
		{"int", "42", int64(42)},
		{"negative_int", "-42", int64(-42)},
		{"uint", "18446744073709551615", uint64(18446744073709551615)},
		{"float", "3.14", 3.14},
		{"string", `"hello"`, "hello"},
		{"empty_array", "[]", []any{}},
		{"int_array", "[1,2,3]", []any{int64(1), int64(2), int64(3)}},
		{"empty_object", "{}", map[string]any{}},
		{"simple_object", `{"a":1,"b":"hello"}`, map[string]any{"a": int64(1), "b": "hello"}},
		{"nested", `{"arr":[1,2],"obj":{"x":true}}`, map[string]any{"arr": []any{int64(1), int64(2)}, "obj": map[string]any{"x": true}}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			val, err := jsonlite.Parse(tt.input)
			if err != nil {
				t.Fatalf("parse %q: %v", tt.input, err)
			}
			got := jsonlite.As[any](val)
			if !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("As[any](%q) = %#v, want %#v", tt.input, got, tt.expected)
			}
		})
	}

	if got := jsonlite.As[any](nil); got != nil {
		t.Errorf("As[any](nil) = %v, want nil", got)
	}
}

func TestAs_sliceAny(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []any
	}{
		{"nil_array", "null", nil},
		{"empty_array", "[]", []any{}},
		{"int_array", "[1,2,3]", []any{int64(1), int64(2), int64(3)}},
		{"mixed_array", `[1, "hello", true, null]`, []any{int64(1), "hello", true, nil}},
		{"nested_array", `[[1,2], {"a":3}]`, []any{[]any{int64(1), int64(2)}, map[string]any{"a": int64(3)}}},
		{"non_array_object", "{}", nil},
		{"non_array_string", `"text"`, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			val, err := jsonlite.Parse(tt.input)
			if err != nil {
				t.Fatalf("parse %q: %v", tt.input, err)
			}
			got := jsonlite.As[[]any](val)
			if !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("As[[]any](%q) = %#v, want %#v", tt.input, got, tt.expected)
			}
		})
	}

	if got := jsonlite.As[[]any](nil); got != nil {
		t.Errorf("As[[]any](nil) = %v, want nil", got)
	}
}

func TestAs_mapAny(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected map[string]any
	}{
		{"nil_object", "null", nil},
		{"empty_object", "{}", map[string]any{}},
		{"simple_object", `{"a":1,"b":"hello","c":true}`, map[string]any{"a": int64(1), "b": "hello", "c": true}},
		{"nested_object", `{"arr":[1,2],"obj":{"x":3}}`, map[string]any{"arr": []any{int64(1), int64(2)}, "obj": map[string]any{"x": int64(3)}}},
		{"non_object_array", "[]", nil},
		{"non_object_number", "42", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			val, err := jsonlite.Parse(tt.input)
			if err != nil {
				t.Fatalf("parse %q: %v", tt.input, err)
			}
			got := jsonlite.As[map[string]any](val)
			if !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("As[map[string]any](%q) = %#v, want %#v", tt.input, got, tt.expected)
			}
		})
	}

	if got := jsonlite.As[map[string]any](nil); got != nil {
		t.Errorf("As[map[string]any](nil) = %v, want nil", got)
	}
}
