package jsonlite

import (
	"fmt"
	"strconv"
	"strings"
	"unicode/utf16"
	"unicode/utf8"
)

const (
	// UTF-16 surrogate pair boundaries (from Unicode standard)
	// Note: These are not valid Unicode scalar values, so we use hex literals
	surrogateMin    = 0xD800 // Start of high surrogate range
	lowSurrogateMin = 0xDC00 // Start of low surrogate range
	lowSurrogateMax = 0xDFFF // End of low surrogate range
)

// Unquote removes quotes from a JSON string and processes escape sequences.
// Returns an error if the string is not properly quoted or contains invalid escapes.
func Unquote(s string) (string, error) {
	if len(s) < 2 || s[0] != '"' || s[len(s)-1] != '"' {
		return "", fmt.Errorf("invalid quoted string: %s", s)
	}
	if strings.IndexByte(s, '\\') < 0 {
		return s[1 : len(s)-1], nil
	}
	b := make([]byte, 0, len(s))
	b, err := AppendUnquote(b, s)
	return string(b), err
}

// AppendUnquote appends the unquoted string to the buffer.
// Returns an error if the string is not properly quoted or contains invalid escapes.
func AppendUnquote(b []byte, s string) ([]byte, error) {
	if len(s) < 2 || s[0] != '"' || s[len(s)-1] != '"' {
		return b, fmt.Errorf("invalid quoted string: %s", s)
	}
	s = s[1 : len(s)-1]

	for {
		i := strings.IndexByte(s, '\\')
		if i < 0 {
			return append(b, s...), nil
		}
		b = append(b, s[:i]...)
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
			r, err := strconv.ParseUint(s[i+2:i+6], 16, 16)
			if err != nil {
				return b, fmt.Errorf("invalid unicode escape sequence: %w", err)
			}

			r1 := rune(r)
			// Check for UTF-16 surrogate pair using utf16 package
			if utf16.IsSurrogate(r1) {
				// Low surrogate without high surrogate is an error
				if r1 >= lowSurrogateMin {
					return b, fmt.Errorf("invalid surrogate pair: unexpected low surrogate")
				}
				// High surrogate, look for low surrogate
				if i+12 > len(s) || s[i+6] != '\\' || s[i+7] != 'u' {
					return b, fmt.Errorf("invalid surrogate pair: missing low surrogate")
				}
				low, err := strconv.ParseUint(s[i+8:i+12], 16, 16)
				if err != nil {
					return b, fmt.Errorf("invalid unicode escape sequence in surrogate pair: %w", err)
				}
				r2 := rune(low)
				if r2 < lowSurrogateMin || r2 > lowSurrogateMax {
					return b, fmt.Errorf("invalid surrogate pair: low surrogate out of range")
				}
				// Decode the surrogate pair using utf16 package
				decoded := utf16.DecodeRune(r1, r2)
				b = utf8.AppendRune(b, decoded)
				s = s[i+12:]
			} else {
				b = utf8.AppendRune(b, r1)
				s = s[i+6:]
			}
		default:
			return b, fmt.Errorf("invalid escape character: %q", c)
		}
	}
}
