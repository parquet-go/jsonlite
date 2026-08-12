package utf8

import (
	"math/rand"
	"strings"
	"testing"
	stdutf8 "unicode/utf8"
)

// boundarySequences covers every interesting encoding boundary: shortest and
// longest encodings per length, surrogates, overlongs, out-of-range, and
// truncations.
func boundarySequences() []string {
	valid := []string{
		"\x00", "\x7F", // 1-byte bounds
		"\xC2\x80", "\xDF\xBF", // 2-byte bounds
		"\xE0\xA0\x80", "\xED\x9F\xBF", "\xEE\x80\x80", "\xEF\xBF\xBF", // 3-byte bounds
		"\xF0\x90\x80\x80", "\xF4\x8F\xBF\xBF", // 4-byte bounds
		"é", "€", "🎉", "日本語",
	}
	invalid := []string{
		"\x80", "\xBF", // stray continuations
		"\xC0\x80", "\xC1\xBF", // overlong 2-byte
		"\xE0\x80\x80", "\xE0\x9F\xBF", // overlong 3-byte
		"\xF0\x80\x80\x80", "\xF0\x8F\xBF\xBF", // overlong 4-byte
		"\xED\xA0\x80", "\xED\xBF\xBF", // surrogates
		"\xF4\x90\x80\x80", "\xF5\x80\x80\x80", "\xFF", "\xFE", // out of range
		"\xC2", "\xE0\xA0", "\xF0\x90\x80", // truncations
		"\xC2\xC2\x80",     // lead then lead
		"\xE2\x82\xAC\x80", // valid 3-byte plus stray continuation
		"\x61\x80",         // ASCII then continuation
		"\xF0\x90\x80",     // truncated 4-byte
	}
	return append(valid, invalid...)
}

// TestValidOracle places every boundary sequence at every offset relative to
// the 16/64-byte block structure, embedded in ASCII padding, and compares
// against unicode/utf8.
func TestValidOracle(t *testing.T) {
	pad := strings.Repeat("a", 200)
	for _, seq := range boundarySequences() {
		for off := 0; off < 70; off++ {
			for _, tailLen := range []int{0, 1, 5, 70} {
				s := pad[:off] + seq + pad[:tailLen]
				got := Valid(s)
				want := stdutf8.ValidString(s)
				if got != want {
					t.Fatalf("Valid(%q) [seq=%q off=%d tail=%d] = %v, want %v",
						s, seq, off, tailLen, got, want)
				}
			}
		}
	}
}

// TestValidRandom compares against the stdlib on random byte strings, biased
// toward interesting bytes.
func TestValidRandom(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	interesting := []byte{
		0x00, 0x41, 0x7F, 0x80, 0x9F, 0xA0, 0xBF, 0xC0, 0xC1, 0xC2, 0xDF,
		0xE0, 0xE1, 0xED, 0xEE, 0xEF, 0xF0, 0xF1, 0xF4, 0xF5, 0xFF,
	}
	for trial := 0; trial < 50000; trial++ {
		n := rng.Intn(200)
		b := make([]byte, n)
		for i := range b {
			if rng.Intn(2) == 0 {
				b[i] = interesting[rng.Intn(len(interesting))]
			} else {
				b[i] = byte(rng.Intn(256))
			}
		}
		s := string(b)
		if got, want := Valid(s), stdutf8.ValidString(s); got != want {
			t.Fatalf("Valid(%q) = %v, want %v", s, got, want)
		}
	}
	// Long valid strings with multibyte runs crossing many block boundaries.
	for trial := 0; trial < 2000; trial++ {
		var sb strings.Builder
		for sb.Len() < 300 {
			switch rng.Intn(4) {
			case 0:
				sb.WriteByte(byte('a' + rng.Intn(26)))
			case 1:
				sb.WriteRune('é')
			case 2:
				sb.WriteRune('€')
			case 3:
				sb.WriteRune('🎉')
			}
		}
		s := sb.String()[:200+rng.Intn(100)]
		if got, want := Valid(s), stdutf8.ValidString(s); got != want {
			t.Fatalf("Valid(%q) = %v, want %v", s, got, want)
		}
	}
}

func FuzzValid(f *testing.F) {
	for _, seq := range boundarySequences() {
		f.Add(strings.Repeat("x", 64) + seq)
	}
	f.Fuzz(func(t *testing.T, s string) {
		if got, want := Valid(s), stdutf8.ValidString(s); got != want {
			t.Fatalf("Valid(%q) = %v, want %v", s, got, want)
		}
	})
}

func benchInputs() []struct{ name, data string } {
	ascii1k := strings.Repeat(`{"level":"info","msg":"request served"} `, 26)[:1024]
	ascii64k := strings.Repeat(ascii1k, 64)
	mixed := strings.Repeat(`{"user":"José","city":"São Paulo","emoji":"🎉"} `, 20)
	unicodeHeavy := strings.Repeat("日本語のテキストです。", 31)
	return []struct{ name, data string }{
		{"ascii_1k", ascii1k},
		{"ascii_64k", ascii64k},
		{"mixed_1k", mixed},
		{"unicode_1k", unicodeHeavy},
		{"small_ascii_48", ascii1k[:48]},
	}
}

func BenchmarkValid(b *testing.B) {
	for _, in := range benchInputs() {
		b.Run(in.name, func(b *testing.B) {
			b.SetBytes(int64(len(in.data)))
			for b.Loop() {
				if !Valid(in.data) {
					b.Fatal("invalid")
				}
			}
		})
	}
}

func BenchmarkValidStdlib(b *testing.B) {
	for _, in := range benchInputs() {
		b.Run(in.name, func(b *testing.B) {
			b.SetBytes(int64(len(in.data)))
			for b.Loop() {
				if !stdutf8.ValidString(in.data) {
					b.Fatal("invalid")
				}
			}
		})
	}
}
