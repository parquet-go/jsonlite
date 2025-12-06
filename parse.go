package jsonlite

import (
	"errors"
	"fmt"
	"slices"
	"strings"
)

var (
	errEndOfObject           = errors.New("}")
	errEndOfArray            = errors.New("]")
	errUnexpectedEndOfObject = errors.New("unexpected end of object")
	errUnexpectedEndOfArray  = errors.New("unexpected end of array")
)

// whitespaceMap is a 256-bit lookup table for ASCII whitespace characters.
// Bit i is set if byte i is whitespace (space, tab, newline, carriage return).
var whitespaceMap = func() [4]uint64 {
	var m [4]uint64
	for _, c := range []byte{' ', '\t', '\n', '\r'} {
		m[c/64] |= 1 << (c % 64)
	}
	return m
}()

// isWhitespace returns true if c is a JSON whitespace character.
func isWhitespace(c byte) bool {
	return (whitespaceMap[c/64] & (1 << (c % 64))) != 0
}

// delimiterMap is a 256-bit lookup table for JSON delimiters and whitespace.
// Used to quickly find the end of numbers/literals.
var delimiterMap = func() [4]uint64 {
	var m [4]uint64
	for _, c := range []byte{' ', '\t', '\n', '\r', '[', ']', '{', '}', ':', ',', '"'} {
		m[c/64] |= 1 << (c % 64)
	}
	return m
}()

// isDelimiter returns true if c is a JSON delimiter or whitespace.
func isDelimiter(c byte) bool {
	return (delimiterMap[c/64] & (1 << (c % 64))) != 0
}

// Tokenizer is a JSON tokenizer that splits input into tokens.
// It skips whitespace and returns individual JSON tokens one at a time.
type Tokenizer struct {
	json string
}

// Tokenize creates a new Tokenizer for the given JSON string.
func Tokenize(json string) *Tokenizer {
	return &Tokenizer{json: json}
}

// Next returns the next token from the input.
// Returns an empty string and false when there are no more tokens.
func (t *Tokenizer) Next() (token string, ok bool) {
	token, t.json, ok = nextToken(t.json)
	return token, ok
}

// nextToken extracts the next JSON token from s.
// Returns the token, the remaining string after the token, and whether a token was found.
// All values are kept in registers - no heap allocation for tokenizer state.
func nextToken(s string) (token, rest string, ok bool) {
	// Skip leading whitespace using lookup table
	switch {
	case len(s) == 0:
		return "", "", false
	case s[0] <= ' ':
		for isWhitespace(s[0]) {
			if s = s[1:]; len(s) == 0 {
				return "", "", false
			}
		}
	}

	switch s[0] {
	case '"':
		// Find closing quote, handling escapes
		j := 1
		for {
			k := strings.IndexByte(s[j:], '"')
			if k < 0 {
				return s, "", true
			}
			j += k + 1
			// Count preceding backslashes to check if quote is escaped
			n := 0
			for i := j - 2; i > 0 && s[i] == '\\'; i-- {
				n++
			}
			if n%2 == 0 {
				return s[:j], s[j:], true
			}
		}
	case ',', ':', '[', ']', '{', '}':
		return s[:1], s[1:], true
	default:
		// Numbers and literals: scan until delimiter using lookup table
		j := 1
		for j < len(s) && !isDelimiter(s[j]) {
			j++
		}
		return s[:j], s[j:], true
	}
}

// Parse parses JSON data and returns a pointer to the root Value.
// Returns an error if the JSON is malformed or empty.
func Parse(data string) (*Value, error) {
	v, rest, err := parseValue(data)
	if err != nil {
		return nil, err
	}
	// Check for trailing content after the root value
	if extra, _, ok := nextToken(rest); ok {
		return nil, fmt.Errorf("unexpected token after root value: %q", extra)
	}
	return &v, nil
}

// parseValue parses a JSON value from s.
// Returns the parsed value, the remaining unparsed string, and any error.
// The string is passed by value to keep it in registers.
func parseValue(s string) (Value, string, error) {
	token, rest, ok := nextToken(s)
	if !ok {
		return Value{}, rest, errUnexpectedEndOfObject
	}
	switch token[0] {
	case 'n':
		if token != "null" {
			return Value{}, rest, fmt.Errorf("invalid token: %q", token)
		}
		return makeNullValue(token[:4]), rest, nil
	case 't':
		if token != "true" {
			return Value{}, rest, fmt.Errorf("invalid token: %q", token)
		}
		return makeTrueValue(token[:4]), rest, nil
	case 'f':
		if token != "false" {
			return Value{}, rest, fmt.Errorf("invalid token: %q", token)
		}
		return makeFalseValue(token[:5]), rest, nil
	case '"':
		str, err := Unquote(token)
		if err != nil {
			return Value{}, rest, fmt.Errorf("invalid token: %q", token)
		}
		return makeStringValue(str), rest, nil
	case '[':
		return parseArray(s, rest)
	case '{':
		return parseObject(s, rest)
	case ']':
		return Value{}, rest, errEndOfArray
	case '}':
		return Value{}, rest, errEndOfObject
	case '-', '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
		if !validNumber(token) {
			return Value{}, rest, fmt.Errorf("invalid number: %q", token)
		}
		return makeNumberValue(token), rest, nil
	default:
		return Value{}, rest, fmt.Errorf("invalid token: %q", token)
	}
}

func parseArray(start, json string) (Value, string, error) {
	elements := make([]Value, 0, 32)

	for i := 0; ; i++ {
		if i != 0 {
			token, rest, ok := nextToken(json)
			if !ok {
				return Value{}, json, errUnexpectedEndOfArray
			}
			if token == "]" {
				cached := start[:len(start)-len(rest)]
				result := make([]Value, len(elements)+1)
				result[0] = makeStringValue(cached)
				copy(result[1:], elements)
				return makeArrayValue(result), rest, nil
			}
			if token != "," {
				return Value{}, json, fmt.Errorf("expected ',' or ']', got %q", token)
			}
			json = rest
		}

		v, rest, err := parseValue(json)
		if err != nil {
			if i == 0 && err == errEndOfArray {
				cached := start[:len(start)-len(rest)]
				result := make([]Value, len(elements)+1)
				result[0] = makeStringValue(cached)
				copy(result[1:], elements)
				return makeArrayValue(result), rest, nil
			}
			if err == errEndOfArray {
				return Value{}, json, fmt.Errorf("unexpected ']' after ','")
			}
			return Value{}, json, err
		}
		json = rest
		elements = append(elements, v)
	}
}

func parseObject(start, json string) (Value, string, error) {
	fields := make([]field, 0, 16)

	for i := 0; ; i++ {
		token, rest, ok := nextToken(json)
		if !ok {
			return Value{}, json, errUnexpectedEndOfObject
		}
		if token == "}" {
			cached := start[:len(start)-len(rest)]
			result := make([]field, len(fields)+1)
			result[0].k = cached
			copy(result[1:], fields)
			// Sort only the real fields (keep cached JSON at index 0)
			slices.SortFunc(result[1:], func(a, b field) int {
				return strings.Compare(a.k, b.k)
			})
			return makeObjectValue(result), rest, nil
		}
		json = rest

		if i != 0 {
			if token != "," {
				return Value{}, json, fmt.Errorf("expected ',' or '}', got %q", token)
			}
			token, rest, ok = nextToken(json)
			if !ok {
				return Value{}, json, errUnexpectedEndOfObject
			}
			json = rest
		}

		key, err := Unquote(token)
		if err != nil {
			return Value{}, json, fmt.Errorf("invalid key: %q: %w", token, err)
		}

		token, rest, ok = nextToken(json)
		if !ok {
			return Value{}, json, errUnexpectedEndOfObject
		}
		if token != ":" {
			return Value{}, json, fmt.Errorf("%q → expected ':', got %q", key, token)
		}
		json = rest

		val, rest, err := parseValue(json)
		if err != nil {
			return Value{}, json, fmt.Errorf("%q → %w", key, err)
		}
		json = rest
		fields = append(fields, field{k: key, v: val})
	}
}
