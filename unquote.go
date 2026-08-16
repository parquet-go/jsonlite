package jsonlite

import (
	"fmt"
	"math/bits"
	"unicode"
	"unicode/utf16"
	"unicode/utf8"

	"github.com/parquet-go/bitpack/unsafecast"
)

// Unquote removes quotes from a JSON string and processes escape sequences.
// Returns an error if the string is not properly quoted or contains invalid escapes.
// When the string contains no escape sequences, returns a zero-copy substring.
func Unquote(s string) (string, error) {
	if len(s) < 2 || s[0] != '"' || s[len(s)-1] != '"' {
		return "", fmt.Errorf("invalid quoted string: %s", s)
	}
	s = s[1 : len(s)-1]
	// Fast path: check if string needs unescaping (has backslash or control chars)
	if !escaped(s) {
		return s, nil
	}
	b, err := unquote(make([]byte, 0, len(s)), s)
	return string(b), err
}

// AppendUnquote appends the unquoted string to the buffer.
// Returns an error if the string is not properly quoted or contains invalid escapes.
func AppendUnquote(b []byte, s string) ([]byte, error) {
	if len(s) < 2 || s[0] != '"' || s[len(s)-1] != '"' {
		return b, fmt.Errorf("invalid quoted string: %s", s)
	}
	s = s[1 : len(s)-1]
	// Fast path: check if string needs unescaping
	if !escaped(s) {
		return append(b, s...), nil
	}
	return unquote(b, s)
}

// escaped reports whether s needs unescaping: it contains a backslash or a
// raw control character. Strings are the bulk of a JSON document, so this
// runs over most of the input.
//
// The word-at-a-time scan lives here rather than behind a second call. A
// dispatching wrapper that called out to it cost 2-4% across the corpus,
// because every string in a document pays that call and almost all of them
// are too short to reach the vector path anyway.
func escaped(s string) bool {
	if len(s) >= 64 && escapedHasWide() {
		return escapedWide(s)
	}

	// Word-at-a-time scan for a backslash or a control character.
	//
	// Strings are the bulk of a JSON document, so this loop runs over most of
	// the input. The bit tricks only work on bytes below 0x80, so high bytes
	// are masked out with `&^ n`: UTF-8 sequences never need unescaping and
	// must not trigger the slow path.
	var i int

	if len(s) >= 32 {
		// Wide tier. Four words are folded together before the mask is
		// tested, which gives the loads room to overlap, and the cast to
		// [][4]uint64 rather than []uint64 means the four loads index a
		// fixed-size array: one bounds check on the block instead of four
		// inside the body. Same reason stage 1 walks [][64]byte.
		//
		// Entered only at 32 bytes and above. Below that the setup costs
		// more than it saves -- keys and short values are most of the strings
		// in a document, and routing them through here cost 2-9%.
		blocks := unsafecast.Slice[[4]uint64](unsafecast.Bytes(s))
		for bi := range blocks {
			w := &blocks[bi]
			m0 := (below(w[0], 0x20) | contains(w[0], '\\')) &^ w[0]
			m1 := (below(w[1], 0x20) | contains(w[1], '\\')) &^ w[1]
			m2 := (below(w[2], 0x20) | contains(w[2], '\\')) &^ w[2]
			m3 := (below(w[3], 0x20) | contains(w[3], '\\')) &^ w[3]
			if ((m0|m1)|(m2|m3))&msb != 0 {
				return true
			}
		}
		i = len(blocks) * 32
	}

	if len(s)-i >= 8 {
		chunks := unsafecast.Slice[uint64](unsafecast.Bytes(s[i:]))
		for _, n := range chunks {
			mask := (below(n, 0x20) | contains(n, '\\')) &^ n
			if (mask & msb) != 0 {
				return true
			}
		}
		i += len(chunks) * 8
	}

	for ; i < len(s); i++ {
		c := s[i]
		if c < 0x20 || c == '\\' {
			return true
		}
	}

	return false
}

// unescapeIndex checks if the string content needs unescaping.
// Returns -1 if no unescaping needed, or the index of the first problematic byte.
// A string needs unescaping if it contains backslash or control characters (< 0x20).
func unescapeIndex(s string) int {
	// SIMD-like scanning for backslash or control characters.
	// The bit tricks only work correctly when all bytes are < 0x80,
	// so we also check for high bytes and fall back to byte-by-byte.
	var i int
	if len(s) >= 8 {
		chunks := unsafecast.Slice[uint64](unsafecast.Bytes(s))
		for j, n := range chunks {
			// Check for high bytes (>= 0x80), backslash, or control chars
			mask := n | below(n, 0x20) | contains(n, '\\')
			if (mask & msb) != 0 {
				// Found something in this chunk - check byte at the position
				k := j*8 + bits.TrailingZeros64(mask&msb)/8
				c := s[k]
				switch {
				case c < 0x20, c == '\\':
					return k
				default:
					// High byte (>= 0x80) - scan rest of chunk byte by byte
					for k++; k < (j+1)*8; k++ {
						c := s[k]
						if c < 0x20 || c == '\\' {
							return k
						}
					}
				}
			}
		}
		i = len(chunks) * 8
	}

	for ; i < len(s); i++ {
		c := s[i]
		if c < 0x20 || c == '\\' {
			return i
		}
	}

	return -1
}

// unquote processes escape sequences in content and appends to b.
// content should not include the surrounding quotes.
func unquote(b []byte, s string) ([]byte, error) {
	for len(s) > 0 {
		i := unescapeIndex(s)
		if i < 0 {
			return append(b, s...), nil
		}

		b = append(b, s[:i]...)
		c := s[i]
		if c < 0x20 {
			return b, fmt.Errorf("invalid control character in string")
		}
		if i+1 >= len(s) {
			return b, fmt.Errorf("invalid escape sequence at end of string")
		}

		switch c := s[i+1]; c {
		case '"', '\\', '/':
			b = append(b, c)
			s = s[i+2:]
		case 'b':
			b = append(b, '\b')
			s = s[i+2:]
		case 'f':
			b = append(b, '\f')
			s = s[i+2:]
		case 'n':
			b = append(b, '\n')
			s = s[i+2:]
		case 'r':
			b = append(b, '\r')
			s = s[i+2:]
		case 't':
			b = append(b, '\t')
			s = s[i+2:]
		case 'u':
			if i+6 > len(s) {
				return b, fmt.Errorf("invalid unicode escape sequence")
			}
			r1, ok := parseHex4(s[i+2 : i+6])
			if !ok {
				return b, fmt.Errorf("invalid unicode escape sequence")
			}

			// An escape that is not a surrogate stands on its own. A high
			// surrogate pairs with a following low surrogate; anything else
			// is unpaired and decodes to the replacement character, which is
			// what RFC 8259 permits and what encoding/json does. Returning an
			// error here instead silently truncated the string, because
			// Value.String discards it and keeps the partial result.
			rest := s[i+6:]
			switch {
			case !utf16.IsSurrogate(r1):
				b = utf8.AppendRune(b, r1)
			case len(rest) >= 6 && rest[0] == '\\' && rest[1] == 'u':
				r2, ok := parseHex4(rest[2:6])
				if r := utf16.DecodeRune(r1, r2); ok && r != unicode.ReplacementChar {
					b = utf8.AppendRune(b, r)
					rest = rest[6:]
				} else {
					// The following escape is not a low surrogate, so r1 is
					// unpaired. Leave the escape for the next iteration to
					// decode on its own terms.
					b = utf8.AppendRune(b, unicode.ReplacementChar)
				}
			default:
				b = utf8.AppendRune(b, unicode.ReplacementChar)
			}
			s = rest
		default:
			return b, fmt.Errorf("invalid escape character: %q", c)
		}
	}
	return b, nil
}

// parseHex4 parses a 4-character hex string into a rune.
// Returns the rune and true on success, or 0 and false on failure.
func parseHex4(s string) (rune, bool) {
	if len(s) < 4 {
		return 0, false
	}
	var r rune
	for i := range 4 {
		r <<= 4
		c := s[i]
		switch {
		case c >= '0' && c <= '9':
			r |= rune(c - '0')
		case c >= 'a' && c <= 'f':
			r |= rune(c - 'a' + 10)
		case c >= 'A' && c <= 'F':
			r |= rune(c - 'A' + 10)
		default:
			return 0, false
		}
	}
	return r, true
}
