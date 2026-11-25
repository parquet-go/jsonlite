//go:build go1.27

package jsonlite

import (
	"math/rand"
	"slices"
	"strconv"
	"testing"
)

// structuralIndexFunc is structuralIndex or a specific kernel.
type structuralIndexFunc func(string, []uint32) ([]uint32, stage1Flags, error)

// TestStructuralIndexMatchesPortable checks that whichever structuralIndex
// this build dispatches to emits the same positions, flags, and errors as
// structuralIndexPortable. On Go 1.26 the AVX2 kernel predates this check
// and is left unchanged, so the test is gated to Go 1.27.
func TestStructuralIndexMatchesPortable(t *testing.T) {
	for _, input := range structuralMatchInputs(t) {
		assertStructuralMatch(t, input, structuralIndex)
	}
}

func FuzzStructuralIndexMatchesPortable(f *testing.F) {
	for _, input := range []string{
		``, `null`, `{"a":1}`, `"hello"`, "\"a\\nb\"",
		string([]byte{0x0C, 0x1A, '"', '\\', 0x80}),
	} {
		f.Add(input)
	}
	for _, input := range differentialInputs(f) {
		f.Add(input)
	}
	f.Fuzz(func(t *testing.T, input string) {
		assertStructuralMatch(t, input, structuralIndex)
	})
}

func structuralMatchInputs(t testing.TB) []string {
	inputs := differentialInputs(t)
	for b := range 256 {
		inputs = append(inputs, string([]byte{byte(b)}))
		buf := make([]byte, 128)
		for i := range buf {
			buf[i] = byte(b)
		}
		inputs = append(inputs, string(buf))
	}
	base := make([]byte, 160)
	for i := range base {
		base[i] = 'a'
	}
	specials := []byte{'"', '\\', '{', '}', '[', ']', ',', ':', ' ', '\t', '\n', '\r', 0x00, 0x0C, 0x1A, 0x1F, 0x7F, 0x80, 0xC3, 0xFF}
	for pos := range len(base) {
		for _, c := range specials {
			buf := append([]byte(nil), base...)
			buf[pos] = c
			inputs = append(inputs, string(buf))
		}
	}
	rng := rand.New(rand.NewSource(1))
	for range 200 {
		buf := make([]byte, rng.Intn(300))
		for i := range buf {
			buf[i] = byte(rng.Intn(256))
		}
		inputs = append(inputs, string(buf))
	}
	return inputs
}

func assertStructuralMatch(t *testing.T, input string, fn structuralIndexFunc) {
	t.Helper()
	got, gotFlags, gotErr := fn(input, nil)
	want, wantFlags, wantErr := structuralIndexPortable(input, nil)
	if gotFlags != wantFlags || !slices.Equal(got, want) || !sameIndexErr(gotErr, wantErr) {
		shown := input
		if len(shown) > 80 {
			shown = shown[:80]
		}
		t.Fatalf("input %s:\n got index=%v flags=%d err=%v\nwant index=%v flags=%d err=%v",
			strconv.Quote(shown), got, gotFlags, gotErr, want, wantFlags, wantErr)
	}
}

func sameIndexErr(a, b error) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return a.Error() == b.Error()
}
