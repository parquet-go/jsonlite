package jsonlite

import (
	"encoding/base64"
	"iter"
	"strconv"
	"time"
)

// AppendFunc is a function that appends a value of type T to a byte slice.
type AppendFunc[T any] func([]byte, T) []byte

// AppendArray appends a JSON array to b by iterating over seq and using fn
// to serialize each element.
func AppendArray[T any](b []byte, seq iter.Seq[T], fn AppendFunc[T]) []byte {
	b = append(b, '[')
	i := 0
	for elem := range seq {
		if i > 0 {
			b = append(b, ',')
		}
		b = fn(b, elem)
		i++
	}
	return append(b, ']')
}

// AppendObject appends a JSON object to b by iterating over seq and using fn
// to serialize each value. Keys are automatically quoted.
func AppendObject[T any](b []byte, seq iter.Seq2[string, T], fn AppendFunc[T]) []byte {
	b = append(b, '{')
	i := 0
	for key, value := range seq {
		if i > 0 {
			b = append(b, ',')
		}
		b = AppendQuote(b, key)
		b = append(b, ':')
		b = fn(b, value)
		i++
	}
	return append(b, '}')
}

// AppendNull appends a JSON null to b.
func AppendNull(b []byte) []byte { return append(b, "null"...) }

// AppendInt appends an integer as JSON to b.
func AppendInt(b []byte, n int64) []byte { return strconv.AppendInt(b, n, 10) }

// AppendUint appends an unsigned integer as JSON to b.
func AppendUint(b []byte, n uint64) []byte { return strconv.AppendUint(b, n, 10) }

// AppendFloat appends a floating point number as JSON to b.
func AppendFloat(b []byte, f float64) []byte { return strconv.AppendFloat(b, f, 'g', -1, 64) }

// AppendBool appends a boolean as JSON to b.
func AppendBool(b []byte, v bool) []byte { return strconv.AppendBool(b, v) }

// AppendTime appends a time.Time as a JSON quoted RFC3339Nano string to b.
func AppendTime(b []byte, t time.Time) []byte {
	b = append(b, '"')
	b = t.AppendFormat(b, time.RFC3339Nano)
	return append(b, '"')
}

// AppendBytes appends a byte slice as a base64-encoded JSON string to b.
func AppendBytes(b []byte, data []byte) []byte {
	b = append(b, '"')
	b = base64.StdEncoding.AppendEncode(b, data)
	return append(b, '"')
}
