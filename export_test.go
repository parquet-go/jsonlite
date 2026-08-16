package jsonlite

// EscapedForTest exposes the escape scan to the external test package.
func EscapedForTest(s string) bool { return escaped(s) }
