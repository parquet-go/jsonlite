//go:build goexperiment.simd && amd64

package utf8

import (
	"simd/archsimd"
	stdutf8 "unicode/utf8"
	"unsafe"
)

func init() {
	// The validator needs AVX512VBMI for the cross-lane byte permute
	// (VPERMI2B) used to compute the previous-byte vectors.
	if archsimd.X86.AVX512() && archsimd.X86.AVX512VBMI() && archsimd.X86.AVX512VBMI2() {
		valid = validSIMD
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

// The three nibble-lookup tables, replicated per 128-bit lane for VPSHUFB.
var tblByte1High = repeat4([16]byte{
	tooLong, tooLong, tooLong, tooLong, tooLong, tooLong, tooLong, tooLong,
	twoConts, twoConts, twoConts, twoConts,
	tooShort | overlong2,
	tooShort,
	tooShort | overlong3 | surrogate,
	tooShort | tooLarge | tooLarge1000 | overlong4,
})

var tblByte1Low = repeat4([16]byte{
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
})

var tblByte2High = repeat4([16]byte{
	tooShort, tooShort, tooShort, tooShort, tooShort, tooShort, tooShort, tooShort,
	tooLong | overlong2 | twoConts | overlong3 | tooLarge1000 | overlong4,
	tooLong | overlong2 | twoConts | overlong3 | tooLarge,
	tooLong | overlong2 | twoConts | surrogate | tooLarge,
	tooLong | overlong2 | twoConts | surrogate | tooLarge,
	tooShort, tooShort, tooShort, tooShort,
})

func repeat4(t [16]byte) (r [64]byte) {
	for i := range r {
		r[i] = t[i%16]
	}
	return r
}

// maxIncomplete flags multibyte sequences truncated at a block boundary:
// bytes greater than these values in the last 3 positions start sequences
// that cannot complete within the block.
var maxIncomplete = func() (r [64]byte) {
	for i := range r {
		r[i] = 255
	}
	r[61] = 0xF0 - 1
	r[62] = 0xE0 - 1
	r[63] = 0xC0 - 1
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

func validSIMD(s string) bool {
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

	i := 0
	for ; i+64 <= n; i += 64 {
		z := archsimd.LoadUint8x64((*[64]byte)(buf[i:]))
		if z.GreaterEqual(highBit).ToBits() == 0 {
			// ASCII block: only a pending truncated sequence can be an error.
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
