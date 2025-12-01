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

// Tokenizer is a JSON tokenizer that splits input into tokens.
// It skips whitespace and returns individual JSON tokens one at a time.
type Tokenizer struct {
	json   string
	offset int
}

// Tokenize creates a new Tokenizer for the given JSON string.
func Tokenize(json string) *Tokenizer {
	return &Tokenizer{json: json}
}

// Next returns the next token from the input.
// Returns an empty string and false when there are no more tokens.
func (t *Tokenizer) Next() (token string, ok bool) {
	s := t.json
	i := t.offset

	if i == len(s) {
		return
	}

	if s[i] <= 0x20 { // whitespace?
	skipSpaces:
		for {
			if i == len(s) {
				return
			}
			switch s[i] {
			case ' ', '\t', '\n', '\r':
				i++
			default:
				break skipSpaces
			}
		}
	}

	j := i + 1
	switch s[i] {
	case '"':
		for {
			k := strings.IndexByte(s[j:], '"')
			if k < 0 {
				j = len(s)
				break
			}
			j += k + 1
			for k = j - 2; k > i && s[k] == '\\'; k-- {
			}
			if (j-k)%2 == 0 {
				break
			}
		}
	case ',', ':', '[', ']', '{', '}':
	default:
		for j < len(s) {
			switch s[j] {
			case ' ', '\t', '\n', '\r', '[', ']', '{', '}', ':', ',', '"':
				t.offset = j
				return s[i:j], true
			}
			j++
		}
	}

	t.offset = j
	return s[i:j], true
}

// Parse parses JSON data and returns a pointer to the root Value.
// Returns an error if the JSON is malformed or empty.
func Parse(data string) (*Value, error) {
	tok := Tokenize(data)
	v, err := parseTokens(tok)
	if err != nil {
		return nil, err
	}
	// Check for trailing content after the root value
	if extra, ok := tok.Next(); ok {
		return nil, fmt.Errorf("unexpected token after root value: %q", extra)
	}
	return &v, nil
}

func parseTokens(tokens *Tokenizer) (Value, error) {
	token, ok := tokens.Next()
	if !ok {
		return Value{}, errUnexpectedEndOfObject
	}
	switch token[0] {
	case 'n':
		if token != "null" {
			return Value{}, fmt.Errorf("invalid token: %q", token)
		}
		return makeNullValue(), nil
	case 't':
		if token != "true" {
			return Value{}, fmt.Errorf("invalid token: %q", token)
		}
		return makeTrueValue(), nil
	case 'f':
		if token != "false" {
			return Value{}, fmt.Errorf("invalid token: %q", token)
		}
		return makeFalseValue(), nil
	case '"':
		s, err := Unquote(token)
		if err != nil {
			return Value{}, fmt.Errorf("invalid token: %q", token)
		}
		return makeStringValue(s), nil
	case '[':
		return parseArray(tokens)
	case '{':
		return parseObject(tokens)
	case ']':
		return Value{}, errEndOfArray
	case '}':
		return Value{}, errEndOfObject
	case '-', '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
		if !validNumber(token) {
			return Value{}, fmt.Errorf("invalid number: %q", token)
		}
		return makeNumberValue(token), nil
	default:
		return Value{}, fmt.Errorf("invalid token: %q", token)
	}
}

func parseArray(tokens *Tokenizer) (Value, error) {
	var elements []Value

	for i := 0; ; i++ {
		if i != 0 {
			token, ok := tokens.Next()
			if !ok {
				return Value{}, errUnexpectedEndOfArray
			}
			if token == "]" {
				break
			}
			if token != "," {
				return Value{}, fmt.Errorf("expected ',' or ']', got %q", token)
			}
		}

		v, err := parseTokens(tokens)
		if err != nil {
			// Only treat errEndOfArray as "empty array" if this is the first element
			// AND we actually saw the ] token directly (not via a nested parse error).
			// Since parseTokens returns errEndOfArray only when it directly sees ],
			// and nested arrays that fail return their own errors, we need to check
			// that err is exactly errEndOfArray and i == 0.
			if i == 0 && err == errEndOfArray {
				return makeArrayValue(elements), nil
			}
			// For trailing comma cases like [1,], we get errEndOfArray from parseTokens
			// seeing ] after the comma - this is an error
			if err == errEndOfArray {
				return Value{}, fmt.Errorf("unexpected ']' after ','")
			}
			return Value{}, err
		}

		if cap(elements) == 0 {
			elements = make([]Value, 0, 8)
		}
		elements = append(elements, v)
	}

	return makeArrayValue(elements), nil
}

func parseObject(tokens *Tokenizer) (Value, error) {
	var fields []field

	for i := 0; ; i++ {
		token, ok := tokens.Next()
		if !ok {
			return Value{}, errUnexpectedEndOfObject
		}
		if token == "}" {
			break
		}

		if i != 0 {
			if token != "," {
				return Value{}, fmt.Errorf("expected ',' or '}', got %q", token)
			}
			token, ok = tokens.Next()
			if !ok {
				return Value{}, errUnexpectedEndOfObject
			}
		}

		key, err := Unquote(token)
		if err != nil {
			return Value{}, fmt.Errorf("invalid key: %q: %w", token, err)
		}

		token, ok = tokens.Next()
		if !ok {
			return Value{}, errUnexpectedEndOfObject
		}
		if token != ":" {
			return Value{}, fmt.Errorf("%q → expected ':', got %q", key, token)
		}

		val, err := parseTokens(tokens)
		if err != nil {
			return Value{}, fmt.Errorf("%q → %w", key, err)
		}
		if cap(fields) == 0 {
			fields = make([]field, 0, 8)
		}
		fields = append(fields, field{k: key, v: val})
	}

	slices.SortFunc(fields, func(a, b field) int {
		return strings.Compare(a.k, b.k)
	})

	return makeObjectValue(fields), nil
}
