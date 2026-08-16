package jsonlite

// Exposed to the external test package so the escape scans can be compared
// against each other and against a plain byte loop.

// Escaped is the dispatching entry point.
func Escaped(s string) bool { return escaped(s) }

// EscapedWide is the vector scan; it requires len(s) >= 64.
func EscapedWide(s string) bool {
	if len(s) < 64 {
		return escaped(s)
	}
	return escapedWide(s)
}

// EscapedHasWide reports whether the vector scan is built in.
func EscapedHasWide() bool { return escapedHasWide() }
