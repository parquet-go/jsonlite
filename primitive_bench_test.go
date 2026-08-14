package jsonlite

import "testing"

// BenchmarkParsePrimitiveRoot measures parsing documents whose root is a
// primitive value, as happens when JSON fragments are written into
// individual typed columns.
func BenchmarkParsePrimitiveRoot(b *testing.B) {
	for _, input := range []struct{ name, data string }{
		{"number", `42`},
		{"float", `95.5`},
		{"bool", `true`},
		{"string", `"user-name-with-some-length"`},
	} {
		b.Run(input.name, func(b *testing.B) {
			b.SetBytes(int64(len(input.data)))
			for b.Loop() {
				if _, err := Parse(input.data); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
