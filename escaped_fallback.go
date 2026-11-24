//go:build !(goexperiment.simd && amd64)

package jsonlite

// escapedWide is never reached without the vector implementation; see
// escaped_amd64.go (Go 1.26) and escaped_amd64_go127.go (Go 1.27+).
func escapedWide(s string) bool { return escaped(s) }

// escapedHasWide reports whether the vector scan is available.
func escapedHasWide() bool { return false }
