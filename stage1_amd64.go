//go:build goexperiment.simd && amd64

package jsonlite

import (
	"simd/archsimd"
	"unsafe"

	"github.com/parquet-go/bitpack/unsafecast"
)

// simdStage1 reports whether the vectorized structural indexer is available.
// The feature check is a cheap branch on a package variable, and will be
// erased by dead-code elimination under GOAMD64=v4 once
// https://go.dev/cl/813420 lands.
func simdStage1() bool { return archsimd.X86.AVX512() }

// structuralIndex scans s and appends emitted positions to index, returning
// the index, document-level flags, and any string-level validation error.
func structuralIndex(s string, index []uint32) ([]uint32, stage1Flags, error) {
	if archsimd.X86.AVX512() {
		return structuralIndexAVX512(s, index)
	}
	return structuralIndexPortable(s, index)
}

// structuralIndexAVX512 is the vectorized structural indexer. The whole block
// loop lives in one function so the compiler hoists the broadcast constants
// out of the loop and keeps vectors in registers.
//
// Codegen notes, measured on Sapphire Rapids: keep this loop free of
// constructs that emit legacy (non-VEX) SSE instructions — variable-count
// vector shifts and per-block function calls that zero structs both do — as
// each legacy SSE instruction executed with dirty ZMM upper state triggers a
// microcode assist costing ~100ns. See the utf8 subpackage for the same
// constraint.
func structuralIndexAVX512(s string, index []uint32) ([]uint32, stage1Flags, error) {
	var st stage1State
	st.prevSep = 1

	backslash := archsimd.BroadcastUint8x64('\\')
	quote := archsimd.BroadcastUint8x64('"')
	space := archsimd.BroadcastUint8x64(' ')
	tab := archsimd.BroadcastUint8x64('\t')
	newline := archsimd.BroadcastUint8x64('\n')
	carriage := archsimd.BroadcastUint8x64('\r')
	lbrace := archsimd.BroadcastUint8x64('{')
	rbrace := archsimd.BroadcastUint8x64('}')
	lbracket := archsimd.BroadcastUint8x64('[')
	rbracket := archsimd.BroadcastUint8x64(']')
	comma := archsimd.BroadcastUint8x64(',')
	colon := archsimd.BroadcastUint8x64(':')
	ctrl := archsimd.BroadcastUint8x64(0x20)
	high := archsimd.BroadcastUint8x64(0x80)

	buf := unsafe.Slice(unsafe.StringData(s), len(s))
	// Ranging over a [64]byte view gives every load a statically bounded
	// index, eliminating the per-block slice bounds check.
	blocks := unsafecast.Slice[[64]byte](buf)
	for bi := range blocks {
		v := archsimd.LoadUint8x64(&blocks[bi])
		var m blockMasks
		m.bs = v.Equal(backslash).ToBits()
		m.quote = v.Equal(quote).ToBits()
		m.ctrl = v.Less(ctrl).ToBits()
		m.hi = v.GreaterEqual(high).ToBits()
		m.ws = v.Equal(space).ToBits() | v.Equal(tab).ToBits() |
			v.Equal(newline).ToBits() | v.Equal(carriage).ToBits()
		m.structural = v.Equal(lbrace).ToBits() | v.Equal(rbrace).ToBits() |
			v.Equal(lbracket).ToBits() | v.Equal(rbracket).ToBits() |
			v.Equal(comma).ToBits() | v.Equal(colon).ToBits()
		index = st.crunch(m, bi*64, index)
	}
	i := len(blocks) * 64
	if i < len(s) {
		var b [64]byte
		for j := range b {
			b[j] = ' '
		}
		copy(b[:], s[i:])
		index = st.crunch(classifyBlockPortable(&b), i, index)
	}
	if st.err != nil {
		return index, st.flags(), st.err
	}
	if st.prevInString != 0 {
		return index, st.flags(), errUnterminatedString
	}
	return index, st.flags(), nil
}
