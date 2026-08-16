package jsonlite

// Parsing hands three kinds of slice off to the values it returns: the byte
// tags of an object's hash index, the []Value of a completed array, and the
// []field of a completed object. Each used to be its own allocation, which on
// container-dense documents dominates the parse -- an 826KB document of
// coordinate pairs costs one allocation per [lat,lon] array and spends most of
// its time in the collector rather than the parser.
//
// They are bump-allocated from per-parse blocks instead. A block is handed off
// to the values that alias it and never reused, so the parser releases its
// reference at the end of the parse rather than recycling it.
//
// Blocks grow geometrically between a floor and a ceiling. A fixed size cannot
// serve both ends: sized for container-dense documents it wastes most of a
// block on a small one, and sized for small documents it stops amortizing.
// The ceiling bounds how much a single retained value can pin, since a block
// stays alive as long as any slice carved from it.
const (
	tagBlockMin, tagBlockMax = 64, 4096
	// Values and fields are 16 and 32 bytes, so these ceilings cap a block at
	// 8KB and 16KB respectively.
	valueBlockMin, valueBlockMax = 8, 512
	fieldBlockMin, fieldBlockMax = 8, 512
)

// arenaAlloc returns n elements from block, starting a new block when the
// current one cannot fit them. Requests larger than a full block get an
// exact-size block of their own.
//
// The returned slice is full (len == cap), so a caller appending to it cannot
// reach into the next allocation's elements.
func arenaAlloc[T any](block *[]T, n, minSize, maxSize int) []T {
	b := *block
	if n > cap(b)-len(b) {
		// The first block of a parse is sized exactly, so a document holding a
		// single small container allocates no more than it would have without
		// an arena. Blocks only start growing once a parse has shown it has
		// more than one container to place, which is when amortizing pays.
		grown := n
		if cap(b) != 0 {
			grown = min(max(cap(b)*2, minSize), maxSize)
		}
		b = make([]T, 0, max(n, grown))
	}
	off := len(b)
	b = b[:off+n]
	*block = b
	return b[off : off+n : off+n]
}

// allocTags returns n bytes of storage for an object's hash index. The bytes
// are never written again once filled, which is what makes it safe to alias
// the result as an immutable string.
func (p *parser) allocTags(n int) []byte {
	return arenaAlloc(&p.tags, n, tagBlockMin, tagBlockMax)
}

// allocValues returns storage for a completed array's elements.
func (p *parser) allocValues(n int) []Value {
	return arenaAlloc(&p.valueBlock, n, valueBlockMin, valueBlockMax)
}

// allocFields returns storage for a completed object's fields.
func (p *parser) allocFields(n int) []field {
	return arenaAlloc(&p.fieldBlock, n, fieldBlockMin, fieldBlockMax)
}
