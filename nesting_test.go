package jsonlite_test

import (
	"strings"
	"testing"

	"github.com/parquet-go/jsonlite"
)

// TestNestingDepthLimit is the regression test for a stack overflow.
//
// parseArray recursed once per nesting level and nothing bounded it: maxDepth
// applied only to objects, so `[[[...]]]` recursed as deep as the input went.
// At around a million levels -- a 2MB document that Valid reports as well
// formed -- the goroutine stack hit the runtime's 1GB ceiling and the process
// died with "fatal error: stack overflow", which recover cannot catch.
//
// The depths here are well past the limit but far below what used to crash,
// so this test fails by reporting an error rather than by killing the test
// binary if the bound is ever removed.
func TestNestingDepthLimit(t *testing.T) {
	deep := strings.Repeat("[", 60000) + strings.Repeat("]", 60000)
	if _, err := jsonlite.Parse(deep); err == nil {
		t.Error("expected an error for input nested past the limit")
	} else if !strings.Contains(err.Error(), "nesting depth") {
		t.Errorf("expected a nesting depth error, got: %v", err)
	}
	// ParseMaxDepth must not be a way around it. maxDepth governs how far
	// objects are eagerly parsed, not how deep recursion may go, and it never
	// applied to arrays at all.
	if _, err := jsonlite.ParseMaxDepth(deep, 1); err == nil {
		t.Error("ParseMaxDepth accepted input nested past the limit")
	}
}

// TestDeepObjectsStayLazy records why only arrays needed the new bound.
//
// parseObject decrements maxDepth and, on reaching zero, stores the rest of
// the object unparsed instead of recursing, so object nesting was already
// bounded by maxDepth and never reached the stack. That is a feature, not a
// failure, and it must keep working: these documents parse without error and
// without tripping the nesting limit, however deep they go.
func TestDeepObjectsStayLazy(t *testing.T) {
	for _, tc := range []struct{ name, open, close string }{
		{"objects", `{"a":`, "}"},
		{"alternating", `[{"a":`, "}]"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			deep := strings.Repeat(tc.open, 60000) + strings.Repeat(tc.close, 60000)
			if _, err := jsonlite.Parse(deep); err != nil {
				t.Errorf("deeply nested objects should parse lazily, got: %v", err)
			}
		})
	}
}

// TestNestingDepthAllowsReasonableDocuments checks the bound is not so tight
// that ordinary documents trip it.
func TestNestingDepthAllowsReasonableDocuments(t *testing.T) {
	for _, depth := range []int{1, 10, 100, 1000, 9999} {
		deep := strings.Repeat("[", depth) + strings.Repeat("]", depth)
		if _, err := jsonlite.Parse(deep); err != nil {
			t.Errorf("depth %d rejected: %v", depth, err)
		}
	}
}

// TestNestingDepthResetsBetweenParses catches the depth counter leaking across
// uses of a pooled parser, which would make later parses fail spuriously.
func TestNestingDepthResetsBetweenParses(t *testing.T) {
	deep := strings.Repeat("[", 9000) + strings.Repeat("]", 9000)
	for range 10 {
		if _, err := jsonlite.Parse(deep); err != nil {
			t.Fatalf("depth 9000 rejected after repeated parses: %v", err)
		}
	}
	// A parse that fails on depth must not leave the counter raised either.
	tooDeep := strings.Repeat("[", 20000) + strings.Repeat("]", 20000)
	for range 10 {
		if _, err := jsonlite.Parse(tooDeep); err == nil {
			t.Fatal("expected a nesting depth error")
		}
		if _, err := jsonlite.Parse(deep); err != nil {
			t.Fatalf("valid document rejected after a depth failure: %v", err)
		}
	}
}
