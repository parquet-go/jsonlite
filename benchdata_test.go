package jsonlite

// The benchmark corpus.
//
// Documents are generated from a fixed seed, so a corpus is byte-identical
// across runs and machines and benchmark numbers stay comparable. They are
// chosen to span the axes that change which code runs: object width either
// side of smallObjectFields, nesting depth, the value type mix, whether
// strings need unescaping, whitespace density, and document size either side
// of indexedParseThreshold.
//
// This lives in package jsonlite rather than an external test package because
// the scan and parse-path benchmarks reach for unexported functions.

import (
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"strings"
	"testing"
)

// benchDoc is one corpus entry. Traits records which code paths the document is
// meant to exercise, so a benchmark result can be read back to a cause.
type benchDoc struct {
	Name   string
	JSON   string
	Traits string
}

// benchCorpus returns the full corpus, ordered small to large.
//
// The documents are chosen to span the axes that change which code runs:
// object width either side of smallObjectFields (8), nesting depth, the value
// type mix, whether strings need unescaping, whitespace density, and document
// size either side of indexedParseThreshold (512).
func benchCorpus() []benchDoc {
	return []benchDoc{
		{"cloud_logging", internalCloudLoggingPayload, "realistic log record; 15-field root, nested groups"},
		{"narrow_records", benchNarrowRecords(200), "array of 4-field objects; below the hash-index threshold"},
		{"wide_records", benchWideRecords(200, 24), "array of 24-field objects; every object builds a hash index"},
		{"deep_nested", benchDeepNested(64), "64 levels of nesting; exercises depth handling"},
		{"numbers_float", benchNumbersFloat(20000), "coordinate pairs; float parsing dominates"},
		{"numbers_int", benchNumbersInt(20000), "integer arrays; integer parsing dominates"},
		{"strings_plain", benchStringsPlain(4000), "long ASCII strings, no escapes; fast unquote path"},
		{"strings_escaped", benchStringsEscaped(4000), "quotes, backslashes and \\u escapes; slow unquote path"},
		{"strings_long", benchStringsLong(400, 512), "512-byte text fields, the shape a log or comment payload has"},
		{"pretty_printed", benchPretty(benchGitHubEvents(400)), "whitespace-heavy; stage 1 and the tokenizer must skip it"},
		{"minified", benchGitHubEvents(400), "same shape as pretty_printed with no whitespace"},
		{"tiny", `{"id":1,"ok":true}`, "below indexedParseThreshold; the scalar path always wins here"},
	}
}

// benchJSONLines returns a JSON Lines document of n records, the shape jsonlite's
// ParseSeq and Iterator are aimed at.
func benchJSONLines(n int) string {
	r := benchRand()
	var b strings.Builder
	for i := range n {
		if i > 0 {
			b.WriteByte('\n')
		}
		b.WriteString(benchEventRecord(r, i))
	}
	return b.String()
}

// benchRand returns a generator with a fixed seed so corpora are reproducible.
func benchRand() *rand.Rand { return rand.New(rand.NewPCG(0x5EED, 0xC0FFEE)) }

var benchWords = []string{
	"alpha", "bravo", "charlie", "delta", "echo", "foxtrot", "golf", "hotel",
	"india", "juliet", "kilo", "lima", "mike", "november", "oscar", "papa",
}

func benchWord(r *rand.Rand) string { return benchWords[r.IntN(len(benchWords))] }

func benchSentence(r *rand.Rand, n int) string {
	parts := make([]string, n)
	for i := range parts {
		parts[i] = benchWord(r)
	}
	return strings.Join(parts, " ")
}
func benchNarrowRecords(n int) string {
	r := benchRand()
	var b strings.Builder
	b.WriteByte('[')
	for i := range n {
		if i > 0 {
			b.WriteByte(',')
		}
		fmt.Fprintf(&b, `{"id":%d,"name":"%s","score":%d,"active":%t}`,
			i, benchWord(r), r.IntN(1000), i%2 == 0)
	}
	b.WriteByte(']')
	return b.String()
}

func benchWideRecords(n, fields int) string {
	r := benchRand()
	var b strings.Builder
	b.WriteByte('[')
	for i := range n {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteByte('{')
		for j := range fields {
			if j > 0 {
				b.WriteByte(',')
			}
			// Keys share a prefix, which is the case a hash reading only the
			// ends of a key handles worst.
			fmt.Fprintf(&b, `"field_%s_%02d":`, benchWord(r), j)
			switch j % 4 {
			case 0:
				fmt.Fprintf(&b, "%d", r.IntN(1<<20))
			case 1:
				fmt.Fprintf(&b, `"%s"`, benchSentence(r, 3))
			case 2:
				fmt.Fprintf(&b, "%t", j%8 == 2)
			case 3:
				fmt.Fprintf(&b, "%.6f", r.Float64()*1000)
			}
		}
		b.WriteByte('}')
	}
	b.WriteByte(']')
	return b.String()
}

func benchDeepNested(depth int) string {
	var b strings.Builder
	for i := range depth {
		fmt.Fprintf(&b, `{"level_%d":`, i)
	}
	b.WriteString(`"bottom"`)
	b.WriteString(strings.Repeat("}", depth))
	return b.String()
}

func benchNumbersFloat(n int) string {
	r := benchRand()
	var b strings.Builder
	b.WriteString(`{"type":"Polygon","coordinates":[[`)
	for i := range n {
		if i > 0 {
			b.WriteByte(',')
		}
		fmt.Fprintf(&b, "[%.15f,%.15f]", r.Float64()*360-180, r.Float64()*180-90)
	}
	b.WriteString(`]]}`)
	return b.String()
}

func benchNumbersInt(n int) string {
	r := benchRand()
	var b strings.Builder
	b.WriteString(`{"values":[`)
	for i := range n {
		if i > 0 {
			b.WriteByte(',')
		}
		fmt.Fprintf(&b, "%d", r.Int64N(1<<40)-(1<<39))
	}
	b.WriteString(`]}`)
	return b.String()
}

func benchStringsPlain(n int) string {
	r := benchRand()
	var b strings.Builder
	b.WriteString(`{"lines":[`)
	for i := range n {
		if i > 0 {
			b.WriteByte(',')
		}
		fmt.Fprintf(&b, `"%s"`, benchSentence(r, 12))
	}
	b.WriteString(`]}`)
	return b.String()
}

func benchStringsEscaped(n int) string {
	r := benchRand()
	var b strings.Builder
	b.WriteString(`{"lines":[`)
	for i := range n {
		if i > 0 {
			b.WriteByte(',')
		}
		fmt.Fprintf(&b, `"%s \"quoted\" %s\n\tpath\\to\\file é中%s"`,
			benchWord(r), benchWord(r), benchWord(r))
	}
	b.WriteString(`]}`)
	return b.String()
}

func benchGitHubEvents(n int) string {
	r := benchRand()
	var b strings.Builder
	b.WriteByte('[')
	for i := range n {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(benchEventRecord(r, i))
	}
	b.WriteByte(']')
	return b.String()
}

// benchEventRecord is shaped after a GitHub Archive event: a wide root object with
// nested actor and repo groups and a mix of value types.
func benchEventRecord(r *rand.Rand, i int) string {
	return fmt.Sprintf(`{"id":"%d","type":"%sEvent","public":%t,`+
		`"created_at":"2024-01-%02dT%02d:%02d:%02dZ",`+
		`"actor":{"id":%d,"login":"%s","display_login":"%s",`+
		`"gravatar_id":"","url":"https://api.github.com/users/%s","avatar_url":"https://avatars.githubusercontent.com/u/%d?"},`+
		`"repo":{"id":%d,"name":"%s/%s","url":"https://api.github.com/repos/%s/%s"},`+
		`"payload":{"ref":"refs/heads/main","ref_type":"branch","master_branch":"main",`+
		`"description":"%s","pusher_type":"user","push_id":%d,"size":%d,"distinct_size":%d,`+
		`"head":"%016x","before":"%016x"},"org":{"id":%d,"login":"%s"}}`,
		1000000+i, benchWord(r), i%3 != 0,
		1+i%28, i%24, i%60, (i*7)%60,
		r.IntN(1<<22), benchWord(r), benchWord(r), benchWord(r), r.IntN(1<<22),
		r.IntN(1<<24), benchWord(r), benchWord(r), benchWord(r), benchWord(r),
		benchSentence(r, 8), r.Int64N(1<<40), r.IntN(10), r.IntN(10),
		r.Uint64(), r.Uint64(), r.IntN(1<<20), benchWord(r))
}

// benchPretty re-indents a document to make it whitespace-heavy, roughly doubling
// its size, without changing the values.
func benchPretty(src string) string {
	var b strings.Builder
	depth, inString, escaped := 0, false, false
	newline := func() {
		b.WriteByte('\n')
		for range depth {
			b.WriteString("    ")
		}
	}
	for i := range len(src) {
		c := src[i]
		if inString {
			b.WriteByte(c)
			switch {
			case escaped:
				escaped = false
			case c == '\\':
				escaped = true
			case c == '"':
				inString = false
			}
			continue
		}
		switch c {
		case '"':
			inString = true
			b.WriteByte(c)
		case '{', '[':
			depth++
			b.WriteByte(c)
			newline()
		case '}', ']':
			depth--
			newline()
			b.WriteByte(c)
		case ',':
			b.WriteByte(c)
			newline()
		case ':':
			b.WriteString(": ")
		default:
			b.WriteByte(c)
		}
	}
	return b.String()
}

// benchSizedRecord returns a realistic record padded to approximately size bytes,
// for locating the crossover between the scalar and indexed parse paths.
func benchSizedRecord(size int) string {
	r := benchRand()
	var b strings.Builder
	b.WriteString(`{"id":1,"ts":"2024-01-15T10:30:00Z","lvl":"INFO"`)
	for i := 0; b.Len() < size-24; i++ {
		fmt.Fprintf(&b, `,"f%02d":"%s"`, i, benchSentence(r, 1+r.IntN(3)))
	}
	b.WriteByte('}')
	return b.String()
}

// TestBenchCorpusIsValidJSON keeps the generators honest. A corpus document
// that is subtly malformed would still parse far enough to produce plausible
// benchmark numbers while measuring the error paths, so check every document
// against encoding/json before trusting anything measured on it.
func TestBenchCorpusIsValidJSON(t *testing.T) {
	for _, d := range benchCorpus() {
		var v any
		if err := json.Unmarshal([]byte(d.JSON), &v); err != nil {
			t.Errorf("%s (%d bytes): invalid JSON: %v", d.Name, len(d.JSON), err)
		}
	}
	for _, line := range strings.Split(benchJSONLines(20), "\n") {
		var v any
		if err := json.Unmarshal([]byte(line), &v); err != nil {
			t.Errorf("JSON Lines record invalid: %v", err)
		}
	}
	for _, size := range []int{64, 256, 1024} {
		var v any
		if err := json.Unmarshal([]byte(benchSizedRecord(size)), &v); err != nil {
			t.Errorf("sized record %d: invalid JSON: %v", size, err)
		}
	}
}

// benchStringsLong builds records whose string values are long, the shape a
// log line, comment body or embedded blob has. The other string documents
// hold values of a few tens of bytes, which leaves the behaviour of the scan
// on long strings unmeasured.
func benchStringsLong(n, size int) string {
	r := benchRand()
	var b strings.Builder
	b.WriteString(`{"records":[`)
	for i := range n {
		if i > 0 {
			b.WriteByte(',')
		}
		var text strings.Builder
		for text.Len() < size {
			text.WriteString(benchSentence(r, 8))
			text.WriteByte(' ')
		}
		fmt.Fprintf(&b, `{"id":%d,"message":"%s"}`, i, text.String()[:size])
	}
	b.WriteString(`]}`)
	return b.String()
}
