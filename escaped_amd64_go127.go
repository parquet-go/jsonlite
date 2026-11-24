//go:build goexperiment.simd && amd64 && go1.27

package jsonlite

import (
	"simd/archsimd"

	"github.com/parquet-go/bitpack/unsafecast"
)

// escapedWide reports whether s contains a backslash or a raw control
// character, scanning 64 bytes per iteration with AVX2.
//
// The vector form is not just wider than the word-at-a-time scan, it is
// simpler. The SWAR version has to mask out bytes >= 0x80 with `&^ n`,
// because its below() trick misfires on them; an unsigned byte compare gets
// that right by construction, since 0xC3 is not less than 0x20.
//
// This is the Go 1.27 form of escaped_amd64.go. LoadUint8x32 on a
// *[32]uint8 is LoadUint8x32Array; the compares and Mask8x32.ToBits are
// the same operations as on Go 1.26.
//
// Callers must ensure len(s) >= 64.
func escapedWide(s string) bool {
	ctrl := archsimd.BroadcastUint8x32(0x20)
	backslash := archsimd.BroadcastUint8x32('\\')

	blocks := unsafecast.Slice[[64]byte](unsafecast.Bytes(s))
	for bi := range blocks {
		blk := &blocks[bi]
		lo := archsimd.LoadUint8x32Array((*[32]byte)(blk[0:32]))
		hi := archsimd.LoadUint8x32Array((*[32]byte)(blk[32:64]))
		m := lo.Less(ctrl).Or(lo.Equal(backslash)).ToBits() |
			hi.Less(ctrl).Or(hi.Equal(backslash)).ToBits()
		if m != 0 {
			// Callers often copy the string next (AppendUnquote). Leaving the
			// upper YMM bits dirty makes that SSE memmove pay the AVX-SSE
			// transition on every call.
			archsimd.ClearAVXUpperBits()
			return true
		}
	}
	archsimd.ClearAVXUpperBits()
	// The tail is shorter than a block, so this re-enters the word-at-a-time
	// path rather than recursing into the vector one.
	return escaped(s[len(blocks)*64:])
}

// escapedHasWide reports whether the vector scan is available.
func escapedHasWide() bool { return archsimd.X86.AVX2() }
