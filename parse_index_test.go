package jsonlite

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// differentialInputs collects a broad set of valid and invalid JSON inputs.
func differentialInputs(t testing.TB) []string {
	inputs := []string{
		`null`, `true`, `false`, `0`, `-1`, `3.14159`, `1e10`, `-2.5e-3`,
		`""`, `"a"`, `"hello world"`, `"héllo wörld 🎉"`, `"\n\t\\\""`,
		`"Aé😀"`,
		`[]`, `[1]`, `[1,2,3]`, `[[[]]]`, `[1,[2,[3,[4]]]]`,
		`{}`, `{"a":1}`, `{"a":{"b":{"c":[1,2,3]}}}`,
		`{"escaped\"key":"escaped\"value"}`,
		` { "a" : [ 1 , 2 ] , "b" : null } `,
		"{\n\t\"pretty\": true,\n\t\"list\": [\n\t\t1,\n\t\t2\n\t]\n}",
		`[1, 2, 3, "four", null, true, false, {"five": 5}]`,
		`{"empty_obj":{},"empty_arr":[],"s":"","n":0}`,
		// invalid inputs
		``, ` `, `{`, `}`, `[`, `]`, `[1,]`, `{"a":}`, `{"a"}`, `{a:1}`,
		`"unterminated`, `"bad \x escape"`, "\"raw\tcontrol\"", `nul`, `truee`,
		`01`, `1.`, `1e`, `--1`, `[1 2]`, `{"a":1,}`, `{"a":1}extra`, `1 2`,
		`["a\"]`, `"\ud800"`, `[}`, `{]`,
	}
	// Long strings crossing 64-byte block boundaries, with escapes at
	// various positions relative to block edges.
	long := strings.Repeat("abcdefgh", 20)
	for _, pos := range []int{0, 7, 62, 63, 64, 65, 127, 128} {
		s := long[:pos] + `\"` + long[pos:]
		inputs = append(inputs, `"`+s+`"`, `{"key":"`+s+`"}`)
		b := long[:pos] + `\\` + long[pos:]
		inputs = append(inputs, `"`+b+`"`)
	}
	// Backslash runs of every length up to 8 at a block boundary.
	for n := 1; n <= 8; n++ {
		pad := strings.Repeat("x", 60)
		inputs = append(inputs, `"`+pad+strings.Repeat(`\\`, n)+`"`)
		inputs = append(inputs, `"`+pad+strings.Repeat(`\`, n)+`n"`)
	}
	// Fuzz corpus, if present.
	for _, dir := range []string{"testdata/fuzz/FuzzParse", "testdata/fuzz/FuzzUnquote"} {
		files, _ := filepath.Glob(filepath.Join(dir, "*"))
		for _, f := range files {
			data, err := os.ReadFile(f)
			if err == nil {
				lines := strings.Split(string(data), "\n")
				for _, line := range lines {
					if strings.HasPrefix(line, "string(") {
						if s, err := unquoteCorpusLine(line); err == nil {
							inputs = append(inputs, s)
						}
					}
				}
			}
		}
	}
	return inputs
}

func unquoteCorpusLine(line string) (string, error) {
	line = strings.TrimPrefix(line, "string(")
	line = strings.TrimSuffix(line, ")")
	return strconv.Unquote(line)
}

// TestParseIndexedDifferential verifies parseIndexed agrees with Parse on
// success/failure and produces identical trees.
func TestParseIndexedDifferential(t *testing.T) {
	for _, input := range differentialInputs(t) {
		v1, err1 := Parse(input)
		v2, err2 := parseIndexed(input, DefaultMaxDepth)
		if (err1 == nil) != (err2 == nil) {
			t.Errorf("input %q: Parse err=%v, parseIndexed err=%v", input, err1, err2)
			continue
		}
		if err1 != nil {
			continue
		}
		if j1, j2 := v1.JSON(), v2.JSON(); j1 != j2 {
			t.Errorf("input %q: JSON mismatch:\n  Parse:        %q\n  parseIndexed: %q", input, j1, j2)
		}
		if c1, c2 := string(v1.Compact(nil)), string(v2.Compact(nil)); c1 != c2 {
			t.Errorf("input %q: Compact mismatch:\n  Parse:        %q\n  parseIndexed: %q", input, c1, c2)
		}
	}
}

// TestParseIndexedMaxDepth verifies lazy-object parity.
func TestParseIndexedMaxDepth(t *testing.T) {
	input := `{"a":{"b":{"c":{"d":1}}},"e":[{"f":2}]}`
	for depth := 0; depth < 6; depth++ {
		v1, err1 := ParseMaxDepth(input, depth)
		v2, err2 := parseIndexed(input, depth)
		if (err1 == nil) != (err2 == nil) {
			t.Fatalf("depth %d: err mismatch %v vs %v", depth, err1, err2)
		}
		if err1 != nil {
			continue
		}
		if j1, j2 := string(v1.Compact(nil)), string(v2.Compact(nil)); j1 != j2 {
			t.Errorf("depth %d: %q vs %q", depth, j1, j2)
		}
	}
}

// TestFindEscapedReference checks the block escape scanner against a naive
// per-byte reference on random inputs.
func TestFindEscapedReference(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	chars := []byte{'\\', '"', 'a'}
	for trial := 0; trial < 2000; trial++ {
		n := 1 + rng.Intn(256)
		b := make([]byte, n)
		for i := range b {
			b[i] = chars[rng.Intn(len(chars))]
		}
		// reference: escaped[i] = preceded by odd run of backslashes
		escaped := make([]bool, n)
		for i := 0; i < n; i++ {
			if b[i] == '\\' && !escaped[i] && i+1 < n {
				escaped[i+1] = true
			}
		}
		var st stage1State
		for base := 0; base < n; base += 64 {
			var block [64]byte
			for j := range block {
				block[j] = 'a'
			}
			copy(block[:], b[base:min(n, base+64)])
			var bs uint64
			for j := 0; j < 64 && base+j < n; j++ {
				if block[j] == '\\' {
					bs |= 1 << j
				}
			}
			got := st.findEscaped(bs)
			for j := 0; j < 64 && base+j < n; j++ {
				want := escaped[base+j]
				if (got&(1<<j) != 0) != want {
					t.Fatalf("trial %d: input %q: byte %d escaped=%v, want %v", trial, b, base+j, !want, want)
				}
			}
		}
	}
}

func FuzzParseIndexedDifferential(f *testing.F) {
	for _, input := range differentialInputs(f) {
		f.Add(input)
	}
	f.Fuzz(func(t *testing.T, input string) {
		v1, err1 := Parse(input)
		v2, err2 := parseIndexed(input, DefaultMaxDepth)
		if (err1 == nil) != (err2 == nil) {
			t.Fatalf("input %q: Parse err=%v, parseIndexed err=%v", input, err1, err2)
		}
		if err1 == nil {
			if j1, j2 := v1.JSON(), v2.JSON(); j1 != j2 {
				t.Fatalf("input %q: %q vs %q", input, j1, j2)
			}
		}
	})
}

func BenchmarkParseIndexed(b *testing.B) {
	benchmarks := []struct {
		name  string
		input string
	}{
		{"Small", `{"name":"test","age":42,"active":true}`},
		{"Medium", `{
				"name": "test",
				"age": 42,
				"active": true,
				"tags": ["a", "b", "c"],
				"metadata": {
					"created": "2024-01-01",
					"updated": null
				}
			}`},
		{"CloudLogging", internalCloudLoggingPayload},
	}
	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			b.ReportAllocs()
			b.SetBytes(int64(len(bm.input)))
			for b.Loop() {
				v, err := parseIndexed(bm.input, DefaultMaxDepth)
				if err != nil {
					b.Fatal(err)
				}
				_ = v
			}
		})
	}
}

func BenchmarkStage1(b *testing.B) {
	input := internalCloudLoggingPayload
	b.SetBytes(int64(len(input)))
	b.ReportAllocs()
	index := make([]uint32, 0, 1024)
	for b.Loop() {
		var err error
		index, _, err = structuralIndex(input, index[:0])
		if err != nil {
			b.Fatal(err)
		}
	}
}

const internalCloudLoggingPayload = `{
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

// classicParse replicates the classic (tokenizer) branch of ParseMaxDepth so
// benchmarks can compare it against parseIndexed regardless of the CPU
// dispatch.
func classicParse(data string, maxDepth int) (*Value, error) {
	p := getParser()
	v, rest, err := p.parseValue(data, max(0, maxDepth))
	putParser(p)
	if err != nil {
		return nil, err
	}
	if extra, _, ok := nextToken(rest); ok {
		return nil, fmt.Errorf("unexpected token after root value: %q", extra)
	}
	return &v, nil
}

// BenchmarkParsePathCrossover compares the classic and indexed parse paths
// across document sizes, to validate indexedParseThreshold per platform.
func BenchmarkParsePathCrossover(b *testing.B) {
	unit := `{"id":123,"name":"item-name","ok":true},`
	for _, size := range []int{256, 512, 1024, 2048, 8192} {
		var sb strings.Builder
		sb.WriteString(`{"items":[`)
		for sb.Len() < size-24 {
			sb.WriteString(unit)
		}
		doc := strings.TrimSuffix(sb.String(), ",") + `],"total":12345}`
		for _, path := range []struct {
			name string
			fn   func(string, int) (*Value, error)
		}{{"classic", classicParse}, {"indexed", parseIndexed}} {
			b.Run(fmt.Sprintf("%s/%d", path.name, len(doc)), func(b *testing.B) {
				b.SetBytes(int64(len(doc)))
				for b.Loop() {
					if _, err := path.fn(doc, DefaultMaxDepth); err != nil {
						b.Fatal(err)
					}
				}
			})
		}
	}
}
