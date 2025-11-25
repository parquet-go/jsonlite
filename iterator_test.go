package jsonlite_test

import (
	"testing"

	"github.com/parquet-go/jsonlite"
)

func TestIteratorBasic(t *testing.T) {
	tests := []struct {
		input string
		kinds []jsonlite.Kind
	}{
		{"null", []jsonlite.Kind{jsonlite.Null}},
		{"true", []jsonlite.Kind{jsonlite.True}},
		{"false", []jsonlite.Kind{jsonlite.False}},
		{"42", []jsonlite.Kind{jsonlite.Number}},
		{`"hello"`, []jsonlite.Kind{jsonlite.String}},
		{"[]", []jsonlite.Kind{jsonlite.Array}},
		{"{}", []jsonlite.Kind{jsonlite.Object}},
	}

	for _, tt := range tests {
		iter := jsonlite.Iterate(tt.input)
		var kinds []jsonlite.Kind
		for iter.Next() {
			kinds = append(kinds, iter.Kind())
		}
		if iter.Err() != nil {
			t.Errorf("iterate %q: %v", tt.input, iter.Err())
		}
		if len(kinds) != len(tt.kinds) {
			t.Errorf("iterate %q: expected %d values, got %d", tt.input, len(tt.kinds), len(kinds))
			continue
		}
		for i := range kinds {
			if kinds[i] != tt.kinds[i] {
				t.Errorf("iterate %q: value %d expected %v, got %v", tt.input, i, tt.kinds[i], kinds[i])
			}
		}
	}
}

func TestIteratorArray(t *testing.T) {
	input := `[1, "two", true, null]`
	iter := jsonlite.Iterate(input)

	expectedKinds := []jsonlite.Kind{
		jsonlite.Array,
		jsonlite.Number,
		jsonlite.String,
		jsonlite.True,
		jsonlite.Null,
	}

	var kinds []jsonlite.Kind
	for iter.Next() {
		kinds = append(kinds, iter.Kind())
	}

	if iter.Err() != nil {
		t.Fatal(iter.Err())
	}

	if len(kinds) != len(expectedKinds) {
		t.Fatalf("expected %d values, got %d", len(expectedKinds), len(kinds))
	}

	for i := range kinds {
		if kinds[i] != expectedKinds[i] {
			t.Errorf("value %d: expected %v, got %v", i, expectedKinds[i], kinds[i])
		}
	}
}

func TestIteratorObject(t *testing.T) {
	input := `{"a": 1, "b": "hello"}`
	iter := jsonlite.Iterate(input)

	type entry struct {
		kind jsonlite.Kind
		key  string
	}

	var entries []entry
	for iter.Next() {
		entries = append(entries, entry{kind: iter.Kind(), key: iter.Key()})
	}

	if iter.Err() != nil {
		t.Fatal(iter.Err())
	}

	// First entry is the object itself
	if len(entries) < 1 || entries[0].kind != jsonlite.Object {
		t.Fatalf("expected first entry to be Object")
	}

	// Remaining entries should be the key-value pairs
	if len(entries) != 3 {
		t.Fatalf("expected 3 entries (object + 2 values), got %d", len(entries))
	}
}

func TestIteratorValue(t *testing.T) {
	input := `{"name": "Alice", "age": 30, "tags": ["a", "b"]}`
	iter := jsonlite.Iterate(input)

	if !iter.Next() {
		t.Fatal("expected at least one value")
	}

	if iter.Kind() != jsonlite.Object {
		t.Fatalf("expected Object, got %v", iter.Kind())
	}

	// Get the whole object as a Value
	val, err := iter.Value()
	if err != nil {
		t.Fatal(err)
	}

	if val.Kind() != jsonlite.Object {
		t.Fatalf("expected Object value, got %v", val.Kind())
	}

	// Check we can access fields
	if name := val.Lookup("name"); name == nil {
		t.Error("expected to find 'name' field")
	} else if name.String() != "Alice" {
		t.Errorf("expected 'Alice', got %q", name.String())
	}

	if tags := val.Lookup("tags"); tags == nil {
		t.Error("expected to find 'tags' field")
	} else if tags.Kind() != jsonlite.Array {
		t.Errorf("expected Array for 'tags', got %v", tags.Kind())
	}
}

func TestIteratorDepth(t *testing.T) {
	input := `{"a": [1, [2, 3]], "b": {"c": 4}}`
	iter := jsonlite.Iterate(input)

	type entry struct {
		kind  jsonlite.Kind
		depth int
	}

	var entries []entry
	for iter.Next() {
		entries = append(entries, entry{kind: iter.Kind(), depth: iter.Depth()})
	}

	if iter.Err() != nil {
		t.Fatal(iter.Err())
	}

	// Expected depths:
	// { depth=1
	//   "a": [ depth=2
	//     1 depth=2
	//     [ depth=3
	//       2 depth=3
	//       3 depth=3
	//     ]
	//   ]
	//   "b": { depth=2
	//     "c": 4 depth=2
	//   }
	// }
	expectedDepths := []int{1, 2, 2, 3, 3, 3, 2, 2}

	if len(entries) != len(expectedDepths) {
		t.Fatalf("expected %d entries, got %d", len(expectedDepths), len(entries))
	}

	for i, e := range entries {
		if e.depth != expectedDepths[i] {
			t.Errorf("entry %d: expected depth %d, got %d", i, expectedDepths[i], e.depth)
		}
	}
}

func TestIteratorNestedValue(t *testing.T) {
	input := `[{"a": 1}, {"b": 2}]`
	iter := jsonlite.Iterate(input)

	if !iter.Next() {
		t.Fatal("expected at least one value")
	}

	if iter.Kind() != jsonlite.Array {
		t.Fatalf("expected Array, got %v", iter.Kind())
	}

	// Get the whole array as a Value
	val, err := iter.Value()
	if err != nil {
		t.Fatal(err)
	}

	if val.Kind() != jsonlite.Array {
		t.Fatalf("expected Array value, got %v", val.Kind())
	}

	arr := val.Array()
	if len(arr) != 2 {
		t.Fatalf("expected 2 elements, got %d", len(arr))
	}

	// Check first object
	if arr[0].Kind() != jsonlite.Object {
		t.Errorf("element 0: expected Object, got %v", arr[0].Kind())
	}
	if a := arr[0].Lookup("a"); a == nil {
		t.Error("element 0: expected to find key 'a'")
	} else if a.Int() != 1 {
		t.Errorf("element 0: expected a=1, got %d", a.Int())
	}

	// Check second object
	if arr[1].Kind() != jsonlite.Object {
		t.Errorf("element 1: expected Object, got %v", arr[1].Kind())
	}
	if b := arr[1].Lookup("b"); b == nil {
		t.Error("element 1: expected to find key 'b'")
	} else if b.Int() != 2 {
		t.Errorf("element 1: expected b=2, got %d", b.Int())
	}
}
