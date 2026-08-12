//go:build goexperiment.simd && amd64

package utf8

import "testing"

// TestTableInvariants verifies the structural properties the kernels rely on:
// lookup tables repeat per 128-bit lane (the AVX2 kernel loads the first two
// lanes), maxIncomplete's last 32 bytes are its 256-bit variant, and
// prevIndices computes the prev<n> byte selections.
func TestTableInvariants(t *testing.T) {
	for name, tbl := range map[string]*[64]byte{
		"tblByte1High": &tblByte1High,
		"tblByte1Low":  &tblByte1Low,
		"tblByte2High": &tblByte2High,
	} {
		for i := 16; i < 64; i++ {
			if tbl[i] != tbl[i%16] {
				t.Errorf("%s[%d] = %d, want %d (lane repeat)", name, i, tbl[i], tbl[i%16])
			}
		}
	}
	for i := 0; i < 61; i++ {
		if maxIncomplete[i] != 255 {
			t.Errorf("maxIncomplete[%d] = %d, want 255", i, maxIncomplete[i])
		}
	}
	if maxIncomplete[61] != 0xF0-1 || maxIncomplete[62] != 0xE0-1 || maxIncomplete[63] != 0xC0-1 {
		t.Errorf("maxIncomplete tail = %v", maxIncomplete[61:])
	}
	for n := 1; n <= 3; n++ {
		for i := 0; i < 64; i++ {
			if int(prevIndices[n-1][i]) != 64-n+i {
				t.Errorf("prevIndices[%d][%d] = %d, want %d", n-1, i, prevIndices[n-1][i], 64-n+i)
			}
		}
	}
}
