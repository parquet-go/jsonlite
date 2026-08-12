//go:build goexperiment.simd && amd64

package utf8

import (
	"simd/archsimd"
	stdutf8 "unicode/utf8"
	"unsafe"
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

// The three nibble-lookup tables. VPSHUFB looks up within each 128-bit lane,
// so the vector-width constants below replicate them per lane.
var baseTblByte1High = [16]byte{
	tooLong, tooLong, tooLong, tooLong, tooLong, tooLong, tooLong, tooLong,
	twoConts, twoConts, twoConts, twoConts,
	tooShort | overlong2,
	tooShort,
	tooShort | overlong3 | surrogate,
	tooShort | tooLarge | tooLarge1000 | overlong4,
}

var baseTblByte1Low = [16]byte{
	carry | overlong3 | overlong2 | overlong4,
	carry | overlong2,
	carry, carry,
	carry | tooLarge,
	carry | tooLarge | tooLarge1000,
	carry | tooLarge | tooLarge1000,
	carry | tooLarge | tooLarge1000,
	carry | tooLarge | tooLarge1000,
	carry | tooLarge | tooLarge1000,
	carry | tooLarge | tooLarge1000,
	carry | tooLarge | tooLarge1000,
	carry | tooLarge | tooLarge1000,
	carry | tooLarge | tooLarge1000 | surrogate,
	carry | tooLarge | tooLarge1000,
	carry | tooLarge | tooLarge1000,
}

var baseTblByte2High = [16]byte{
	tooShort, tooShort, tooShort, tooShort, tooShort, tooShort, tooShort, tooShort,
	tooLong | overlong2 | twoConts | overlong3 | tooLarge1000 | overlong4,
	tooLong | overlong2 | twoConts | overlong3 | tooLarge,
	tooLong | overlong2 | twoConts | surrogate | tooLarge,
	tooLong | overlong2 | twoConts | surrogate | tooLarge,
	tooShort, tooShort, tooShort, tooShort,
}

var (
	tblByte1High64 = repeatLanes64(baseTblByte1High)
	tblByte1Low64  = repeatLanes64(baseTblByte1Low)
	tblByte2High64 = repeatLanes64(baseTblByte2High)
	tblByte1High32 = repeatLanes32(baseTblByte1High)
	tblByte1Low32  = repeatLanes32(baseTblByte1Low)
	tblByte2High32 = repeatLanes32(baseTblByte2High)
)

func repeatLanes64(t [16]byte) (r [64]byte) {
	for i := range r {
		r[i] = t[i%16]
	}
	return r
}

func repeatLanes32(t [16]byte) (r [32]byte) {
	for i := range r {
		r[i] = t[i%16]
	}
	return r
}

// maxIncomplete flags multibyte sequences truncated at a block boundary:
// bytes greater than these values in the last 3 positions start sequences
// that cannot complete within the block.
var maxIncomplete64 = func() (r [64]byte) {
	for i := range r {
		r[i] = 255
	}
	r[61] = 0xF0 - 1
	r[62] = 0xE0 - 1
	r[63] = 0xC0 - 1
	return r
}()

var maxIncomplete32 = func() (r [32]byte) {
	for i := range r {
		r[i] = 255
	}
	r[29] = 0xF0 - 1
	r[30] = 0xE0 - 1
	r[31] = 0xC0 - 1
	return r
}()

// prevIndices[n-1] holds the VPERMI2B indices computing prev<n>: element i of
// the result selects byte 64-n+i of concat(prevBlock, block).
var prevIndices = func() (r [3][64]byte) {
	for n := 1; n <= 3; n++ {
		for i := range r[n-1] {
			r[n-1][i] = byte(64 - n + i)
		}
	}
	return r
}()

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
	t1hi := archsimd.LoadUint8x64(&tblByte1High64)
	t1lo := archsimd.LoadUint8x64(&tblByte1Low64)
	t2hi := archsimd.LoadUint8x64(&tblByte2High64)
	maxVal := archsimd.LoadUint8x64(&maxIncomplete64)
	indices1 := archsimd.LoadUint8x64(&prevIndices[0])
	indices2 := archsimd.LoadUint8x64(&prevIndices[1])
	indices3 := archsimd.LoadUint8x64(&prevIndices[2])
	lowNibble := archsimd.BroadcastUint8x64(0x0F)
	highBit := archsimd.BroadcastUint8x64(0x80)
	sub2 := archsimd.BroadcastUint8x64(0xE0 - 0x80)
	sub3 := archsimd.BroadcastUint8x64(0xF0 - 0x80)
	zero16x32 := archsimd.BroadcastUint16x32(0)

	i := 0
	// Chunked ASCII fast path: OR-accumulate 512-byte chunks with no
	// per-block compare, movemask, or branch — the scalar stdlib loop beats
	// a naive per-block vector skip precisely because it keeps branches and
	// vector-to-integer crossings off the per-iteration path. All-ASCII
	// chunks are skipped wholesale; chunks containing non-ASCII bytes run
	// the checker on every block unconditionally (the checker is a no-op on
	// ASCII blocks), which keeps the inner loop branch-free too.
	for ; i+512 <= n; i += 512 {
		acc := archsimd.LoadUint8x64((*[64]byte)(buf[i:]))
		acc = acc.Or(archsimd.LoadUint8x64((*[64]byte)(buf[i+64:])))
		acc = acc.Or(archsimd.LoadUint8x64((*[64]byte)(buf[i+128:])))
		acc = acc.Or(archsimd.LoadUint8x64((*[64]byte)(buf[i+192:])))
		acc = acc.Or(archsimd.LoadUint8x64((*[64]byte)(buf[i+256:])))
		acc = acc.Or(archsimd.LoadUint8x64((*[64]byte)(buf[i+320:])))
		acc = acc.Or(archsimd.LoadUint8x64((*[64]byte)(buf[i+384:])))
		acc = acc.Or(archsimd.LoadUint8x64((*[64]byte)(buf[i+448:])))
		if acc.GreaterEqual(highBit).ToBits() == 0 {
			// ASCII chunk: only a pending truncated sequence can be an error.
			errv = errv.Or(prevIncomplete)
			prevIncomplete = zero
			prev = archsimd.LoadUint8x64((*[64]byte)(buf[i+448:]))
			continue
		}
		for k := 0; k < 512; k += 64 {
			z := archsimd.LoadUint8x64((*[64]byte)(buf[i+k:]))
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
	// Remainder blocks after the last full chunk.
	for ; i+64 <= n; i += 64 {
		z := archsimd.LoadUint8x64((*[64]byte)(buf[i:]))
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
// another: (a & laneSelect) | (b &^ laneSelect).
var laneSelect32 = func() (r [32]byte) {
	for i := 0; i < 16; i++ {
		r[i] = 0xFF
	}
	return r
}()

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
	t1hi := archsimd.LoadUint8x32(&tblByte1High32)
	t1lo := archsimd.LoadUint8x32(&tblByte1Low32)
	t2hi := archsimd.LoadUint8x32(&tblByte2High32)
	maxVal := archsimd.LoadUint8x32(&maxIncomplete32)
	laneSwap := archsimd.LoadUint32x8(&laneSwap32)
	laneSel := archsimd.LoadUint8x32(&laneSelect32)
	lowNibble := archsimd.BroadcastUint8x32(0x0F)
	highBit := archsimd.BroadcastUint8x32(0x80)
	sub2 := archsimd.BroadcastUint8x32(0xE0 - 0x80)
	sub3 := archsimd.BroadcastUint8x32(0xF0 - 0x80)
	nibbleMul := archsimd.BroadcastUint16x16(0x1000)

	i := 0
	// Chunked ASCII fast path; see validAVX512 for rationale.
	for ; i+256 <= n; i += 256 {
		acc := archsimd.LoadUint8x32((*[32]byte)(buf[i:]))
		acc = acc.Or(archsimd.LoadUint8x32((*[32]byte)(buf[i+32:])))
		acc = acc.Or(archsimd.LoadUint8x32((*[32]byte)(buf[i+64:])))
		acc = acc.Or(archsimd.LoadUint8x32((*[32]byte)(buf[i+96:])))
		acc = acc.Or(archsimd.LoadUint8x32((*[32]byte)(buf[i+128:])))
		acc = acc.Or(archsimd.LoadUint8x32((*[32]byte)(buf[i+160:])))
		acc = acc.Or(archsimd.LoadUint8x32((*[32]byte)(buf[i+192:])))
		acc = acc.Or(archsimd.LoadUint8x32((*[32]byte)(buf[i+224:])))
		if acc.AsInt8x32().Less(zeroInt).ToBits() == 0 {
			errv = errv.Or(prevIncomplete)
			prevIncomplete = zero
			prev = archsimd.LoadUint8x32((*[32]byte)(buf[i+224:]))
			continue
		}
		for k := 0; k < 256; k += 32 {
			z := archsimd.LoadUint8x32((*[32]byte)(buf[i+k:]))
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
	// Remainder blocks after the last full chunk.
	for ; i+32 <= n; i += 32 {
		z := archsimd.LoadUint8x32((*[32]byte)(buf[i:]))
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
