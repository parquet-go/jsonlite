// Package jsonlite provides a lightweight JSON parser optimized for
// performance through careful memory management. It parses JSON into
// a tree of Value nodes that can be inspected and serialized back to JSON.
//
// The parser handles all standard JSON types: null, booleans, numbers,
// strings, arrays, and objects. It properly handles UTF-16 surrogate
// pairs for emoji and extended Unicode characters.
package jsonlite

import (
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"unicode/utf16"
	"unicode/utf8"
	"unsafe"
)

var (
	errEndOfObject           = errors.New("}")
	errEndOfArray            = errors.New("]")
	errUnexpectedEndOfObject = errors.New("unexpected end of object")
	errUnexpectedEndOfArray  = errors.New("unexpected end of array")
)

const (
	// UTF-16 surrogate pair boundaries (from Unicode standard)
	// Note: These are not valid Unicode scalar values, so we use hex literals
	surrogateMin    = 0xD800 // Start of high surrogate range
	lowSurrogateMin = 0xDC00 // Start of low surrogate range
	lowSurrogateMax = 0xDFFF // End of low surrogate range
)

const (
	// kindShift is calculated based on pointer size to use the high bits
	// for the kind field. We have 7 Kind values (0-6), requiring 3 bits.
	// On 64-bit systems this is 61 (top 3 bits for kind, bottom 61 for length),
	// on 32-bit systems this is 29 (top 3 bits for kind, bottom 29 for length).
	kindShift = (unsafe.Sizeof(uintptr(0))*8 - 3)
	kindMask  = (1 << kindShift) - 1
)

// Kind represents the type of a JSON value.
type Kind int

const (
	// Null represents a JSON null value.
	Null Kind = iota
	// True represents a JSON true boolean value.
	True
	// False represents a JSON false boolean value.
	False
	// Number represents a JSON number value.
	Number
	// String represents a JSON string value.
	String
	// Object represents a JSON object value.
	Object
	// Array represents a JSON array value.
	Array
)

// Field represents a key-value pair in a JSON object.
type Field struct {
	Key string
	Val Value
}

// Value represents a JSON value of any type.
type Value struct {
	p unsafe.Pointer
	n uintptr
}

// Kind returns the type of the JSON value.
func (v *Value) Kind() Kind {
	return Kind(v.n >> kindShift)
}

// Len returns the length of the value.
// For strings, it returns the number of bytes.
// For arrays, it returns the number of elements.
// For objects, it returns the number of fields.
// Panics if called on other types.
func (v *Value) Len() int {
	switch v.Kind() {
	case String, Number, Array, Object:
		return int(v.n & kindMask)
	default:
		panic("jsonlite: Len called on non-string/array/object value")
	}
}

// Int returns the value as a signed 64-bit integer.
// Panics if the value is not a number or if parsing fails.
func (v *Value) Int() int64 {
	if v.Kind() != Number {
		panic("jsonlite: Int called on non-number value")
	}
	i, err := strconv.ParseInt(v.raw(), 10, 64)
	if err != nil {
		panic(err)
	}
	return i
}

// Uint returns the value as an unsigned 64-bit integer.
// Panics if the value is not a number or if parsing fails.
func (v *Value) Uint() uint64 {
	if v.Kind() != Number {
		panic("jsonlite: Uint called on non-number value")
	}
	u, err := strconv.ParseUint(v.raw(), 10, 64)
	if err != nil {
		panic(err)
	}
	return u
}

// Float returns the value as a 64-bit floating point number.
// Panics if the value is not a number or if parsing fails.
func (v *Value) Float() float64 {
	if v.Kind() != Number {
		panic("jsonlite: Float called on non-number value")
	}
	f, err := strconv.ParseFloat(v.raw(), 64)
	if err != nil {
		panic(err)
	}
	return f
}

// String returns the value as a string.
// For string and number values, returns the raw value.
// For other types, returns the JSON representation.
func (v *Value) String() string {
	switch v.Kind() {
	case String, Number:
		return v.raw()
	case Null:
		return "null"
	case True:
		return "true"
	case False:
		return "false"
	default:
		return string(v.Append(nil))
	}
}

// Array returns the value as a slice of Values.
// Panics if the value is not an array.
func (v *Value) Array() []Value {
	if v.Kind() != Array {
		panic("jsonlite: Array called on non-array value")
	}
	return unsafe.Slice((*Value)(v.p), v.len())
}

// Object returns the value as a slice of Fields.
// Panics if the value is not an object.
func (v *Value) Object() []Field {
	if v.Kind() != Object {
		panic("jsonlite: Object called on non-object value")
	}
	return unsafe.Slice((*Field)(v.p), v.len())
}

// Lookup searches for a field by key in an object and returns a pointer to its value.
// Returns nil if the key is not found.
// Panics if the value is not an object.
func (v *Value) Lookup(k string) *Value {
	if v.Kind() != Object {
		panic("jsonlite: Lookup called on non-object value")
	}
	fields := v.Object()
	i, ok := slices.BinarySearchFunc(fields, k, func(a Field, b string) int {
		return strings.Compare(a.Key, b)
	})
	if ok {
		return &fields[i].Val
	}
	return nil
}

// NumberType returns the classification of the number (int, uint, or float).
// Panics if the value is not a number.
func (v *Value) NumberType() NumberType {
	if v.Kind() != Number {
		panic("jsonlite: NumberType called on non-number value")
	}
	return NumberTypeOf(v.raw())
}

// Number returns the value as a json.Number.
// Panics if the value is not a number.
func (v *Value) Number() json.Number {
	if v.Kind() != Number {
		panic("jsonlite: Number called on non-number value")
	}
	return json.Number(v.raw())
}

// raw returns the underlying string data without type checking.
// This is used internally by methods that have already verified the type.
func (v *Value) raw() string {
	return unsafe.String((*byte)(v.p), v.len())
}

// len returns the length stored in the value without type checking.
func (v *Value) len() int {
	return int(v.n & kindMask)
}

// NumberType represents the classification of a JSON number.
type NumberType int

const (
	// Int indicates a signed integer number (has a leading minus sign, no decimal point or exponent).
	Int NumberType = iota
	// Uint indicates an unsigned integer number (no minus sign, no decimal point or exponent).
	Uint
	// Float indicates a floating point number (has decimal point or exponent).
	Float
)

// NumberTypeOf returns the classification of a number string.
func NumberTypeOf(s string) NumberType {
	if len(s) == 0 {
		return Float
	}
	t := Uint
	if s[0] == '-' {
		s = s[1:]
		t = Int
	}
	for i := range len(s) {
		switch s[i] {
		case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
			continue
		default:
			return Float
		}
	}
	return t
}

func makeNullValue() Value {
	return Value{n: uintptr(Null) << kindShift}
}

func makeTrueValue() Value {
	return Value{n: uintptr(True)<<kindShift | 1}
}

func makeFalseValue() Value {
	return Value{n: uintptr(False)<<kindShift | 0}
}

func makeNumberValue(s string) Value {
	return Value{
		p: unsafe.Pointer(unsafe.StringData(s)),
		n: (uintptr(Number) << kindShift) | uintptr(len(s)),
	}
}

func makeStringValue(s string) Value {
	return Value{
		p: unsafe.Pointer(unsafe.StringData(s)),
		n: (uintptr(String) << kindShift) | uintptr(len(s)),
	}
}

func makeArrayValue(elements []Value) Value {
	return Value{
		p: unsafe.Pointer(unsafe.SliceData(elements)),
		n: (uintptr(Array) << kindShift) | uintptr(len(elements)),
	}
}

func makeObjectValue(fields []Field) Value {
	return Value{
		p: unsafe.Pointer(unsafe.SliceData(fields)),
		n: (uintptr(Object) << kindShift) | uintptr(len(fields)),
	}
}

type tokenizer struct {
	json   string
	offset int
}

func (t *tokenizer) next() (token string, ok bool) {
	for t.offset < len(t.json) {
		i := t.offset
		j := i + 1
		switch t.json[i] {
		case ' ', '\t', '\n', '\r':
			t.offset++
			continue
		case '[', ']', '{', '}', ':', ',':
			t.offset = j
			return t.json[i:j], true
		case '"':
			for {
				k := strings.IndexByte(t.json[j:], '"')
				if k < 0 {
					j = len(t.json)
					break
				}
				j += k + 1
				for k = j - 2; k > i && t.json[k] == '\\'; k-- {
				}
				if (j-k)%2 == 0 {
					break
				}
			}
			t.offset = j
			return t.json[i:j], true
		default:
			for j < len(t.json) {
				switch t.json[j] {
				case ' ', '\t', '\n', '\r', '[', ']', '{', '}', ':', ',', '"':
					t.offset = j
					return t.json[i:j], true
				}
				j++
			}
			t.offset = j
			return t.json[i:j], true
		}
	}
	return "", false
}

// Parse parses JSON data and returns a pointer to the root Value.
// Returns an error if the JSON is malformed.
func Parse(data string) (*Value, error) {
	if len(data) == 0 {
		v := makeNullValue()
		return &v, nil
	}
	tok := &tokenizer{json: data}
	v, err := parseTokens(tok)
	if err != nil {
		return nil, err
	}
	return &v, nil
}

func parseTokens(tokens *tokenizer) (Value, error) {
	token, ok := tokens.next()
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
		return makeNumberValue(token), nil
	default:
		return Value{}, fmt.Errorf("invalid token: %q", token)
	}
}

func parseArray(tokens *tokenizer) (Value, error) {
	elements := make([]Value, 0, 8)

	for i := 0; ; i++ {
		if i != 0 {
			token, ok := tokens.next()
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
			if i == 0 && errors.Is(err, errEndOfArray) {
				return makeArrayValue(elements), nil
			}
			return Value{}, err
		}
		elements = append(elements, v)
	}

	return makeArrayValue(elements), nil
}

func parseObject(tokens *tokenizer) (Value, error) {
	fields := make([]Field, 0, 8)

	for i := 0; ; i++ {
		if i != 0 {
			token, ok := tokens.next()
			if !ok {
				return Value{}, errUnexpectedEndOfObject
			}
			if token == "}" {
				break
			}
			if token != "," {
				return Value{}, fmt.Errorf("expected ',' or '}', got %q", token)
			}
		}

		token, ok := tokens.next()
		if !ok {
			return Value{}, errUnexpectedEndOfObject
		}
		if i == 0 && token == "}" {
			return makeObjectValue(fields), nil
		}
		if token[0] != '"' {
			return Value{}, fmt.Errorf("expected string key, got %q", token)
		}
		key, err := Unquote(token)
		if err != nil {
			return Value{}, fmt.Errorf("invalid key: %q: %w", token, err)
		}

		token, ok = tokens.next()
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
		fields = append(fields, Field{Key: key, Val: val})
	}

	slices.SortFunc(fields, func(a, b Field) int {
		return strings.Compare(a.Key, b.Key)
	})

	return makeObjectValue(fields), nil
}

// Append serializes the Value to JSON and appends it to the buffer.
// Returns the extended buffer.
func (v *Value) Append(buf []byte) []byte {
	switch v.Kind() {
	case Null:
		return append(buf, "null"...)

	case True:
		return append(buf, "true"...)

	case False:
		return append(buf, "false"...)

	case Number:
		return strconv.AppendFloat(buf, v.Float(), 'g', -1, 64)

	case String:
		return strconv.AppendQuote(buf, v.String())

	case Array:
		buf = append(buf, '[')
		array := v.Array()
		for i := range array {
			if i > 0 {
				buf = append(buf, ',')
			}
			buf = array[i].Append(buf)
		}
		return append(buf, ']')

	case Object:
		buf = append(buf, '{')
		fields := v.Object()
		for i := range fields {
			if i > 0 {
				buf = append(buf, ',')
			}
			buf = strconv.AppendQuote(buf, fields[i].Key)
			buf = append(buf, ':')
			buf = fields[i].Val.Append(buf)
		}
		return append(buf, '}')

	default:
		return buf
	}
}

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
