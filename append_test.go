package jsonlite_test

import (
	"slices"
	"testing"

	"github.com/parquet-go/jsonlite"
)

func TestAppendArrayEmpty(t *testing.T) {
	seq := func(yield func(int64) bool) {}
	result := jsonlite.AppendArray(nil, seq, jsonlite.AppendInt)
	if string(result) != "[]" {
		t.Errorf("expected [], got %s", result)
	}
}

func TestAppendArraySingle(t *testing.T) {
	seq := slices.Values([]int64{42})
	result := jsonlite.AppendArray(nil, seq, jsonlite.AppendInt)
	if string(result) != "[42]" {
		t.Errorf("expected [42], got %s", result)
	}
}

func TestAppendArrayMultiple(t *testing.T) {
	seq := slices.Values([]int64{1, 2, 3})
	result := jsonlite.AppendArray(nil, seq, jsonlite.AppendInt)
	if string(result) != "[1,2,3]" {
		t.Errorf("expected [1,2,3], got %s", result)
	}
}

func TestAppendArrayStrings(t *testing.T) {
	seq := slices.Values([]string{"hello", "world"})
	result := jsonlite.AppendArray(nil, seq, jsonlite.AppendQuote)
	if string(result) != `["hello","world"]` {
		t.Errorf(`expected ["hello","world"], got %s`, result)
	}
}

func TestAppendArrayStringsWithEscaping(t *testing.T) {
	seq := slices.Values([]string{"hello\nworld", "quote\"here"})
	result := jsonlite.AppendArray(nil, seq, jsonlite.AppendQuote)
	expected := `["hello\nworld","quote\"here"]`
	if string(result) != expected {
		t.Errorf("expected %s, got %s", expected, result)
	}
}

func TestAppendArrayFloats(t *testing.T) {
	seq := slices.Values([]float64{1.5, 2.5, 3.5})
	result := jsonlite.AppendArray(nil, seq, jsonlite.AppendFloat)
	if string(result) != "[1.5,2.5,3.5]" {
		t.Errorf("expected [1.5,2.5,3.5], got %s", result)
	}
}

func TestAppendArrayBooleans(t *testing.T) {
	seq := slices.Values([]bool{true, false, true})
	result := jsonlite.AppendArray(nil, seq, jsonlite.AppendBool)
	if string(result) != "[true,false,true]" {
		t.Errorf("expected [true,false,true], got %s", result)
	}
}

func TestAppendArrayNested(t *testing.T) {
	appendInner := func(b []byte, nums []int64) []byte {
		return jsonlite.AppendArray(b, slices.Values(nums), jsonlite.AppendInt)
	}
	outer := slices.Values([][]int64{{1, 2}, {3, 4}})
	result := jsonlite.AppendArray(nil, outer, appendInner)
	if string(result) != "[[1,2],[3,4]]" {
		t.Errorf("expected [[1,2],[3,4]], got %s", result)
	}
}

func TestAppendArrayExistingBuffer(t *testing.T) {
	buf := []byte("prefix:")
	seq := slices.Values([]int64{1, 2, 3})
	result := jsonlite.AppendArray(buf, seq, jsonlite.AppendInt)
	if string(result) != "prefix:[1,2,3]" {
		t.Errorf("expected prefix:[1,2,3], got %s", result)
	}
}

func TestAppendObjectEmpty(t *testing.T) {
	seq := func(yield func(string, int64) bool) {}
	result := jsonlite.AppendObject(nil, seq, jsonlite.AppendInt)
	if string(result) != "{}" {
		t.Errorf("expected {}, got %s", result)
	}
}

func TestAppendObjectSingle(t *testing.T) {
	seq := func(yield func(string, int64) bool) {
		yield("key", 42)
	}
	result := jsonlite.AppendObject(nil, seq, jsonlite.AppendInt)
	if string(result) != `{"key":42}` {
		t.Errorf(`expected {"key":42}, got %s`, result)
	}
}

func TestAppendObjectMultiple(t *testing.T) {
	seq := func(yield func(string, int64) bool) {
		yield("a", 1)
		yield("b", 2)
	}
	result := jsonlite.AppendObject(nil, seq, jsonlite.AppendInt)
	if string(result) != `{"a":1,"b":2}` {
		t.Errorf(`expected {"a":1,"b":2}, got %s`, result)
	}
}

func TestAppendObjectStringValues(t *testing.T) {
	seq := func(yield func(string, string) bool) {
		yield("name", "Alice")
		yield("city", "NYC")
	}
	result := jsonlite.AppendObject(nil, seq, jsonlite.AppendQuote)
	if string(result) != `{"name":"Alice","city":"NYC"}` {
		t.Errorf(`expected {"name":"Alice","city":"NYC"}, got %s`, result)
	}
}

func TestAppendObjectKeysWithEscaping(t *testing.T) {
	seq := func(yield func(string, int64) bool) {
		yield("hello\nworld", 1)
		yield("quote\"key", 2)
	}
	result := jsonlite.AppendObject(nil, seq, jsonlite.AppendInt)
	// Verify it's valid JSON
	_, err := jsonlite.Parse(string(result))
	if err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
}

func TestAppendObjectNested(t *testing.T) {
	appendInner := func(b []byte, v int64) []byte {
		return jsonlite.AppendObject(b, func(yield func(string, int64) bool) {
			yield("inner", v)
		}, jsonlite.AppendInt)
	}
	result := jsonlite.AppendObject(nil, func(yield func(string, int64) bool) {
		yield("outer", 42)
	}, appendInner)
	if string(result) != `{"outer":{"inner":42}}` {
		t.Errorf(`expected {"outer":{"inner":42}}, got %s`, result)
	}
}

func TestAppendObjectExistingBuffer(t *testing.T) {
	buf := []byte("data=")
	seq := func(yield func(string, int64) bool) {
		yield("x", 1)
	}
	result := jsonlite.AppendObject(buf, seq, jsonlite.AppendInt)
	if string(result) != `data={"x":1}` {
		t.Errorf(`expected data={"x":1}, got %s`, result)
	}
}

func TestAppendArrayRoundTrip(t *testing.T) {
	original := []int64{1, 2, 3, 4, 5}
	result := jsonlite.AppendArray(nil, slices.Values(original), jsonlite.AppendInt)

	val, err := jsonlite.Parse(string(result))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if val.Kind() != jsonlite.Array {
		t.Fatalf("expected array, got %v", val.Kind())
	}

	var parsed []int64
	for elem := range val.Array {
		parsed = append(parsed, elem.Int())
	}
	if !slices.Equal(original, parsed) {
		t.Errorf("round-trip failed: %v != %v", original, parsed)
	}
}

func TestAppendObjectRoundTrip(t *testing.T) {
	result := jsonlite.AppendObject(nil, func(yield func(string, string) bool) {
		yield("hello", "world")
		yield("foo", "bar")
	}, jsonlite.AppendQuote)

	val, err := jsonlite.Parse(string(result))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if val.Kind() != jsonlite.Object {
		t.Fatalf("expected object, got %v", val.Kind())
	}

	if val.Lookup("hello").String() != "world" {
		t.Errorf("expected world, got %s", val.Lookup("hello").String())
	}
	if val.Lookup("foo").String() != "bar" {
		t.Errorf("expected bar, got %s", val.Lookup("foo").String())
	}
}

func TestAppendNull(t *testing.T) {
	result := string(jsonlite.AppendNull(nil))
	if result != "null" {
		t.Errorf("expected null, got %s", result)
	}
}

func TestAppendInt(t *testing.T) {
	tests := []struct {
		input    int64
		expected string
	}{
		{0, "0"},
		{42, "42"},
		{-42, "-42"},
		{9223372036854775807, "9223372036854775807"},
		{-9223372036854775808, "-9223372036854775808"},
	}
	for _, tt := range tests {
		result := string(jsonlite.AppendInt(nil, tt.input))
		if result != tt.expected {
			t.Errorf("AppendInt(%d) = %s, expected %s", tt.input, result, tt.expected)
		}
	}
}

func TestAppendUint(t *testing.T) {
	tests := []struct {
		input    uint64
		expected string
	}{
		{0, "0"},
		{42, "42"},
		{18446744073709551615, "18446744073709551615"},
	}
	for _, tt := range tests {
		result := string(jsonlite.AppendUint(nil, tt.input))
		if result != tt.expected {
			t.Errorf("AppendUint(%d) = %s, expected %s", tt.input, result, tt.expected)
		}
	}
}

func TestAppendFloat(t *testing.T) {
	tests := []struct {
		input    float64
		expected string
	}{
		{0, "0"},
		{1.5, "1.5"},
		{-3.14, "-3.14"},
	}
	for _, tt := range tests {
		result := string(jsonlite.AppendFloat(nil, tt.input))
		if result != tt.expected {
			t.Errorf("AppendFloat(%v) = %s, expected %s", tt.input, result, tt.expected)
		}
	}
}

func TestAppendBool(t *testing.T) {
	if string(jsonlite.AppendBool(nil, true)) != "true" {
		t.Error("AppendBool(true) should return true")
	}
	if string(jsonlite.AppendBool(nil, false)) != "false" {
		t.Error("AppendBool(false) should return false")
	}
}
