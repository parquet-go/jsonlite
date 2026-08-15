package jsonlite_test

import (
	"fmt"
	"runtime"
	"strings"
	"testing"

	"github.com/parquet-go/jsonlite"
)

// manyIndexedObjects builds a document with `count` objects that each clear
// smallObjectFields, so every one of them allocates a tag index and they end
// up sharing bump-allocated blocks.
//
// Every object gets a *different* key set. That matters: if each object used
// the same keys it would also compute the same tag bytes, and an allocator
// that handed two objects the same block would be indistinguishable from a
// correct one.
func manyIndexedObjects(count, fields int) (string, func(obj, field int) string) {
	key := func(obj, field int) string { return fmt.Sprintf("o%02d_key_%02d", obj, field) }
	objs := make([]string, count)
	for i := range count {
		parts := make([]string, fields)
		for j := range fields {
			parts[j] = fmt.Sprintf(`"%s":%d`, key(i, j), i*1000+j)
		}
		objs[i] = "{" + strings.Join(parts, ",") + "}"
	}
	return "[" + strings.Join(objs, ",") + "]", key
}

// TestTagArenaIsolation is the regression test for bump-allocating tag
// indexes: every object in a document must see its own tags, not a
// neighbour's, no matter how the blocks were carved up.
func TestTagArenaIsolation(t *testing.T) {
	// Field counts chosen to straddle the block sizes so blocks are shared,
	// exactly filled, and individually exceeded.
	for _, fields := range []int{9, 16, 64, 300, 600} {
		t.Run(fmt.Sprintf("fields=%d", fields), func(t *testing.T) {
			doc, key := manyIndexedObjects(40, fields)
			root, err := jsonlite.Parse(doc)
			if err != nil {
				t.Fatal(err)
			}
			for i := range 40 {
				obj := root.Index(i)
				for j := range fields {
					k := key(i, j)
					v := obj.Lookup(k)
					if v == nil {
						t.Fatalf("object %d: key %q not found", i, k)
					}
					want := int64(i*1000 + j)
					if got := v.Int(); got != want {
						t.Fatalf("object %d key %q: got %d, want %d", i, k, got, want)
					}
				}
				// A key that belongs to a different object must not resolve
				// here; it would if the two shared a tag index.
				if i > 0 && obj.Lookup(key(i-1, 0)) != nil {
					t.Fatalf("object %d: resolved a neighbour's key", i)
				}
				if obj.Lookup("absent") != nil {
					t.Fatalf("object %d: found absent key", i)
				}
			}
		})
	}
}

// TestTagArenaSurvivesParserReuse checks the hand-off: the block an object
// aliases must not be written to by a later parse that draws the same pooled
// parser. A retained object has to keep reading its own tags.
func TestTagArenaSurvivesParserReuse(t *testing.T) {
	doc, key := manyIndexedObjects(8, 12)
	retained, err := jsonlite.Parse(doc)
	if err != nil {
		t.Fatal(err)
	}

	// Churn the parser pool so a later parse reuses the same parser, then
	// force a GC so any block wrongly treated as unreferenced would be reused
	// or collected.
	for range 200 {
		other, _ := manyIndexedObjects(8, 12)
		if _, err := jsonlite.Parse(other); err != nil {
			t.Fatal(err)
		}
	}
	runtime.GC()

	for i := range 8 {
		obj := retained.Index(i)
		for j := range 12 {
			k := key(i, j)
			v := obj.Lookup(k)
			if v == nil {
				t.Fatalf("object %d: key %q lost after parser reuse", i, k)
			}
			if want := int64(i*1000 + j); v.Int() != want {
				t.Fatalf("object %d key %q: got %d, want %d", i, k, v.Int(), want)
			}
		}
	}
	runtime.KeepAlive(retained)
}
