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
// remain is the number of input bytes still unparsed after the container
// being placed. See arenaBlockSize for how it bounds the new block.
//
// The returned slice is full (len == cap), so a caller appending to it cannot
// reach into the next allocation's elements.
func arenaAlloc[T any](block *[]T, n, minSize, maxSize, remain int) []T {
	b := *block
	if n > cap(b)-len(b) {
		// Same decision as arenaBlockSize, written out so this stays under
		// the inliner budget and can be inlined into the noinline wrappers
		// below. A call here would be a second call per container.
		grown := n
		if cap(b) != 0 {
			grown = min(max(cap(b)*2, minSize), maxSize)
			if remain < 2 {
				grown = n
			}
		}
		b = make([]T, 0, max(n, grown))
	}
	off := len(b)
	b = b[:off+n]
	*block = b
	return b[off : off+n : off+n]
}

// arenaBlockSize is the capacity of a fresh block serving a request for n
// elements. prev is the capacity of the block that just ran out (zero when
// this is the first block of the parse).
//
// The first block is sized exactly, so a document holding a single small
// container allocates no more than it would have without an arena. Later
// blocks grow geometrically between minSize and maxSize, which is what
// amortizes a long run of small containers. Growth is skipped when fewer
// than two bytes of input remain: that is not enough to start another value
// (a single digit is one byte, a closing bracket is one), so the block is
// the request itself. Without that, a nested object that exactly fills the
// first block -- allocFields(7+1) for a 7-field http object -- forces its
// parent into a doubled block, 16 slots for a 5-slot object, 24 field slots
// where 13 suffice.
func arenaBlockSize(n, prev, minSize, maxSize, remain int) int {
	if prev <= 0 {
		return n
	}
	grown := min(max(prev*2, minSize), maxSize)
	if remain < 2 || grown < n {
		return n
	}
	return grown
}

// The alloc methods are marked noinline. Inlining arenaAlloc into
// parseArray and parseObject makes those functions large enough that the
// element loop loses registers, which costs more than the call on a document
// of many small containers. The call is one per container, not one per element.
//
//go:noinline
func (p *parser) allocTags(n, remain int) []byte {
	return arenaAlloc(&p.tags, n, tagBlockMin, tagBlockMax, remain)
}

//go:noinline
func (p *parser) allocValues(n, remain int) []Value {
	return arenaAlloc(&p.valueBlock, n, valueBlockMin, valueBlockMax, remain)
}

//go:noinline
func (p *parser) allocFields(n, remain int) []field {
	return arenaAlloc(&p.fieldBlock, n, fieldBlockMin, fieldBlockMax, remain)
}
