//go:build !(goexperiment.simd && amd64)

package utf8

import stdutf8 "unicode/utf8"

func valid(s string) bool { return stdutf8.ValidString(s) }
