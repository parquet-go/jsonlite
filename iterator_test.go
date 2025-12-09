package jsonlite_test

import (
	"fmt"
	"reflect"
	"testing"
	"time"

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

	if val.Len() != 2 {
		t.Fatalf("expected 2 elements, got %d", val.Len())
	}

	// Collect array elements
	var arr []*jsonlite.Value
	for v := range val.Array {
		arr = append(arr, v)
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

func TestIteratorObjectSeq(t *testing.T) {
	input := `{"name": "Alice", "age": 30, "active": true}`
	iter := jsonlite.Iterate(input)

	if !iter.Next() {
		t.Fatal("expected at least one value")
	}

	if iter.Kind() != jsonlite.Object {
		t.Fatalf("expected Object, got %v", iter.Kind())
	}

	// Use Object() to iterate over fields
	fields := make(map[string]jsonlite.Kind)
	for key, err := range iter.Object {
		if err != nil {
			t.Fatal(err)
		}
		fields[key] = iter.Kind()
	}

	if len(fields) != 3 {
		t.Fatalf("expected 3 fields, got %d", len(fields))
	}

	if fields["name"] != jsonlite.String {
		t.Errorf("expected name to be String, got %v", fields["name"])
	}
	if fields["age"] != jsonlite.Number {
		t.Errorf("expected age to be Number, got %v", fields["age"])
	}
	if fields["active"] != jsonlite.True {
		t.Errorf("expected active to be True, got %v", fields["active"])
	}
}

func TestIteratorArraySeq(t *testing.T) {
	input := `[1, "two", true, null]`
	iter := jsonlite.Iterate(input)

	if !iter.Next() {
		t.Fatal("expected at least one value")
	}

	if iter.Kind() != jsonlite.Array {
		t.Fatalf("expected Array, got %v", iter.Kind())
	}

	// Use Array() to iterate over elements
	expectedKinds := []jsonlite.Kind{jsonlite.Number, jsonlite.String, jsonlite.True, jsonlite.Null}
	var gotKinds []jsonlite.Kind

	for idx, err := range iter.Array {
		if err != nil {
			t.Fatal(err)
		}
		if idx != len(gotKinds) {
			t.Errorf("expected index %d, got %d", len(gotKinds), idx)
		}
		gotKinds = append(gotKinds, iter.Kind())
	}

	if len(gotKinds) != len(expectedKinds) {
		t.Fatalf("expected %d elements, got %d", len(expectedKinds), len(gotKinds))
	}

	for i, expected := range expectedKinds {
		if gotKinds[i] != expected {
			t.Errorf("element %d: expected %v, got %v", i, expected, gotKinds[i])
		}
	}
}

func TestIteratorNestedObjectArray(t *testing.T) {
	input := `{"users": [{"name": "Alice"}, {"name": "Bob"}]}`
	iter := jsonlite.Iterate(input)

	if !iter.Next() {
		t.Fatal("expected at least one value")
	}

	var names []string

	for key, err := range iter.Object {
		if err != nil {
			t.Fatal(err)
		}
		if key == "users" {
			if iter.Kind() != jsonlite.Array {
				t.Fatalf("expected Array for users, got %v", iter.Kind())
			}
			for _, err := range iter.Array {
				if err != nil {
					t.Fatal(err)
				}
				if iter.Kind() != jsonlite.Object {
					t.Fatalf("expected Object in users array, got %v", iter.Kind())
				}
				for k, err := range iter.Object {
					if err != nil {
						t.Fatal(err)
					}
					if k == "name" {
						v, err := iter.Value()
						if err != nil {
							t.Fatal(err)
						}
						names = append(names, v.String())
					}
				}
			}
		}
	}

	if len(names) != 2 {
		t.Fatalf("expected 2 names, got %d", len(names))
	}
	if names[0] != "Alice" {
		t.Errorf("expected first name 'Alice', got %q", names[0])
	}
	if names[1] != "Bob" {
		t.Errorf("expected second name 'Bob', got %q", names[1])
	}
}

// TestIteratorNestedArrays tests deeply nested arrays with auto-consume
func TestIteratorNestedArrays(t *testing.T) {
	input := `[[1, 2], [3, 4], [5, 6]]`
	iter := jsonlite.Iterate(input)

	if !iter.Next() {
		t.Fatal("expected at least one value")
	}

	var results [][]int64
	for _, err := range iter.Array {
		if err != nil {
			t.Fatal(err)
		}
		var inner []int64
		for _, err := range iter.Array {
			if err != nil {
				t.Fatal(err)
			}
			v, err := iter.Int()
			if err != nil {
				t.Fatal(err)
			}
			inner = append(inner, v)
		}
		results = append(results, inner)
	}

	expected := [][]int64{{1, 2}, {3, 4}, {5, 6}}
	if !reflect.DeepEqual(results, expected) {
		t.Errorf("expected %v, got %v", expected, results)
	}
}

// TestIteratorNestedObjects tests deeply nested objects with auto-consume
func TestIteratorNestedObjects(t *testing.T) {
	input := `{"a": {"x": 1}, "b": {"y": 2}, "c": {"z": 3}}`
	iter := jsonlite.Iterate(input)

	if !iter.Next() {
		t.Fatal("expected at least one value")
	}

	results := make(map[string]map[string]int64)
	for key, err := range iter.Object {
		if err != nil {
			t.Fatal(err)
		}
		inner := make(map[string]int64)
		for innerKey, err := range iter.Object {
			if err != nil {
				t.Fatal(err)
			}
			v, err := iter.Int()
			if err != nil {
				t.Fatal(err)
			}
			inner[innerKey] = v
		}
		results[key] = inner
	}

	expected := map[string]map[string]int64{
		"a": {"x": 1},
		"b": {"y": 2},
		"c": {"z": 3},
	}
	if !reflect.DeepEqual(results, expected) {
		t.Errorf("expected %v, got %v", expected, results)
	}
}

// TestIteratorMixedNesting tests mixed object/array nesting
func TestIteratorMixedNesting(t *testing.T) {
	input := `{"items": [{"id": 1, "tags": ["a", "b"]}, {"id": 2, "tags": ["c"]}]}`
	iter := jsonlite.Iterate(input)

	if !iter.Next() {
		t.Fatal("expected at least one value")
	}

	type item struct {
		id   int64
		tags []string
	}
	var items []item

	for key, err := range iter.Object {
		if err != nil {
			t.Fatal(err)
		}
		if key == "items" {
			for _, err := range iter.Array {
				if err != nil {
					t.Fatal(err)
				}
				var it item
				for field, err := range iter.Object {
					if err != nil {
						t.Fatal(err)
					}
					switch field {
					case "id":
						it.id, err = iter.Int()
						if err != nil {
							t.Fatal(err)
						}
					case "tags":
						for _, err := range iter.Array {
							if err != nil {
								t.Fatal(err)
							}
							tag, err := iter.String()
							if err != nil {
								t.Fatal(err)
							}
							it.tags = append(it.tags, tag)
						}
					}
				}
				items = append(items, it)
			}
		}
	}

	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}
	if items[0].id != 1 || !reflect.DeepEqual(items[0].tags, []string{"a", "b"}) {
		t.Errorf("item 0: expected {1, [a b]}, got %v", items[0])
	}
	if items[1].id != 2 || !reflect.DeepEqual(items[1].tags, []string{"c"}) {
		t.Errorf("item 1: expected {2, [c]}, got %v", items[1])
	}
}

// TestIteratorSkipNestedArray tests skipping nested arrays
func TestIteratorSkipNestedArray(t *testing.T) {
	input := `{"skip": [[1, 2], [3, 4]], "keep": "value"}`
	iter := jsonlite.Iterate(input)

	if !iter.Next() {
		t.Fatal("expected at least one value")
	}

	var keepValue string
	for key, err := range iter.Object {
		if err != nil {
			t.Fatal(err)
		}
		if key == "keep" {
			keepValue, err = iter.String()
			if err != nil {
				t.Fatal(err)
			}
		}
		// "skip" is auto-consumed
	}

	if keepValue != "value" {
		t.Errorf("expected 'value', got %q", keepValue)
	}
}

// TestIteratorSkipNestedObject tests skipping nested objects
func TestIteratorSkipNestedObject(t *testing.T) {
	input := `{"skip": {"a": {"b": {"c": 1}}}, "keep": 42}`
	iter := jsonlite.Iterate(input)

	if !iter.Next() {
		t.Fatal("expected at least one value")
	}

	var keepValue int64
	for key, err := range iter.Object {
		if err != nil {
			t.Fatal(err)
		}
		if key == "keep" {
			keepValue, err = iter.Int()
			if err != nil {
				t.Fatal(err)
			}
		}
		// "skip" is auto-consumed
	}

	if keepValue != 42 {
		t.Errorf("expected 42, got %d", keepValue)
	}
}

// TestIteratorAlternateConsumeSkip tests alternating between consuming and skipping
func TestIteratorAlternateConsumeSkip(t *testing.T) {
	input := `[{"a": 1}, [2, 3], {"b": 4}, [5, 6], {"c": 7}]`
	iter := jsonlite.Iterate(input)

	if !iter.Next() {
		t.Fatal("expected at least one value")
	}

	var objectValues []int64
	for i, err := range iter.Array {
		if err != nil {
			t.Fatal(err)
		}
		// Only consume objects (at even indices)
		if i%2 == 0 {
			for _, err := range iter.Object {
				if err != nil {
					t.Fatal(err)
				}
				v, err := iter.Int()
				if err != nil {
					t.Fatal(err)
				}
				objectValues = append(objectValues, v)
			}
		}
		// Arrays at odd indices are auto-skipped
	}

	expected := []int64{1, 4, 7}
	if !reflect.DeepEqual(objectValues, expected) {
		t.Errorf("expected %v, got %v", expected, objectValues)
	}
}

// TestIteratorConsumeWithValue tests using Value() to consume nested structures
func TestIteratorConsumeWithValue(t *testing.T) {
	input := `{"nested": {"a": [1, 2, 3], "b": "test"}, "simple": 42}`
	iter := jsonlite.Iterate(input)

	if !iter.Next() {
		t.Fatal("expected at least one value")
	}

	var nestedValue *jsonlite.Value
	var simpleValue int64
	for key, err := range iter.Object {
		if err != nil {
			t.Fatal(err)
		}
		switch key {
		case "nested":
			nestedValue, err = iter.Value()
			if err != nil {
				t.Fatal(err)
			}
		case "simple":
			simpleValue, err = iter.Int()
			if err != nil {
				t.Fatal(err)
			}
		}
	}

	if nestedValue == nil {
		t.Fatal("nested value is nil")
	}
	if nestedValue.Kind() != jsonlite.Object {
		t.Errorf("expected nested to be Object, got %v", nestedValue.Kind())
	}
	if simpleValue != 42 {
		t.Errorf("expected simple=42, got %d", simpleValue)
	}
}

// TestIteratorEmptyNestedStructures tests empty nested arrays and objects
func TestIteratorEmptyNestedStructures(t *testing.T) {
	input := `{"empty_arr": [], "empty_obj": {}, "arr_of_empty": [{}, [], {}], "after": "ok"}`
	iter := jsonlite.Iterate(input)

	if !iter.Next() {
		t.Fatal("expected at least one value")
	}

	var keys []string
	var afterValue string
	for key, err := range iter.Object {
		if err != nil {
			t.Fatal(err)
		}
		keys = append(keys, key)
		switch key {
		case "empty_arr":
			count := 0
			for _, err := range iter.Array {
				if err != nil {
					t.Fatal(err)
				}
				count++
			}
			if count != 0 {
				t.Errorf("expected empty array, got %d elements", count)
			}
		case "empty_obj":
			count := 0
			for _, err := range iter.Object {
				if err != nil {
					t.Fatal(err)
				}
				count++
			}
			if count != 0 {
				t.Errorf("expected empty object, got %d fields", count)
			}
		case "arr_of_empty":
			count := 0
			for _, err := range iter.Array {
				if err != nil {
					t.Fatal(err)
				}
				count++
				// Don't consume, let them auto-skip
			}
			if count != 3 {
				t.Errorf("expected 3 elements, got %d", count)
			}
		case "after":
			afterValue, err = iter.String()
			if err != nil {
				t.Fatal(err)
			}
		}
	}

	expectedKeys := []string{"empty_arr", "empty_obj", "arr_of_empty", "after"}
	if !reflect.DeepEqual(keys, expectedKeys) {
		t.Errorf("expected keys %v, got %v", expectedKeys, keys)
	}
	if afterValue != "ok" {
		t.Errorf("expected after='ok', got %q", afterValue)
	}
}

// TestIteratorDeeplyNested tests very deeply nested structures
func TestIteratorDeeplyNested(t *testing.T) {
	input := `{"l1": {"l2": {"l3": {"l4": {"value": 42}}}}}`
	iter := jsonlite.Iterate(input)

	if !iter.Next() {
		t.Fatal("expected at least one value")
	}

	var value int64
	for _, err := range iter.Object { // l1
		if err != nil {
			t.Fatal(err)
		}
		for _, err := range iter.Object { // l2
			if err != nil {
				t.Fatal(err)
			}
			for _, err := range iter.Object { // l3
				if err != nil {
					t.Fatal(err)
				}
				for _, err := range iter.Object { // l4
					if err != nil {
						t.Fatal(err)
					}
					for key, err := range iter.Object { // value
						if err != nil {
							t.Fatal(err)
						}
						if key == "value" {
							value, err = iter.Int()
							if err != nil {
								t.Fatal(err)
							}
						}
					}
				}
			}
		}
	}

	if value != 42 {
		t.Errorf("expected 42, got %d", value)
	}
}

// TestIteratorArrayOfArrays tests array containing arrays with partial consumption
func TestIteratorArrayOfArrays(t *testing.T) {
	input := `[[1, 2, 3], [4, 5, 6], [7, 8, 9]]`
	iter := jsonlite.Iterate(input)

	if !iter.Next() {
		t.Fatal("expected at least one value")
	}

	// Only get first element of each inner array
	var firstElements []int64
	for _, err := range iter.Array {
		if err != nil {
			t.Fatal(err)
		}
		for i, err := range iter.Array {
			if err != nil {
				t.Fatal(err)
			}
			if i == 0 {
				v, err := iter.Int()
				if err != nil {
					t.Fatal(err)
				}
				firstElements = append(firstElements, v)
			}
			// Rest are auto-skipped
		}
	}

	expected := []int64{1, 4, 7}
	if !reflect.DeepEqual(firstElements, expected) {
		t.Errorf("expected %v, got %v", expected, firstElements)
	}
}

// TestIteratorObjectValueThenSkip tests consuming first field, skipping rest
func TestIteratorObjectValueThenSkip(t *testing.T) {
	input := `[{"id": 1, "data": {"nested": "value"}}, {"id": 2, "data": [1, 2, 3]}]`
	iter := jsonlite.Iterate(input)

	if !iter.Next() {
		t.Fatal("expected at least one value")
	}

	var ids []int64
	for _, err := range iter.Array {
		if err != nil {
			t.Fatal(err)
		}
		for key, err := range iter.Object {
			if err != nil {
				t.Fatal(err)
			}
			if key == "id" {
				id, err := iter.Int()
				if err != nil {
					t.Fatal(err)
				}
				ids = append(ids, id)
			}
			// "data" fields are auto-skipped
		}
	}

	expected := []int64{1, 2}
	if !reflect.DeepEqual(ids, expected) {
		t.Errorf("expected %v, got %v", expected, ids)
	}
}

// TestIteratorBreakFromNestedLoop tests breaking out of nested iteration
func TestIteratorBreakFromNestedLoop(t *testing.T) {
	input := `{"items": [{"id": 1}, {"id": 2}, {"id": 3}], "other": "value"}`
	iter := jsonlite.Iterate(input)

	if !iter.Next() {
		t.Fatal("expected at least one value")
	}

	var foundId int64
	var otherValue string
outerLoop:
	for key, err := range iter.Object {
		if err != nil {
			t.Fatal(err)
		}
		switch key {
		case "items":
			for _, err := range iter.Array {
				if err != nil {
					t.Fatal(err)
				}
				for field, err := range iter.Object {
					if err != nil {
						t.Fatal(err)
					}
					if field == "id" {
						foundId, err = iter.Int()
						if err != nil {
							t.Fatal(err)
						}
						if foundId == 2 {
							// Found what we want, but we're mid-iteration
							// This tests that we handle partial iteration correctly
							break outerLoop
						}
					}
				}
				if foundId == 2 {
					break
				}
			}
		case "other":
			otherValue, err = iter.String()
			if err != nil {
				t.Fatal(err)
			}
		}
		if foundId == 2 {
			break
		}
	}

	if foundId != 2 {
		t.Errorf("expected foundId=2, got %d", foundId)
	}
	// Note: otherValue won't be set because we broke out early
	if otherValue != "" {
		t.Errorf("expected otherValue='', got %q", otherValue)
	}
}

// TestIteratorNullObject tests that null is treated as an empty object
func TestIteratorNullObject(t *testing.T) {
	input := `{"data": null, "after": "ok"}`
	iter := jsonlite.Iterate(input)

	if !iter.Next() {
		t.Fatal("expected at least one value")
	}

	var afterValue string
	var dataIterations int
	for key, err := range iter.Object {
		if err != nil {
			t.Fatal(err)
		}
		switch key {
		case "data":
			// Iterate over null as if it were an object - should yield nothing
			for _, err := range iter.Object {
				if err != nil {
					t.Fatal(err)
				}
				dataIterations++
			}
		case "after":
			afterValue, err = iter.String()
			if err != nil {
				t.Fatal(err)
			}
		}
	}

	if dataIterations != 0 {
		t.Errorf("expected 0 iterations over null object, got %d", dataIterations)
	}
	if afterValue != "ok" {
		t.Errorf("expected after='ok', got %q", afterValue)
	}
}

// TestIteratorNullArray tests that null is treated as an empty array
func TestIteratorNullArray(t *testing.T) {
	input := `{"items": null, "after": "ok"}`
	iter := jsonlite.Iterate(input)

	if !iter.Next() {
		t.Fatal("expected at least one value")
	}

	var afterValue string
	var itemIterations int
	for key, err := range iter.Object {
		if err != nil {
			t.Fatal(err)
		}
		switch key {
		case "items":
			// Iterate over null as if it were an array - should yield nothing
			for _, err := range iter.Array {
				if err != nil {
					t.Fatal(err)
				}
				itemIterations++
			}
		case "after":
			afterValue, err = iter.String()
			if err != nil {
				t.Fatal(err)
			}
		}
	}

	if itemIterations != 0 {
		t.Errorf("expected 0 iterations over null array, got %d", itemIterations)
	}
	if afterValue != "ok" {
		t.Errorf("expected after='ok', got %q", afterValue)
	}
}

// TestIteratorNullInArray tests null values inside an array
func TestIteratorNullInArray(t *testing.T) {
	input := `[1, null, 3]`
	iter := jsonlite.Iterate(input)

	if !iter.Next() {
		t.Fatal("expected at least one value")
	}

	var values []int64
	var nullCount int
	for _, err := range iter.Array {
		if err != nil {
			t.Fatal(err)
		}
		if iter.Null() {
			nullCount++
			// Still need to "consume" null - Int() handles it
			_, _ = iter.Int()
		} else {
			v, err := iter.Int()
			if err != nil {
				t.Fatal(err)
			}
			values = append(values, v)
		}
	}

	expected := []int64{1, 3}
	if !reflect.DeepEqual(values, expected) {
		t.Errorf("expected %v, got %v", expected, values)
	}
	if nullCount != 1 {
		t.Errorf("expected 1 null, got %d", nullCount)
	}
}

// TestIteratorNullValuesInObject tests null values for object fields
func TestIteratorNullValuesInObject(t *testing.T) {
	input := `{"name": null, "age": null, "active": null}`
	iter := jsonlite.Iterate(input)

	if !iter.Next() {
		t.Fatal("expected at least one value")
	}

	var name string
	var age int64
	var active bool
	for key, err := range iter.Object {
		if err != nil {
			t.Fatal(err)
		}
		switch key {
		case "name":
			name, err = iter.String()
			if err != nil {
				t.Fatal(err)
			}
		case "age":
			age, err = iter.Int()
			if err != nil {
				t.Fatal(err)
			}
		case "active":
			active, err = iter.Bool()
			if err != nil {
				t.Fatal(err)
			}
		}
	}

	// All should be zero values
	if name != "" {
		t.Errorf("expected name='', got %q", name)
	}
	if age != 0 {
		t.Errorf("expected age=0, got %d", age)
	}
	if active != false {
		t.Errorf("expected active=false, got %v", active)
	}
}

func TestIteratorObjectAutoConsume(t *testing.T) {
	// Test that values are automatically consumed when not explicitly read
	input := `{"a": [1, 2, 3], "b": {"nested": "object"}, "c": "simple"}`
	iter := jsonlite.Iterate(input)

	if !iter.Next() {
		t.Fatal("expected at least one value")
	}

	var keys []string
	for key, err := range iter.Object {
		if err != nil {
			t.Fatal(err)
		}
		keys = append(keys, key)
		// Deliberately NOT consuming any values
	}

	if len(keys) != 3 {
		t.Fatalf("expected 3 keys, got %d", len(keys))
	}

	expected := []string{"a", "b", "c"}
	for i, k := range keys {
		if k != expected[i] {
			t.Errorf("key %d: expected %q, got %q", i, expected[i], k)
		}
	}

	if iter.Err() != nil {
		t.Errorf("unexpected error: %v", iter.Err())
	}
}

func TestIteratorArrayAutoConsume(t *testing.T) {
	// Test that values are automatically consumed when not explicitly read
	input := `[[1, 2], {"a": 1}, [3, 4], "simple"]`
	iter := jsonlite.Iterate(input)

	if !iter.Next() {
		t.Fatal("expected at least one value")
	}

	var count int
	for _, err := range iter.Array {
		if err != nil {
			t.Fatal(err)
		}
		count++
		// Deliberately NOT consuming any values
	}

	if count != 4 {
		t.Fatalf("expected 4 elements, got %d", count)
	}

	if iter.Err() != nil {
		t.Errorf("unexpected error: %v", iter.Err())
	}
}

func TestIteratorNestedAutoConsume(t *testing.T) {
	// Test deeply nested auto-consume
	input := `{"users": [{"name": "Alice", "tags": ["a", "b"]}, {"name": "Bob", "tags": ["c", "d"]}], "meta": {"count": 2}}`
	iter := jsonlite.Iterate(input)

	if !iter.Next() {
		t.Fatal("expected at least one value")
	}

	// Only iterate over top-level keys, not consuming any values
	var keys []string
	for key, err := range iter.Object {
		if err != nil {
			t.Fatal(err)
		}
		keys = append(keys, key)
	}

	if len(keys) != 2 {
		t.Fatalf("expected 2 keys, got %d", len(keys))
	}

	if keys[0] != "users" {
		t.Errorf("expected first key 'users', got %q", keys[0])
	}
	if keys[1] != "meta" {
		t.Errorf("expected second key 'meta', got %q", keys[1])
	}

	if iter.Err() != nil {
		t.Errorf("unexpected error: %v", iter.Err())
	}
}

func TestIteratorPartialConsume(t *testing.T) {
	// Test that we can consume some values and auto-skip others
	input := `{"name": "Alice", "skip_this": {"nested": [1,2,3]}, "age": 30}`
	iter := jsonlite.Iterate(input)

	if !iter.Next() {
		t.Fatal("expected at least one value")
	}

	results := make(map[string]interface{})
	for key, err := range iter.Object {
		if err != nil {
			t.Fatal(err)
		}
		if key == "name" || key == "age" {
			val, err := iter.Value()
			if err != nil {
				t.Fatal(err)
			}
			switch key {
			case "name":
				results[key] = val.String()
			case "age":
				results[key] = val.Int()
			}
		}
		// skip_this is not consumed, should be auto-skipped
	}

	if results["name"] != "Alice" {
		t.Errorf("expected name='Alice', got %q", results["name"])
	}
	if results["age"] != int64(30) {
		t.Errorf("expected age=30, got %v", results["age"])
	}

	if iter.Err() != nil {
		t.Errorf("unexpected error: %v", iter.Err())
	}
}

func TestIteratorArrayPartialConsume(t *testing.T) {
	// Test that we can consume some elements and auto-skip others
	input := `[1, [2, 3], 4, {"a": 5}, 6]`
	iter := jsonlite.Iterate(input)

	if !iter.Next() {
		t.Fatal("expected at least one value")
	}

	var numbers []int64
	for idx, err := range iter.Array {
		if err != nil {
			t.Fatal(err)
		}
		// Only consume simple numbers (indices 0, 2, 4)
		if idx == 0 || idx == 2 || idx == 4 {
			val, err := iter.Value()
			if err != nil {
				t.Fatal(err)
			}
			numbers = append(numbers, val.Int())
		}
		// Nested array and object are auto-skipped
	}

	expected := []int64{1, 4, 6}
	if len(numbers) != len(expected) {
		t.Fatalf("expected %d numbers, got %d", len(expected), len(numbers))
	}
	for i, n := range numbers {
		if n != expected[i] {
			t.Errorf("number %d: expected %d, got %d", i, expected[i], n)
		}
	}

	if iter.Err() != nil {
		t.Errorf("unexpected error: %v", iter.Err())
	}
}

func TestIteratorEmptyRangeObject(t *testing.T) {
	// Test that ranging over an object without consuming works
	input := `{"a": 1, "b": 2, "c": 3}`
	iter := jsonlite.Iterate(input)

	if !iter.Next() {
		t.Fatal("expected at least one value")
	}

	// Just range, don't use the values at all (testing the for range iter.Object {} pattern)
	count := 0
	for range iter.Object {
		count++
	}

	if count != 3 {
		t.Fatalf("expected 3 iterations, got %d", count)
	}

	if iter.Err() != nil {
		t.Errorf("unexpected error: %v", iter.Err())
	}
}

func TestIteratorEmptyRangeArray(t *testing.T) {
	// Test that ranging over an array without consuming works
	input := `[{"a": 1}, {"b": 2}, [3, 4, 5]]`
	iter := jsonlite.Iterate(input)

	if !iter.Next() {
		t.Fatal("expected at least one value")
	}

	// Just range, don't use the values at all (testing the for range iter.Array {} pattern)
	count := 0
	for range iter.Array {
		count++
	}

	if count != 3 {
		t.Fatalf("expected 3 iterations, got %d", count)
	}

	if iter.Err() != nil {
		t.Errorf("unexpected error: %v", iter.Err())
	}
}

func TestIteratorValueNoAlloc(t *testing.T) {
	input := `{"name": "Alice", "age": 30, "active": true}`

	// Create iterator outside the allocation measurement
	// We're testing that Value() doesn't allocate, not Iterate()
	iter := jsonlite.Iterate(input)

	allocs := testing.AllocsPerRun(100, func() {
		iter.Reset(input)

		if !iter.Next() {
			t.Fatal("expected at least one value")
		}
		for _, err := range iter.Object {
			if err != nil {
				t.Fatal(err)
			}
			val, err := iter.Value()
			if err != nil {
				t.Fatal(err)
			}
			_ = val
		}
	})

	if allocs != 0 {
		t.Errorf("expected 0 allocations, got %v", allocs)
	}
}

func TestIteratorNull(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{`null`, true},
		{`true`, false},
		{`false`, false},
		{`0`, false},
		{`""`, false},
		{`[]`, false},
		{`{}`, false},
	}

	for _, tt := range tests {
		iter := jsonlite.Iterate(tt.input)
		if !iter.Next() {
			t.Fatalf("iterate %q: expected value", tt.input)
		}
		got := iter.Null()
		if got != tt.expected {
			t.Errorf("iterate %q: expected Null()=%v, got %v", tt.input, tt.expected, got)
		}
	}
}

func TestIteratorBool(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
		wantErr  bool
	}{
		{`true`, true, false},
		{`false`, false, false},
		{`"true"`, true, false},
		{`"false"`, false, false},
		{`"1"`, true, false},
		{`"0"`, false, false},
		{`null`, false, false}, // null returns zero value (false)
		{`123`, false, true},
		{`"invalid"`, false, true},
	}

	for _, tt := range tests {
		iter := jsonlite.Iterate(tt.input)
		if !iter.Next() {
			t.Fatalf("iterate %q: expected value", tt.input)
		}
		got, err := iter.Bool()
		if tt.wantErr {
			if err == nil {
				t.Errorf("iterate %q: expected error, got %v", tt.input, got)
			}
		} else {
			if err != nil {
				t.Errorf("iterate %q: unexpected error: %v", tt.input, err)
			} else if got != tt.expected {
				t.Errorf("iterate %q: expected %v, got %v", tt.input, tt.expected, got)
			}
		}
	}
}

func TestIteratorInt(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
		wantErr  bool
	}{
		{`42`, 42, false},
		{`-123`, -123, false},
		{`0`, 0, false},
		{`"42"`, 42, false},
		{`"-123"`, -123, false},
		{`"0"`, 0, false},
		{`null`, 0, false}, // null returns zero value (0)
		{`3.14`, 0, true},  // float is not a valid int
		{`true`, 0, true},  // bool is not valid
		{`"abc"`, 0, true}, // non-numeric string
	}

	for _, tt := range tests {
		iter := jsonlite.Iterate(tt.input)
		if !iter.Next() {
			t.Fatalf("iterate %q: expected value", tt.input)
		}
		got, err := iter.Int()
		if tt.wantErr {
			if err == nil {
				t.Errorf("iterate %q: expected error, got %v", tt.input, got)
			}
		} else {
			if err != nil {
				t.Errorf("iterate %q: unexpected error: %v", tt.input, err)
			} else if got != tt.expected {
				t.Errorf("iterate %q: expected %v, got %v", tt.input, tt.expected, got)
			}
		}
	}
}

func TestIteratorFloat(t *testing.T) {
	tests := []struct {
		input    string
		expected float64
		wantErr  bool
	}{
		{`3.14`, 3.14, false},
		{`42`, 42.0, false},
		{`-123.456`, -123.456, false},
		{`0`, 0.0, false},
		{`1e10`, 1e10, false},
		{`"3.14"`, 3.14, false},
		{`"42"`, 42.0, false},
		{`"-123.456"`, -123.456, false},
		{`null`, 0, false}, // null returns zero value (0)
		{`true`, 0, true},  // bool is not valid
		{`"abc"`, 0, true}, // non-numeric string
	}

	for _, tt := range tests {
		iter := jsonlite.Iterate(tt.input)
		if !iter.Next() {
			t.Fatalf("iterate %q: expected value", tt.input)
		}
		got, err := iter.Float()
		if tt.wantErr {
			if err == nil {
				t.Errorf("iterate %q: expected error, got %v", tt.input, got)
			}
		} else {
			if err != nil {
				t.Errorf("iterate %q: unexpected error: %v", tt.input, err)
			} else if got != tt.expected {
				t.Errorf("iterate %q: expected %v, got %v", tt.input, tt.expected, got)
			}
		}
	}
}

func TestIteratorString(t *testing.T) {
	tests := []struct {
		input    string
		expected string
		wantErr  bool
	}{
		{`"hello"`, "hello", false},
		{`""`, "", false},
		{`"hello world"`, "hello world", false},
		{`"with\nnewline"`, "with\nnewline", false},
		{`"with\ttab"`, "with\ttab", false},
		{`null`, "", false}, // null returns zero value ("")
		{`42`, "", true},    // number is not valid
		{`true`, "", true},  // bool is not valid
		{`[1,2]`, "", true}, // array is not valid
	}

	for _, tt := range tests {
		iter := jsonlite.Iterate(tt.input)
		if !iter.Next() {
			t.Fatalf("iterate %q: expected value", tt.input)
		}
		got, err := iter.String()
		if tt.wantErr {
			if err == nil {
				t.Errorf("iterate %q: expected error, got %q", tt.input, got)
			}
		} else {
			if err != nil {
				t.Errorf("iterate %q: unexpected error: %v", tt.input, err)
			} else if got != tt.expected {
				t.Errorf("iterate %q: expected %q, got %q", tt.input, tt.expected, got)
			}
		}
	}
}

func TestIteratorDuration(t *testing.T) {
	tests := []struct {
		input    string
		expected time.Duration
		wantErr  bool
	}{
		{`1`, time.Second, false},
		{`0.5`, 500 * time.Millisecond, false},
		{`60`, time.Minute, false},
		{`3600`, time.Hour, false},
		{`"1s"`, time.Second, false},
		{`"500ms"`, 500 * time.Millisecond, false},
		{`"1m"`, time.Minute, false},
		{`"1h"`, time.Hour, false},
		{`"1h30m"`, 90 * time.Minute, false},
		{`null`, 0, false},     // null returns zero value (0)
		{`true`, 0, true},      // bool is not valid
		{`"invalid"`, 0, true}, // invalid duration string
	}

	for _, tt := range tests {
		iter := jsonlite.Iterate(tt.input)
		if !iter.Next() {
			t.Fatalf("iterate %q: expected value", tt.input)
		}
		got, err := iter.Duration()
		if tt.wantErr {
			if err == nil {
				t.Errorf("iterate %q: expected error, got %v", tt.input, got)
			}
		} else {
			if err != nil {
				t.Errorf("iterate %q: unexpected error: %v", tt.input, err)
			} else if got != tt.expected {
				t.Errorf("iterate %q: expected %v, got %v", tt.input, tt.expected, got)
			}
		}
	}
}

func TestIteratorTime(t *testing.T) {
	refTime := time.Date(2023, 6, 15, 12, 30, 45, 0, time.UTC)
	unixTime := float64(refTime.Unix())

	tests := []struct {
		input    string
		expected time.Time
		wantErr  bool
	}{
		{fmt.Sprintf(`%v`, unixTime), refTime, false},
		{`0`, time.Unix(0, 0).UTC(), false},
		{`"2023-06-15T12:30:45Z"`, refTime, false},
		{`"2023-06-15T12:30:45+00:00"`, refTime, false},
		{`null`, time.Time{}, false},        // null returns zero time
		{`true`, time.Time{}, true},         // bool is not valid
		{`"invalid"`, time.Time{}, true},    // invalid time string
		{`"2023-06-15"`, time.Time{}, true}, // wrong format (not RFC3339)
	}

	for _, tt := range tests {
		iter := jsonlite.Iterate(tt.input)
		if !iter.Next() {
			t.Fatalf("iterate %q: expected value", tt.input)
		}
		got, err := iter.Time()
		if tt.wantErr {
			if err == nil {
				t.Errorf("iterate %q: expected error, got %v", tt.input, got)
			}
		} else {
			if err != nil {
				t.Errorf("iterate %q: unexpected error: %v", tt.input, err)
			} else if !got.Equal(tt.expected) {
				t.Errorf("iterate %q: expected %v, got %v", tt.input, tt.expected, got)
			}
		}
	}
}

func TestIteratorTypedValuesInObject(t *testing.T) {
	input := `{"name": "Alice", "age": 30, "score": 95.5, "active": true, "duration": "1h30m", "created": "2023-06-15T12:30:45Z"}`
	iter := jsonlite.Iterate(input)

	if !iter.Next() {
		t.Fatal("expected at least one value")
	}

	results := make(map[string]interface{})
	for key, err := range iter.Object {
		if err != nil {
			t.Fatal(err)
		}
		switch key {
		case "name":
			v, err := iter.String()
			if err != nil {
				t.Fatalf("String() error: %v", err)
			}
			results[key] = v
		case "age":
			v, err := iter.Int()
			if err != nil {
				t.Fatalf("Int() error: %v", err)
			}
			results[key] = v
		case "score":
			v, err := iter.Float()
			if err != nil {
				t.Fatalf("Float() error: %v", err)
			}
			results[key] = v
		case "active":
			v, err := iter.Bool()
			if err != nil {
				t.Fatalf("Bool() error: %v", err)
			}
			results[key] = v
		case "duration":
			v, err := iter.Duration()
			if err != nil {
				t.Fatalf("Duration() error: %v", err)
			}
			results[key] = v
		case "created":
			v, err := iter.Time()
			if err != nil {
				t.Fatalf("Time() error: %v", err)
			}
			results[key] = v
		}
	}

	expected := map[string]interface{}{
		"name":     "Alice",
		"age":      int64(30),
		"score":    95.5,
		"active":   true,
		"duration": 90 * time.Minute,
		"created":  time.Date(2023, 6, 15, 12, 30, 45, 0, time.UTC),
	}

	if !reflect.DeepEqual(results, expected) {
		t.Errorf("expected %v, got %v", expected, results)
	}
}

func BenchmarkIteratorObject(b *testing.B) {
	benchmarks := []struct {
		name  string
		input string
	}{
		{
			name:  "small_object",
			input: `{"a":1,"b":2,"c":3}`,
		},
		{
			name:  "medium_object",
			input: `{"name":"John","age":30,"email":"john@example.com","city":"NYC","country":"USA","active":true,"score":95.5,"tags":["tag1","tag2"],"nested":{"x":1,"y":2}}`,
		},
		{
			name:  "large_object",
			input: `{"field1":"value1","field2":"value2","field3":"value3","field4":"value4","field5":"value5","field6":"value6","field7":"value7","field8":"value8","field9":"value9","field10":"value10","field11":"value11","field12":"value12","field13":"value13","field14":"value14","field15":"value15","field16":"value16","field17":"value17","field18":"value18","field19":"value19","field20":"value20"}`,
		},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				it := jsonlite.Iterate(bm.input)
				if !it.Next() {
					b.Fatal("expected object")
				}
				for _, err := range it.Object {
					if err != nil {
						b.Fatal(err)
					}
				}
			}
		})
	}
}

func BenchmarkIteratorArray(b *testing.B) {
	benchmarks := []struct {
		name  string
		input string
	}{
		{
			name:  "small_array",
			input: `[1,2,3]`,
		},
		{
			name:  "medium_array",
			input: `[1,2,3,4,5,6,7,8,9,10,"a","b","c","d","e",true,false,null,{"x":1},{"y":2}]`,
		},
		{
			name:  "large_array",
			input: `[1,2,3,4,5,6,7,8,9,10,11,12,13,14,15,16,17,18,19,20,21,22,23,24,25,26,27,28,29,30,31,32,33,34,35,36,37,38,39,40,41,42,43,44,45,46,47,48,49,50]`,
		},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				it := jsonlite.Iterate(bm.input)
				if !it.Next() {
					b.Fatal("expected array")
				}
				for _, err := range it.Array {
					if err != nil {
						b.Fatal(err)
					}
				}
			}
		})
	}
}
