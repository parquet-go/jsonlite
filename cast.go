package jsonlite

import "unsafe"

// stringBytes exposes the bytes of s without copying. The result must not be
// written to: it aliases the string's backing array, and mutating it would
// break the immutability the language guarantees for string values.
//
// This is the one raw string-to-slice conversion in the package. Everything
// that needs to reinterpret a document as wider words goes through here and
// then unsafecast.Slice, which derives the length and capacity from the type
// sizes rather than open-coding the arithmetic at each call site.
func stringBytes(s string) []byte {
	return unsafe.Slice(unsafe.StringData(s), len(s))
}
