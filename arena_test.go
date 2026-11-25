package jsonlite

import (
	"runtime"
	"runtime/debug"
	"strings"
	"testing"
	"unsafe"
)

// parquetSmallRow is the "small" document from parquet-go's
// BenchmarkWriteJSONShredded: a 4-field record whose http field is a 7-field
// object. Field storage for that shape is 8 slots (the nested object) plus 5
// (the parent), 13 in all.
const parquetSmallRow = `{"severity":"INFO","message":"request served","trace":"","http":{"method":"GET","url":"/","status":200,"latency_ms":1,"user_agent":"","remote_ip":"","cache_hit":true}}`

// parquetSmallRowHTTPFirst is the same record with the nested object first,
// so the first block is carved while most of the document is still unparsed.
const parquetSmallRowHTTPFirst = `{"http":{"method":"GET","url":"/","status":200,"latency_ms":1,"user_agent":"","remote_ip":"","cache_hit":true},"severity":"INFO","message":"request served","trace":""}`

func TestArenaBlockSize(t *testing.T) {
	t.Run("first block is exact", func(t *testing.T) {
		for _, n := range []int{1, 2, 3, 5, 8, 13, 100, 1000} {
			for _, remain := range []int{0, 1, 1 << 20} {
				got := arenaBlockSize(n, 0, fieldBlockMin, fieldBlockMax, remain)
				if got != n {
					t.Fatalf("arenaBlockSize(%d, 0, remain=%d) = %d, want %d", n, remain, got, n)
				}
			}
		}
	})

	t.Run("exhausted tail sizes the block to the request", func(t *testing.T) {
		// The parquet row: nested object took an exact 8-slot block, parent
		// needs 5, and nothing is left in the input. Fewer than two trailing
		// bytes cannot hold another slot, so the block is exactly the request
		// rather than the doubled 16. Two trailing bytes still grow.
		for _, remain := range []int{-1, 0, 1} {
			got := arenaBlockSize(5, 8, fieldBlockMin, fieldBlockMax, remain)
			if got != 5 {
				t.Fatalf("arenaBlockSize(5, 8, remain=%d) = %d, want 5", remain, got)
			}
			got = arenaBlockSize(2, 8, valueBlockMin, valueBlockMax, remain)
			if got != 2 {
				t.Fatalf("arenaBlockSize(2, 8, remain=%d) = %d, want 2", remain, got)
			}
		}
		if got := arenaBlockSize(5, 8, fieldBlockMin, fieldBlockMax, 2); got != 16 {
			t.Fatalf("arenaBlockSize(5, 8, remain=2) = %d, want 16", got)
		}
		if got := arenaBlockSize(2, 8, valueBlockMin, valueBlockMax, 2); got != 16 {
			t.Fatalf("arenaBlockSize(2, 8, remain=2) = %d, want 16", got)
		}
	})

	t.Run("large tail keeps geometric growth", func(t *testing.T) {
		// Powers of two from the floor, capped at the ceiling. This is the
		// growth a long run of small containers depends on.
		prev := 3
		want := []int{8, 16, 32, 64, 128, 256, 512, 512}
		for _, w := range want {
			got := arenaBlockSize(3, prev, valueBlockMin, valueBlockMax, 1<<20)
			if got != w {
				t.Fatalf("arenaBlockSize(3, prev=%d, large tail) = %d, want %d", prev, got, w)
			}
			prev = got
		}
		prev = 5
		want = []int{10, 20, 40, 80, 160, 320, 512}
		for _, w := range want {
			got := arenaBlockSize(5, prev, fieldBlockMin, fieldBlockMax, 1<<20)
			if got != w {
				t.Fatalf("arenaBlockSize(5, prev=%d, large tail) = %d, want %d", prev, got, w)
			}
			prev = got
		}
		// Tag blocks share the rule, including their higher floor.
		if got := arenaBlockSize(10, 10, tagBlockMin, tagBlockMax, 1<<20); got != tagBlockMin {
			t.Fatalf("tag growth from 10 with a large tail = %d, want %d", got, tagBlockMin)
		}
		if got := arenaBlockSize(10, 10, tagBlockMin, tagBlockMax, 0); got != 10 {
			t.Fatalf("tag growth from 10 with an empty tail = %d, want 10", got)
		}
	})

	t.Run("request larger than a block stays exact", func(t *testing.T) {
		n := valueBlockMax + 50
		if got := arenaBlockSize(n, valueBlockMax, valueBlockMin, valueBlockMax, 1<<20); got != n {
			t.Fatalf("oversized request = %d, want %d", got, n)
		}
		if got := arenaBlockSize(n, 0, valueBlockMin, valueBlockMax, 0); got != n {
			t.Fatalf("oversized first block = %d, want %d", got, n)
		}
		// The block that just filled was itself an exact oversized allocation.
		// A large tail still grows to the ceiling; an empty tail does not.
		if got := arenaBlockSize(3, 10000, valueBlockMin, valueBlockMax, 1<<20); got != valueBlockMax {
			t.Fatalf("growth after an oversized block = %d, want %d", got, valueBlockMax)
		}
		if got := arenaBlockSize(3, 10000, valueBlockMin, valueBlockMax, 0); got != 3 {
			t.Fatalf("empty tail after an oversized block = %d, want 3", got)
		}
	})

	t.Run("never larger than unbounded geometric growth", func(t *testing.T) {
		sizes := [][2]int{
			{tagBlockMin, tagBlockMax},
			{valueBlockMin, valueBlockMax},
			{fieldBlockMin, fieldBlockMax},
		}
		prevs := []int{0, 1, 3, 5, 8, 10, 16, 64, 100, 200, 300, 512, 1000, 4000}
		ns := []int{1, 2, 3, 5, 8, 10, 13, 64, 100, 513, 1000, 9000}
		remains := []int{-1, 0, 1, 2, 3, 10, 40, 100, 1000, 1 << 20}
		for _, mm := range sizes {
			minSize, maxSize := mm[0], mm[1]
			for _, prev := range prevs {
				if prev > maxSize {
					continue
				}
				for _, n := range ns {
					for _, remain := range remains {
						got := arenaBlockSize(n, prev, minSize, maxSize, remain)
						ceiling := geometricBlockSize(n, prev, minSize, maxSize)
						if got > ceiling || got < n {
							t.Fatalf("arenaBlockSize(n=%d, prev=%d, min=%d, max=%d, remain=%d) = %d, ceiling %d",
								n, prev, minSize, maxSize, remain, got, ceiling)
						}
						if remain >= 1<<20 && got != ceiling {
							t.Fatalf("large tail n=%d prev=%d min=%d max=%d: got %d, geometric %d",
								n, prev, minSize, maxSize, got, ceiling)
						}
					}
				}
			}
		}
	})
}

// geometricBlockSize is the previous doubling rule, kept here so the tests
// can state that the tail bound only ever shrinks a block.
func geometricBlockSize(n, prev, minSize, maxSize int) int {
	if prev <= 0 {
		return n
	}
	doubled := prev * 2
	if doubled < minSize {
		doubled = minSize
	}
	if doubled > maxSize {
		doubled = maxSize
	}
	if n > doubled {
		return n
	}
	return doubled
}

func TestAllocMethodsMatchBlockSize(t *testing.T) {
	// allocValues, allocFields and allocTags are specialized copies of
	// arenaAlloc so they inline into the parser. A drift shows up as a
	// different block capacity than arenaBlockSize.
	check := func(t *testing.T, got, n, prev, minSize, maxSize, remain int) {
		t.Helper()
		want := arenaBlockSize(n, prev, minSize, maxSize, remain)
		if got != want {
			t.Fatalf("block cap %d, arenaBlockSize(%d, prev=%d, remain=%d) = %d", got, n, prev, remain, want)
		}
	}
	t.Run("values", func(t *testing.T) {
		p := &parser{}
		prev := 0
		for _, c := range []struct{ n, remain int }{{8, 100}, {3, 1}, {3, 1000}} {
			if c.n <= cap(p.valueBlock)-len(p.valueBlock) {
				t.Fatalf("n=%d remain=%d unexpectedly fit", c.n, c.remain)
			}
			p.allocValues(c.n, c.remain)
			check(t, cap(p.valueBlock), c.n, prev, valueBlockMin, valueBlockMax, c.remain)
			prev = cap(p.valueBlock)
		}
	})
	t.Run("fields", func(t *testing.T) {
		p := &parser{}
		prev := 0
		for _, c := range []struct{ n, remain int }{{8, 1}, {5, 0}, {5, 1000}} {
			if c.n <= cap(p.fieldBlock)-len(p.fieldBlock) {
				t.Fatalf("n=%d remain=%d unexpectedly fit", c.n, c.remain)
			}
			p.allocFields(c.n, c.remain)
			check(t, cap(p.fieldBlock), c.n, prev, fieldBlockMin, fieldBlockMax, c.remain)
			prev = cap(p.fieldBlock)
		}
	})
	t.Run("tags", func(t *testing.T) {
		p := &parser{}
		prev := 0
		for _, c := range []struct{ n, remain int }{{10, 1}, {10, 0}, {10, 1000}} {
			if c.n <= cap(p.tags)-len(p.tags) {
				t.Fatalf("n=%d remain=%d unexpectedly fit", c.n, c.remain)
			}
			p.allocTags(c.n, c.remain)
			check(t, cap(p.tags), c.n, prev, tagBlockMin, tagBlockMax, c.remain)
			prev = cap(p.tags)
		}
	})
}

func TestArenaAllocCarvesFullSlices(t *testing.T) {
	// Parquet row, through the real allocator: an 8-slot block, then a 5-slot
	// block. The first slice must not be able to see the second.
	var fields []field
	first := arenaAlloc(&fields, 8, fieldBlockMin, fieldBlockMax, 1)
	if len(first) != 8 || cap(first) != 8 || cap(fields) != 8 {
		t.Fatalf("first field block len=%d cap=%d blockcap=%d, want 8", len(first), cap(first), cap(fields))
	}
	second := arenaAlloc(&fields, 5, fieldBlockMin, fieldBlockMax, 0)
	if len(second) != 5 || cap(second) != 5 || cap(fields) != 5 {
		t.Fatalf("second field block len=%d cap=%d blockcap=%d, want 5", len(second), cap(second), cap(fields))
	}
	if cap(first) != 8 {
		t.Fatalf("first slice cap changed to %d", cap(first))
	}

	// Same pattern for arrays: 7 elements fill an 8-slot block, the parent
	// array needs 2, and a closing bracket is all that remains.
	var values []Value
	inner := arenaAlloc(&values, 8, valueBlockMin, valueBlockMax, 1)
	outer := arenaAlloc(&values, 2, valueBlockMin, valueBlockMax, 0)
	if cap(inner) != 8 || cap(outer) != 2 {
		t.Fatalf("array blocks cap %d then %d, want 8 then 2", cap(inner), cap(outer))
	}
}

func TestArenaAllocReachesValueCeiling(t *testing.T) {
	var values []Value
	var caps []int
	prev := -1
	for range 5000 {
		arenaAlloc(&values, 3, valueBlockMin, valueBlockMax, 1<<20)
		if cap(values) != prev {
			caps = append(caps, cap(values))
			prev = cap(values)
		}
	}
	want := []int{3, 8, 16, 32, 64, 128, 256, valueBlockMax}
	if len(caps) != len(want) {
		t.Fatalf("block caps %v, want %v", caps, want)
	}
	for i := range want {
		if caps[i] != want[i] {
			t.Fatalf("block caps %v, want %v", caps, want)
		}
	}
}

func TestParseSmallNestedRowBytes(t *testing.T) {
	// 13 field slots plus the returned *Value. Both sizes land on allocator
	// size classes, so the measured total is exact: a doubled second block
	// would be 8+16 slots and 784 bytes instead of 432.
	const (
		nestedSlots = 8 // allocFields(7+1)
		parentSlots = 5 // allocFields(4+1)
	)
	wantSlots := nestedSlots + parentSlots
	wantBytes := int64(wantSlots)*int64(unsafe.Sizeof(field{})) + int64(unsafe.Sizeof(Value{}))
	if wantSlots != 13 || wantBytes != 432 {
		t.Fatalf("test assumption: 13 field slots are %d bytes, got slots=%d bytes=%d", 432, wantSlots, wantBytes)
	}

	docs := []string{parquetSmallRow, parquetSmallRowHTTPFirst}
	parses := []struct {
		name  string
		parse func(string) (*Value, error)
	}{
		{"classic", func(s string) (*Value, error) { return classicParse(s, DefaultMaxDepth) }},
		{"indexed", func(s string) (*Value, error) { return parseIndexed(s, DefaultMaxDepth) }},
	}
	for _, doc := range docs {
		for _, p := range parses {
			t.Run(p.name+"/"+doc[:12], func(t *testing.T) {
				v, err := p.parse(doc)
				if err != nil {
					t.Fatal(err)
				}
				if v.JSON() != doc {
					t.Fatalf("JSON() = %s, want %s", v.JSON(), doc)
				}
				if v.Len() != 4 {
					t.Fatalf("parent fields = %d, want 4", v.Len())
				}
				http := v.Lookup("http")
				if http == nil || http.Kind() != Object || http.Len() != 7 {
					t.Fatalf("http = %v", http)
				}
				bytes, mallocs := measuredAllocs(func() {
					v, err := p.parse(doc)
					if err != nil {
						t.Fatal(err)
					}
					parseSink = v
				})
				if bytes != wantBytes || mallocs != 3 {
					t.Fatalf("parse allocated %d bytes, %d mallocs; want %d bytes, 3 mallocs (13 field slots, not a doubled 16-slot second block)",
						bytes, mallocs, wantBytes)
				}
			})
		}
	}
}

func TestParseNestedArrayBytes(t *testing.T) {
	// Array analogues of the parquet row.
	//
	// nested: the inner array fills an exact 8-slot block and the parent
	// needs 2, with only the closing bracket left. 10 value slots.
	//
	// siblings: two inner arrays (8 slots, then 5) under a parent that needs
	// 3. The second inner array is carved while a single closing bracket
	// remains, so it must not double to 16 and swallow the parent.
	tests := []struct {
		name   string
		doc    string
		slots  int
		malloc int64
	}{
		{"nested", `[[0,0,0,0,0,0,0]]`, 10, 3},
		{"siblings", `[[0,0,0,0,0,0,0],[0,0,0,0]]`, 16, 4},
	}
	parses := []struct {
		name  string
		parse func(string) (*Value, error)
	}{
		{"classic", func(s string) (*Value, error) { return classicParse(s, DefaultMaxDepth) }},
		{"indexed", func(s string) (*Value, error) { return parseIndexed(s, DefaultMaxDepth) }},
	}
	for _, tt := range tests {
		wantBytes := int64(tt.slots)*int64(unsafe.Sizeof(Value{})) + int64(unsafe.Sizeof(Value{}))
		for _, parse := range parses {
			t.Run(tt.name+"/"+parse.name, func(t *testing.T) {
				v, err := parse.parse(tt.doc)
				if err != nil {
					t.Fatal(err)
				}
				if v.JSON() != tt.doc {
					t.Fatalf("JSON() = %s, want %s", v.JSON(), tt.doc)
				}
				bytes, mallocs := measuredAllocs(func() {
					v, err := parse.parse(tt.doc)
					if err != nil {
						t.Fatal(err)
					}
					parseSink = v
				})
				if bytes != wantBytes || mallocs != tt.malloc {
					t.Fatalf("parse allocated %d bytes, %d mallocs; want %d bytes, %d mallocs (%d value slots)",
						bytes, mallocs, wantBytes, tt.malloc, tt.slots)
				}
			})
		}
	}
}

func TestParseSingleObjectStaysExact(t *testing.T) {
	// The first block is exact, so a document with one container does not
	// round up to the block floor. {"id":1,"ok":true} is 3 field slots.
	const doc = `{"id":1,"ok":true}`
	wantBytes := int64(3)*int64(unsafe.Sizeof(field{})) + int64(unsafe.Sizeof(Value{}))
	bytes, mallocs := measuredAllocs(func() {
		v, err := Parse(doc)
		if err != nil {
			t.Fatal(err)
		}
		parseSink = v
	})
	if bytes != wantBytes || mallocs != 2 {
		t.Fatalf("parse allocated %d bytes, %d mallocs; want %d bytes, 2 mallocs", bytes, mallocs, wantBytes)
	}
}

func TestParseDensePairsShareBlocks(t *testing.T) {
	// A few hundred coordinate pairs used to be one allocation each. Growth
	// has to keep reaching the block ceiling while the tail is large, or
	// this climbs back toward one malloc per pair.
	const pairs = 200
	var b strings.Builder
	b.WriteString(`{"type":"Polygon","coordinates":[`)
	for i := range pairs {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(`[1,2]`)
	}
	b.WriteString(`]}`)
	doc := b.String()

	v, err := Parse(doc)
	if err != nil {
		t.Fatal(err)
	}
	if v.Lookup("coordinates").Len() != pairs {
		t.Fatalf("coordinates = %d, want %d", v.Lookup("coordinates").Len(), pairs)
	}
	_, mallocs := measuredAllocs(func() {
		v, err := Parse(doc)
		if err != nil {
			t.Fatal(err)
		}
		parseSink = v
	})
	// One block per pair would be about 200 mallocs. Geometric growth of
	// 3-slot arrays stays under a couple of dozen, including the parent
	// containers and the returned *Value.
	if mallocs >= 40 {
		t.Fatalf("parsing %d pairs made %d mallocs; blocks are not being shared", pairs, mallocs)
	}
}

// measuredAllocs reports bytes and mallocs per call. GC is disabled for the
// measurement so a collection cannot clear the parser pool and add its
// allocations to the count. The round with the fewest bytes is the one
// reported, which drops a stray runtime allocation.
func measuredAllocs(fn func()) (bytes, mallocs int64) {
	old := debug.SetGCPercent(-1)
	defer debug.SetGCPercent(old)
	fn()
	bytes = -1
	for range 5 {
		var before, after runtime.MemStats
		runtime.ReadMemStats(&before)
		const n = 32
		for range n {
			fn()
		}
		runtime.ReadMemStats(&after)
		deltaB := after.TotalAlloc - before.TotalAlloc
		deltaM := after.Mallocs - before.Mallocs
		if deltaB%uint64(n) != 0 || deltaM%uint64(n) != 0 {
			continue
		}
		b := int64(deltaB / uint64(n))
		m := int64(deltaM / uint64(n))
		if bytes < 0 || b < bytes {
			bytes = b
			mallocs = m
		}
	}
	return bytes, mallocs
}

func TestParsePerRowShapes(t *testing.T) {
	// medium and large in BenchmarkParsePerRow are the same field counts as
	// the small row, with longer strings. Strings are views into the input,
	// so they must not add field slots.
	httpJSON := `"http":{"method":"GET","url":"https://example.com/api/v1/items",` +
		`"status":200,"latency_ms":12.5,"user_agent":"Mozilla/5.0 (compatible)",` +
		`"remote_ip":"192.168.1.100","cache_hit":false}`
	docs := []string{
		parquetSmallRow,
		`{"severity":"INFO","message":"` + strings.Repeat("request served ", 8) +
			`","trace":"projects/test-project/traces/1234567890abcdef",` + httpJSON + `}`,
		`{"severity":"WARNING","message":"` + strings.Repeat("upstream latency above threshold ", 24) +
			`","trace":"projects/test-project/traces/1234567890abcdef",` + httpJSON + `}`,
	}
	parses := []struct {
		name  string
		parse func(string) (*Value, error)
	}{
		{"classic", func(s string) (*Value, error) { return classicParse(s, DefaultMaxDepth) }},
		{"indexed", func(s string) (*Value, error) { return parseIndexed(s, DefaultMaxDepth) }},
	}
	for _, parse := range parses {
		t.Run(parse.name, func(t *testing.T) {
			var got []int64
			for i, doc := range docs {
				v, err := parse.parse(doc)
				if err != nil {
					t.Fatal(err)
				}
				if v.Len() != 4 || v.Lookup("http") == nil || v.Lookup("http").Len() != 7 {
					t.Fatalf("doc %d shape", i)
				}
				b, m := measuredAllocs(func() {
					v, err := parse.parse(doc)
					if err != nil {
						t.Fatal(err)
					}
					parseSink = v
				})
				if m != 3 {
					t.Fatalf("doc %d (%d bytes of JSON) allocated %d mallocs, want 3", i, len(doc), m)
				}
				got = append(got, b)
			}
			if got[0] != got[1] || got[1] != got[2] {
				t.Fatalf("per-row documents allocated %v bytes; string length changed field storage", got)
			}
			if got[0] != int64(13)*int64(unsafe.Sizeof(field{}))+int64(unsafe.Sizeof(Value{})) {
				t.Fatalf("per-row documents allocated %d bytes, want 13 field slots plus the returned value", got[0])
			}
		})
	}
}
