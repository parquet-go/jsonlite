package jsonlite

import "fmt"

// Iterator provides a streaming interface for traversing JSON values.
// It automatically handles control tokens (braces, brackets, colons, commas)
// and presents only the logical JSON values to the caller.
type Iterator struct {
	tokens Tokenizer
	token  string
	kind   Kind
	key    string
	err    error
	depth  int    // track nesting depth
	state  []byte // stack of states: 'a' for array, 'o' for object (expecting key), 'v' for object (expecting value)
	bytes  [16]byte
}

// Iterate creates a new Iterator for the given JSON string.
func Iterate(json string) *Iterator {
	it := &Iterator{tokens: Tokenizer{json: json}}
	it.state = it.bytes[:0]
	return it
}

// Next advances the iterator to the next JSON value.
// Returns true if there is a value to process, false when done or on error.
func (it *Iterator) Next() bool {
	for {
		token, ok := it.tokens.Next()
		if !ok {
			if len(it.state) > 0 {
				if it.state[len(it.state)-1] == 'a' {
					it.err = errUnexpectedEndOfArray
				} else {
					it.err = errUnexpectedEndOfObject
				}
			}
			return false
		}

		if len(it.state) > 0 {
			s := it.state[len(it.state)-1]
			switch s {
			case 'a': // in array, expecting value or ]
				if token == "]" {
					it.state = it.state[:len(it.state)-1]
					it.depth--
					continue
				}
				if token == "," {
					continue
				}
			case 'o': // in object, expecting key or }
				if token == "}" {
					it.state = it.state[:len(it.state)-1]
					it.depth--
					continue
				}
				if token == "," {
					continue
				}
				// Must be a key
				if token[0] != '"' {
					it.err = fmt.Errorf("expected string key, got %q", token)
					return false
				}
				key, err := Unquote(token)
				if err != nil {
					it.err = fmt.Errorf("invalid key: %q: %w", token, err)
					return false
				}
				it.key = key
				// Now expect colon
				colon, ok := it.tokens.Next()
				if !ok {
					it.err = errUnexpectedEndOfObject
					return false
				}
				if colon != ":" {
					it.err = fmt.Errorf("expected ':', got %q", colon)
					return false
				}
				// Change state to expect value
				it.state[len(it.state)-1] = 'v'
				continue
			case 'v': // in object, expecting value
				// Change state back to expect key/}
				it.state[len(it.state)-1] = 'o'
			}
		}

		it.token = token
		switch token[0] {
		case 'n':
			if token != "null" {
				it.err = fmt.Errorf("invalid token: %q", token)
				return false
			}
			it.kind = Null
		case 't':
			if token != "true" {
				it.err = fmt.Errorf("invalid token: %q", token)
				return false
			}
			it.kind = True
		case 'f':
			if token != "false" {
				it.err = fmt.Errorf("invalid token: %q", token)
				return false
			}
			it.kind = False
		case '"':
			it.kind = String
		case '[':
			it.kind = Array
			it.state = append(it.state, 'a')
			it.depth++
		case '{':
			it.kind = Object
			it.state = append(it.state, 'o')
			it.depth++
		case '-', '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
			it.kind = Number
		default:
			it.err = fmt.Errorf("invalid token: %q", token)
			return false
		}

		return true
	}
}

// Kind returns the kind of the current value.
func (it *Iterator) Kind() Kind {
	return it.kind
}

// Key returns the object key for the current value, if inside an object.
// Returns an empty string if not inside an object or at the top level.
func (it *Iterator) Key() string {
	return it.key
}

// Err returns any error that occurred during iteration.
func (it *Iterator) Err() error {
	return it.err
}

// Depth returns the current nesting depth (0 at top level).
func (it *Iterator) Depth() int {
	return it.depth
}

// Value parses and returns the current value.
// For arrays and objects, this consumes all nested tokens and returns the
// complete parsed structure.
func (it *Iterator) Value() (Value, error) {
	if it.err != nil {
		return Value{}, it.err
	}

	switch it.kind {
	case Null:
		return makeNullValue(), nil
	case True:
		return makeTrueValue(), nil
	case False:
		return makeFalseValue(), nil
	case Number:
		return makeNumberValue(it.token), nil
	case String:
		s, err := Unquote(it.token)
		if err != nil {
			return Value{}, fmt.Errorf("invalid string: %q", it.token)
		}
		return makeStringValue(s), nil
	case Array:
		val, err := parseArray(&it.tokens)
		if err != nil {
			it.err = err
		}
		// Pop the array state we pushed when we saw '['
		it.state = it.state[:len(it.state)-1]
		it.depth--
		return val, err
	case Object:
		val, err := parseObject(&it.tokens)
		if err != nil {
			it.err = err
		}
		// Pop the object state we pushed when we saw '{'
		it.state = it.state[:len(it.state)-1]
		it.depth--
		return val, err
	default:
		return Value{}, fmt.Errorf("unexpected kind: %v", it.kind)
	}
}
