// Package benchdata builds the JSON corpus the benchmarks run against.
//
// It lives in internal/ so both the jsonlite and jsonlite_test packages can
// share one corpus; before this existed the cloud-logging payload was pasted
// into two test files and everything else was a handful of inline literals
// well under a kilobyte, which left most of the parser's behaviour unmeasured.
//
// Every document is generated from a fixed seed, so a corpus is byte-identical
// across runs and machines and benchmark numbers stay comparable.
package benchdata

import (
	"fmt"
	"math/rand/v2"
	"strings"
)

// Doc is one corpus entry. Traits records which code paths the document is
// meant to exercise, so a benchmark result can be read back to a cause.
type Doc struct {
	Name   string
	JSON   string
	Traits string
}

// Corpus returns the full corpus, ordered small to large.
//
// The documents are chosen to span the axes that change which code runs:
// object width either side of smallObjectFields (8), nesting depth, the value
// type mix, whether strings need unescaping, whitespace density, and document
// size either side of indexedParseThreshold (512).
func Corpus() []Doc {
	return []Doc{
		{"cloud_logging", CloudLogging, "realistic log record; 15-field root, nested groups"},
		{"narrow_records", narrowRecords(200), "array of 4-field objects; below the hash-index threshold"},
		{"wide_records", wideRecords(200, 24), "array of 24-field objects; every object builds a hash index"},
		{"deep_nested", deepNested(64), "64 levels of nesting; exercises depth handling"},
		{"numbers_float", numbersFloat(20000), "coordinate pairs; float parsing dominates"},
		{"numbers_int", numbersInt(20000), "integer arrays; integer parsing dominates"},
		{"strings_plain", stringsPlain(4000), "long ASCII strings, no escapes; fast unquote path"},
		{"strings_escaped", stringsEscaped(4000), "quotes, backslashes and \\u escapes; slow unquote path"},
		{"pretty_printed", pretty(gitHubEvents(400)), "whitespace-heavy; stage 1 and the tokenizer must skip it"},
		{"minified", gitHubEvents(400), "same shape as pretty_printed with no whitespace"},
		{"tiny", `{"id":1,"ok":true}`, "below indexedParseThreshold; the scalar path always wins here"},
	}
}

// JSONLines returns a JSON Lines document of n records, the shape jsonlite's
// ParseSeq and Iterator are aimed at.
func JSONLines(n int) string {
	r := newRand()
	var b strings.Builder
	for i := range n {
		if i > 0 {
			b.WriteByte('\n')
		}
		b.WriteString(eventRecord(r, i))
	}
	return b.String()
}

// newRand returns a generator with a fixed seed so corpora are reproducible.
func newRand() *rand.Rand { return rand.New(rand.NewPCG(0x5EED, 0xC0FFEE)) }

var words = []string{
	"alpha", "bravo", "charlie", "delta", "echo", "foxtrot", "golf", "hotel",
	"india", "juliet", "kilo", "lima", "mike", "november", "oscar", "papa",
}

func word(r *rand.Rand) string { return words[r.IntN(len(words))] }

func sentence(r *rand.Rand, n int) string {
	parts := make([]string, n)
	for i := range parts {
		parts[i] = word(r)
	}
	return strings.Join(parts, " ")
}

// CloudLogging is a Google Cloud Logging record: a 15-field root object with
// nested groups both above and below the hash-index threshold. Previously
// duplicated in parse_test.go and parse_index_test.go.
const CloudLogging = `{
        "logName": "projects/test-project/logs/test-log",
        "insertId": "test-insert-id",
        "timestamp": "2024-01-15T10:30:00Z",
        "receiveTimestamp": "2024-01-15T10:30:01Z",
        "severity": "INFO",
        "textPayload": "test log message",
        "resource": {
            "type": "gce_instance",
            "labels": {
                "instance_id": "1234567890",
                "zone": "us-central1-a"
            }
        },
        "labels": {
            "env": "test"
        },
        "httpRequest": {
            "requestMethod": "GET",
            "requestUrl": "https://example.com/api",
            "requestSize": 1024,
            "status": 200,
            "responseSize": 2048,
            "userAgent": "Mozilla/5.0",
            "remoteIp": "192.168.1.1",
            "serverIp": "10.0.0.1",
            "referer": "https://example.com",
            "latency": "0.5s",
            "cacheLookup": true,
            "cacheHit": false,
            "protocol": "HTTP/1.1"
        },
        "trace": "projects/test-project/traces/1234567890abcdef",
        "spanId": "abcdef1234567890",
        "traceSampled": true,
        "operation": {
            "id": "op-123",
            "producer": "test-producer",
            "first": true,
            "last": false
        },
        "sourceLocation": {
            "file": "test.go",
            "line": 42,
            "function": "TestFunction"
        },
        "split": {
            "uid": "split-123",
            "index": 1,
            "totalSplits": 3
        }
    }`

func narrowRecords(n int) string {
	r := newRand()
	var b strings.Builder
	b.WriteByte('[')
	for i := range n {
		if i > 0 {
			b.WriteByte(',')
		}
		fmt.Fprintf(&b, `{"id":%d,"name":"%s","score":%d,"active":%t}`,
			i, word(r), r.IntN(1000), i%2 == 0)
	}
	b.WriteByte(']')
	return b.String()
}

func wideRecords(n, fields int) string {
	r := newRand()
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
			fmt.Fprintf(&b, `"field_%s_%02d":`, word(r), j)
			switch j % 4 {
			case 0:
				fmt.Fprintf(&b, "%d", r.IntN(1<<20))
			case 1:
				fmt.Fprintf(&b, `"%s"`, sentence(r, 3))
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

func deepNested(depth int) string {
	var b strings.Builder
	for i := range depth {
		fmt.Fprintf(&b, `{"level_%d":`, i)
	}
	b.WriteString(`"bottom"`)
	b.WriteString(strings.Repeat("}", depth))
	return b.String()
}

func numbersFloat(n int) string {
	r := newRand()
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

func numbersInt(n int) string {
	r := newRand()
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

func stringsPlain(n int) string {
	r := newRand()
	var b strings.Builder
	b.WriteString(`{"lines":[`)
	for i := range n {
		if i > 0 {
			b.WriteByte(',')
		}
		fmt.Fprintf(&b, `"%s"`, sentence(r, 12))
	}
	b.WriteString(`]}`)
	return b.String()
}

func stringsEscaped(n int) string {
	r := newRand()
	var b strings.Builder
	b.WriteString(`{"lines":[`)
	for i := range n {
		if i > 0 {
			b.WriteByte(',')
		}
		fmt.Fprintf(&b, `"%s \"quoted\" %s\n\tpath\\to\\file é中%s"`,
			word(r), word(r), word(r))
	}
	b.WriteString(`]}`)
	return b.String()
}

func gitHubEvents(n int) string {
	r := newRand()
	var b strings.Builder
	b.WriteByte('[')
	for i := range n {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(eventRecord(r, i))
	}
	b.WriteByte(']')
	return b.String()
}

// eventRecord is shaped after a GitHub Archive event: a wide root object with
// nested actor and repo groups and a mix of value types.
func eventRecord(r *rand.Rand, i int) string {
	return fmt.Sprintf(`{"id":"%d","type":"%sEvent","public":%t,`+
		`"created_at":"2024-01-%02dT%02d:%02d:%02dZ",`+
		`"actor":{"id":%d,"login":"%s","display_login":"%s",`+
		`"gravatar_id":"","url":"https://api.github.com/users/%s","avatar_url":"https://avatars.githubusercontent.com/u/%d?"},`+
		`"repo":{"id":%d,"name":"%s/%s","url":"https://api.github.com/repos/%s/%s"},`+
		`"payload":{"ref":"refs/heads/main","ref_type":"branch","master_branch":"main",`+
		`"description":"%s","pusher_type":"user","push_id":%d,"size":%d,"distinct_size":%d,`+
		`"head":"%016x","before":"%016x"},"org":{"id":%d,"login":"%s"}}`,
		1000000+i, word(r), i%3 != 0,
		1+i%28, i%24, i%60, (i*7)%60,
		r.IntN(1<<22), word(r), word(r), word(r), r.IntN(1<<22),
		r.IntN(1<<24), word(r), word(r), word(r), word(r),
		sentence(r, 8), r.Int64N(1<<40), r.IntN(10), r.IntN(10),
		r.Uint64(), r.Uint64(), r.IntN(1<<20), word(r))
}

// pretty re-indents a document to make it whitespace-heavy, roughly doubling
// its size, without changing the values.
func pretty(src string) string {
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
