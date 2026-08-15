//go:build !amd64 && !arm64

package jsonlite

import "hash/maphash"

// hashKey falls back to maphash on architectures where the unaligned loads
// hash_fast.go relies on are not guaranteed to be permitted or cheap.
//
// The tag is only a filter, so the two implementations need not agree: nothing
// outside a single process ever compares them.
func hashKey(k string) byte { return byte(maphash.String(hashseed, k)) }
