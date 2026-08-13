// Package utf8 provides fast UTF-8 validation.
//
// On amd64 with GOEXPERIMENT=simd, Valid uses a vectorized validator based
// on the lookup algorithm from Keiser & Lemire, "Validating UTF-8 In Less
// Than One Instruction Per Byte" (https://arxiv.org/abs/2010.03090), as
// implemented in simdjson, selecting an AVX-512 or AVX2 kernel at startup
// based on CPU support. Other builds fall back to unicode/utf8.
package utf8

// Valid reports whether s consists entirely of valid UTF-8-encoded runes.
// It is equivalent to unicode/utf8.ValidString.
func Valid(s string) bool { return valid(s) }
