//go:build !(goexperiment.simd && amd64 && !go1.27)

package jsonlite

// escapedWide is never reached without the vector implementation; see
// escaped_amd64.go.
func escapedWide(s string) bool { return escaped(s) }

// escapedHasWide reports whether the vector scan is available.
func escapedHasWide() bool { return false }
