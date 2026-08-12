// Package utf8 provides fast UTF-8 validation.
//
// On amd64 with GOEXPERIMENT=simd and AVX-512 support, Valid uses a
// vectorized validator based on the lookup algorithm from Keiser & Lemire,
// "Validating UTF-8 In Less Than One Instruction Per Byte"
// (https://arxiv.org/abs/2010.03090), as implemented in simdjson.
// Other platforms fall back to unicode/utf8.
package utf8

import stdutf8 "unicode/utf8"

// Valid reports whether s consists entirely of valid UTF-8-encoded runes.
// It is equivalent to unicode/utf8.ValidString.
func Valid(s string) bool { return valid(s) }

var valid = stdutf8.ValidString
