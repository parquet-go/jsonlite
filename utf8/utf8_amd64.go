//go:build goexperiment.simd && amd64

package utf8

import (
	"simd/archsimd"
	stdutf8 "unicode/utf8"
	"unsafe"

	"github.com/parquet-go/bitpack/unsafecast"
)

// valid dispatches on CPU support at each call. The feature checks are
// cheap branches on package variables, and will be erased by dead-code
// elimination under GOAMD64=v3/v4 once https://go.dev/cl/813420 lands.
func valid(s string) bool {
	switch {
	// The AVX-512 validator needs VBMI for the cross-lane byte permute
	// (VPERMI2B) used to compute the previous-byte vectors, and VBMI2 for
	// the immediate-form funnel shift (VPSHRDW).
	case archsimd.X86.AVX512() && archsimd.X86.AVX512VBMI() && archsimd.X86.AVX512VBMI2():
		return validAVX512(s)
	case archsimd.X86.AVX2():
		return validAVX2(s)
	default:
		return stdutf8.ValidString(s)
	}
}

// Error classification bits from the simdjson lookup4 algorithm. A byte pair
// (prev1, input) is invalid iff the intersection of the three table lookups
// is nonzero (after accounting for legitimate 3/4-byte continuations).
const (
	tooShort     = 1 << 0
	tooLong      = 1 << 1
	overlong3    = 1 << 2
	tooLarge     = 1 << 3
	surrogate    = 1 << 4
	overlong2    = 1 << 5
	tooLarge1000 = 1 << 6
	overlong4    = 1 << 6
	twoConts     = 1 << 7
	carry        = tooShort | tooLong | twoConts
)

// The three nibble-lookup tables of the lookup4 algorithm, encoding the
// error classification bits defined above (e.g. 2 = tooLong, 128 = twoConts,
// 131 = carry). The 16-entry table is repeated per 128-bit lane for VPSHUFB;
// the AVX2 kernel loads the first two lanes.
var tblByte1High = [64]byte{
	2, 2, 2, 2, 2, 2, 2, 2, 128, 128, 128, 128, 33, 1, 21, 73,
	2, 2, 2, 2, 2, 2, 2, 2, 128, 128, 128, 128, 33, 1, 21, 73,
	2, 2, 2, 2, 2, 2, 2, 2, 128, 128, 128, 128, 33, 1, 21, 73,
	2, 2, 2, 2, 2, 2, 2, 2, 128, 128, 128, 128, 33, 1, 21, 73,
}

var tblByte1Low = [64]byte{
	231, 163, 131, 131, 139, 203, 203, 203, 203, 203, 203, 203, 203, 219, 203, 203,
	231, 163, 131, 131, 139, 203, 203, 203, 203, 203, 203, 203, 203, 219, 203, 203,
	231, 163, 131, 131, 139, 203, 203, 203, 203, 203, 203, 203, 203, 219, 203, 203,
	231, 163, 131, 131, 139, 203, 203, 203, 203, 203, 203, 203, 203, 219, 203, 203,
}

var tblByte2High = [64]byte{
	1, 1, 1, 1, 1, 1, 1, 1, 230, 174, 186, 186, 1, 1, 1, 1,
	1, 1, 1, 1, 1, 1, 1, 1, 230, 174, 186, 186, 1, 1, 1, 1,
	1, 1, 1, 1, 1, 1, 1, 1, 230, 174, 186, 186, 1, 1, 1, 1,
	1, 1, 1, 1, 1, 1, 1, 1, 230, 174, 186, 186, 1, 1, 1, 1,
}

// maxIncomplete flags multibyte sequences truncated at a block boundary:
// bytes greater than these values in the last 3 positions start sequences
// that cannot complete within the block. The AVX2 kernel loads the last 32
// bytes.
var maxIncomplete = [64]byte{
	255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255,
	255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255,
	255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255,
	255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 0xF0 - 1, 0xE0 - 1, 0xC0 - 1,
}

// prevIndices[n-1] holds the VPERMI2B indices computing prev<n>: element i of
// the result selects byte 64-n+i of concat(prevBlock, block).
var prevIndices = [3][64]byte{
	{
		63, 64, 65, 66, 67, 68, 69, 70, 71, 72, 73, 74, 75, 76, 77, 78,
		79, 80, 81, 82, 83, 84, 85, 86, 87, 88, 89, 90, 91, 92, 93, 94,
		95, 96, 97, 98, 99, 100, 101, 102, 103, 104, 105, 106, 107, 108, 109, 110,
		111, 112, 113, 114, 115, 116, 117, 118, 119, 120, 121, 122, 123, 124, 125, 126,
	},
	{
		62, 63, 64, 65, 66, 67, 68, 69, 70, 71, 72, 73, 74, 75, 76, 77,
		78, 79, 80, 81, 82, 83, 84, 85, 86, 87, 88, 89, 90, 91, 92, 93,
		94, 95, 96, 97, 98, 99, 100, 101, 102, 103, 104, 105, 106, 107, 108, 109,
		110, 111, 112, 113, 114, 115, 116, 117, 118, 119, 120, 121, 122, 123, 124, 125,
	},
	{
		61, 62, 63, 64, 65, 66, 67, 68, 69, 70, 71, 72, 73, 74, 75, 76,
		77, 78, 79, 80, 81, 82, 83, 84, 85, 86, 87, 88, 89, 90, 91, 92,
		93, 94, 95, 96, 97, 98, 99, 100, 101, 102, 103, 104, 105, 106, 107, 108,
		109, 110, 111, 112, 113, 114, 115, 116, 117, 118, 119, 120, 121, 122, 123, 124,
	},
}

func validAVX512(s string) bool {
	n := len(s)
	if n < 64 {
		return stdutf8.ValidString(s)
	}
	buf := unsafe.Slice(unsafe.StringData(s), n)

	zero := archsimd.BroadcastUint8x64(0)
	errv := zero
	prev := zero
	prevIncomplete := zero
	t1hi := archsimd.LoadUint8x64(&tblByte1High)
	t1lo := archsimd.LoadUint8x64(&tblByte1Low)
	t2hi := archsimd.LoadUint8x64(&tblByte2High)
	maxVal := archsimd.LoadUint8x64(&maxIncomplete)
	indices1 := archsimd.LoadUint8x64(&prevIndices[0])
	indices2 := archsimd.LoadUint8x64(&prevIndices[1])
	indices3 := archsimd.LoadUint8x64(&prevIndices[2])
	lowNibble := archsimd.BroadcastUint8x64(0x0F)
	highBit := archsimd.BroadcastUint8x64(0x80)
	sub2 := archsimd.BroadcastUint8x64(0xE0 - 0x80)
	sub3 := archsimd.BroadcastUint8x64(0xF0 - 0x80)
	zero16x32 := archsimd.BroadcastUint16x32(0)

	var i int
	// Chunked ASCII fast path: OR-accumulate 512-byte chunks with no
	// per-block compare, movemask, or branch — the scalar stdlib loop beats
	// a naive per-block vector skip precisely because it keeps branches and
	// vector-to-integer crossings off the per-iteration path. All-ASCII
	// chunks are skipped wholesale; chunks containing non-ASCII bytes run
	// the checker on every block unconditionally (the checker is a no-op on
	// ASCII blocks), which keeps the inner loop branch-free too.
	// Viewing the input as [8][64]byte chunks gives every load a statically
	// bounded index, eliminating the per-load slice bounds checks.
	chunks := unsafecast.Slice[[8][64]byte](buf)
	for ci := range chunks {
		c := &chunks[ci]
		acc := archsimd.LoadUint8x64(&c[0])
		acc = acc.Or(archsimd.LoadUint8x64(&c[1]))
		acc = acc.Or(archsimd.LoadUint8x64(&c[2]))
		acc = acc.Or(archsimd.LoadUint8x64(&c[3]))
		acc = acc.Or(archsimd.LoadUint8x64(&c[4]))
		acc = acc.Or(archsimd.LoadUint8x64(&c[5]))
		acc = acc.Or(archsimd.LoadUint8x64(&c[6]))
		acc = acc.Or(archsimd.LoadUint8x64(&c[7]))
		if acc.GreaterEqual(highBit).ToBits() == 0 {
			// ASCII chunk: only a pending truncated sequence can be an error.
			errv = errv.Or(prevIncomplete)
			prevIncomplete = zero
			prev = archsimd.LoadUint8x64(&c[7])
			continue
		}
		for k := range c {
			z := archsimd.LoadUint8x64(&c[k])
			prev1 := prev.ConcatPermute(z, indices1)
			hi1 := prev1.AsUint16x32().ShiftAllRightConcat(4, zero16x32).AsUint8x64().And(lowNibble)
			lo1 := prev1.And(lowNibble)
			hi2 := z.AsUint16x32().ShiftAllRightConcat(4, zero16x32).AsUint8x64().And(lowNibble)
			sc := t1hi.PermuteOrZeroGrouped(hi1.AsInt8x64()).
				And(t1lo.PermuteOrZeroGrouped(lo1.AsInt8x64())).
				And(t2hi.PermuteOrZeroGrouped(hi2.AsInt8x64()))
			prev2 := prev.ConcatPermute(z, indices2)
			prev3 := prev.ConcatPermute(z, indices3)
			must23 := prev2.SubSaturated(sub2).Or(prev3.SubSaturated(sub3))
			errv = errv.Or(must23.And(highBit).Xor(sc))
			prevIncomplete = z.SubSaturated(maxVal)
			prev = z
		}
	}
	i = len(chunks) * 512
	// Remainder blocks after the last full chunk, as a [64]byte view for
	// the same bounds-check elimination as the chunk loop.
	rest := unsafecast.Slice[[64]byte](buf[i:])
	for ri := range rest {
		z := archsimd.LoadUint8x64(&rest[ri])
		if z.GreaterEqual(highBit).ToBits() == 0 {
			errv = errv.Or(prevIncomplete)
			prevIncomplete = zero
			prev = z
			continue
		}
		prev1 := prev.ConcatPermute(z, indices1)
		hi1 := prev1.AsUint16x32().ShiftAllRightConcat(4, zero16x32).AsUint8x64().And(lowNibble)
		lo1 := prev1.And(lowNibble)
		hi2 := z.AsUint16x32().ShiftAllRightConcat(4, zero16x32).AsUint8x64().And(lowNibble)
		sc := t1hi.PermuteOrZeroGrouped(hi1.AsInt8x64()).
			And(t1lo.PermuteOrZeroGrouped(lo1.AsInt8x64())).
			And(t2hi.PermuteOrZeroGrouped(hi2.AsInt8x64()))
		prev2 := prev.ConcatPermute(z, indices2)
		prev3 := prev.ConcatPermute(z, indices3)
		must23 := prev2.SubSaturated(sub2).Or(prev3.SubSaturated(sub3))
		errv = errv.Or(must23.And(highBit).Xor(sc))
		prevIncomplete = z.SubSaturated(maxVal)
		prev = z
	}
	i += len(rest) * 64

	if errv.NotEqual(zero).ToBits() != 0 {
		return false
	}

	// Scalar tail. If the tail does not begin with a continuation byte, no
	// sequence straddles the boundary, so any pending incomplete sequence
	// from the vector region is an error. Otherwise back up to the sequence
	// lead so the straddling sequence is re-validated in full.
	if i == n || s[i]&0xC0 != 0x80 {
		if prevIncomplete.NotEqual(zero).ToBits() != 0 {
			return false
		}
		return stdutf8.ValidString(s[i:])
	}
	j := i
	for k := 0; k < 3 && j > 0 && s[j]&0xC0 == 0x80; k++ {
		j--
	}
	return stdutf8.ValidString(s[j:])
}

// laneSwap32 swaps the two 128-bit lanes of a 256-bit vector when used as
// VPERMD indices.
var laneSwap32 = [8]uint32{4, 5, 6, 7, 0, 1, 2, 3}

// laneSelect32 selects the low lane from one vector and the high lane from
// another: (a & laneSelect) | (b &^ laneSelect). Only the low lane is set;
// the high lane is zero.
var laneSelect32 = [32]byte{
	0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF,
	0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF,
}

// validAVX2 is the 256-bit validator for CPUs without AVX-512. Differences
// from validAVX512:
//
//   - prev<n> is computed with the haswell-kernel trick: a VPERMD lane swap
//     builds w = [prev.high, block.low], then a per-lane VPALIGNR of
//     (block, w) shifts in the previous bytes.
//   - The high nibble is extracted with VPMULHUW by 0x1000 (x*0x1000>>16 ==
//     x>>4 per 16-bit lane): AVX2 has no immediate-form funnel shift, and
//     the variable-count shift materializes its count through a legacy SSE
//     MOVQ, which stalls when mixed with VEX code on some cores.
func validAVX2(s string) bool {
	n := len(s)
	if n < 32 {
		return stdutf8.ValidString(s)
	}
	buf := unsafe.Slice(unsafe.StringData(s), n)

	zero := archsimd.BroadcastUint8x32(0)
	zeroInt := archsimd.BroadcastInt8x32(0)
	errv := zero
	prev := zero
	prevIncomplete := zero
	t1hi := archsimd.LoadUint8x32((*[32]byte)(tblByte1High[0:32]))
	t1lo := archsimd.LoadUint8x32((*[32]byte)(tblByte1Low[0:32]))
	t2hi := archsimd.LoadUint8x32((*[32]byte)(tblByte2High[0:32]))
	maxVal := archsimd.LoadUint8x32((*[32]byte)(maxIncomplete[32:64]))
	laneSwap := archsimd.LoadUint32x8(&laneSwap32)
	laneSel := archsimd.LoadUint8x32(&laneSelect32)
	lowNibble := archsimd.BroadcastUint8x32(0x0F)
	highBit := archsimd.BroadcastUint8x32(0x80)
	sub2 := archsimd.BroadcastUint8x32(0xE0 - 0x80)
	sub3 := archsimd.BroadcastUint8x32(0xF0 - 0x80)
	nibbleMul := archsimd.BroadcastUint16x16(0x1000)

	var i int
	// Chunked ASCII fast path; see validAVX512 for rationale.
	// As in validAVX512, the [8][32]byte view eliminates per-load bounds
	// checks.
	chunks := unsafecast.Slice[[8][32]byte](buf)
	for ci := range chunks {
		c := &chunks[ci]
		acc := archsimd.LoadUint8x32(&c[0])
		acc = acc.Or(archsimd.LoadUint8x32(&c[1]))
		acc = acc.Or(archsimd.LoadUint8x32(&c[2]))
		acc = acc.Or(archsimd.LoadUint8x32(&c[3]))
		acc = acc.Or(archsimd.LoadUint8x32(&c[4]))
		acc = acc.Or(archsimd.LoadUint8x32(&c[5]))
		acc = acc.Or(archsimd.LoadUint8x32(&c[6]))
		acc = acc.Or(archsimd.LoadUint8x32(&c[7]))
		if acc.AsInt8x32().Less(zeroInt).ToBits() == 0 {
			errv = errv.Or(prevIncomplete)
			prevIncomplete = zero
			prev = archsimd.LoadUint8x32(&c[7])
			continue
		}
		for k := range c {
			z := archsimd.LoadUint8x32(&c[k])
			pSwap := prev.AsUint32x8().Permute(laneSwap).AsUint8x32()
			zSwap := z.AsUint32x8().Permute(laneSwap).AsUint8x32()
			w := pSwap.And(laneSel).Or(zSwap.AndNot(laneSel))
			prev1 := z.ConcatShiftBytesRightGrouped(15, w)
			hi1 := prev1.AsUint16x16().MulHigh(nibbleMul).AsUint8x32().And(lowNibble)
			lo1 := prev1.And(lowNibble)
			hi2 := z.AsUint16x16().MulHigh(nibbleMul).AsUint8x32().And(lowNibble)
			sc := t1hi.PermuteOrZeroGrouped(hi1.AsInt8x32()).
				And(t1lo.PermuteOrZeroGrouped(lo1.AsInt8x32())).
				And(t2hi.PermuteOrZeroGrouped(hi2.AsInt8x32()))
			prev2 := z.ConcatShiftBytesRightGrouped(14, w)
			prev3 := z.ConcatShiftBytesRightGrouped(13, w)
			must23 := prev2.SubSaturated(sub2).Or(prev3.SubSaturated(sub3))
			errv = errv.Or(must23.And(highBit).Xor(sc))
			prevIncomplete = z.SubSaturated(maxVal)
			prev = z
		}
	}
	i = len(chunks) * 256
	// Remainder blocks after the last full chunk; see validAVX512.
	rest := unsafecast.Slice[[32]byte](buf[i:])
	for ri := range rest {
		z := archsimd.LoadUint8x32(&rest[ri])
		// High bit set anywhere means non-ASCII (signed less-than-zero).
		if z.AsInt8x32().Less(zeroInt).ToBits() == 0 {
			errv = errv.Or(prevIncomplete)
			prevIncomplete = zero
			prev = z
			continue
		}
		pSwap := prev.AsUint32x8().Permute(laneSwap).AsUint8x32()
		zSwap := z.AsUint32x8().Permute(laneSwap).AsUint8x32()
		w := pSwap.And(laneSel).Or(zSwap.AndNot(laneSel))
		prev1 := z.ConcatShiftBytesRightGrouped(15, w)
		hi1 := prev1.AsUint16x16().MulHigh(nibbleMul).AsUint8x32().And(lowNibble)
		lo1 := prev1.And(lowNibble)
		hi2 := z.AsUint16x16().MulHigh(nibbleMul).AsUint8x32().And(lowNibble)
		sc := t1hi.PermuteOrZeroGrouped(hi1.AsInt8x32()).
			And(t1lo.PermuteOrZeroGrouped(lo1.AsInt8x32())).
			And(t2hi.PermuteOrZeroGrouped(hi2.AsInt8x32()))
		prev2 := z.ConcatShiftBytesRightGrouped(14, w)
		prev3 := z.ConcatShiftBytesRightGrouped(13, w)
		must23 := prev2.SubSaturated(sub2).Or(prev3.SubSaturated(sub3))
		errv = errv.Or(must23.And(highBit).Xor(sc))
		prevIncomplete = z.SubSaturated(maxVal)
		prev = z
	}
	i += len(rest) * 32

	if errv.NotEqual(zero).ToBits() != 0 {
		return false
	}

	// Scalar tail, as in validAVX512.
	if i == n || s[i]&0xC0 != 0x80 {
		if prevIncomplete.NotEqual(zero).ToBits() != 0 {
			return false
		}
		return stdutf8.ValidString(s[i:])
	}
	j := i
	for k := 0; k < 3 && j > 0 && s[j]&0xC0 == 0x80; k++ {
		j--
	}
	return stdutf8.ValidString(s[j:])
}
