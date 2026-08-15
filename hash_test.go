package jsonlite

import (
	"fmt"
	"hash/maphash"
	"strings"
	"testing"
)

// hashCorpora are key sets shaped like real JSON objects. Shared prefixes,
// shared suffixes and numeric tails are where a hash that reads only the ends
// of a key degrades, so they are the cases worth guarding.
func hashCorpora() map[string][]string {
	gen := func(format string, n int) []string {
		out := make([]string, n)
		for i := range out {
			out[i] = fmt.Sprintf(format, i)
		}
		return out
	}
	return map[string][]string{
		"cloudLogging": {
			"logName",
			"insertId",
			"timestamp",
			"receiveTimestamp",
			"severity",
			"textPayload",
			"resource",
			"labels",
			"httpRequest",
			"trace",
			"spanId",
			"traceSampled",
			"operation",
			"sourceLocation",
			"split",
			"requestMethod",
			"requestUrl",
			"requestSize",
			"status",
			"responseSize",
			"userAgent",
			"remoteIp",
			"serverIp",
			"referer",
			"latency",
			"cacheLookup",
			"cacheHit",
			"protocol",
		},
		"snakeCase": {
			"user_id",
			"user_name",
			"user_email",
			"user_created_at",
			"user_updated_at",
			"account_id",
			"account_name",
			"account_type",
			"request_id",
			"request_method",
			"request_url",
			"request_headers",
			"response_code",
			"response_body",
			"response_headers",
			"trace_id",
		},
		"shortKeys": {
			"id",
			"ts",
			"op",
			"k",
			"v",
			"a",
			"b",
			"x",
			"y",
			"z",
			"url",
			"sql",
			"err",
			"msg",
			"tag",
			"seq",
		},
		"sharedPrefix": gen("user_profile_%d", 32),
		"sharedSuffix": gen("f%d_timestamp_utc", 32),
		"numericTail":  gen("k%07d", 32),
	}
}

// keyCompares is the expected number of full key comparisons a successful
// Lookup performs against this key set: the cost a weak tag actually imposes.
// A perfect tag scores 1.00.
func keyCompares(keys []string) float64 {
	var buckets [256]int
	for _, k := range keys {
		buckets[hashKey(k)]++
	}
	total := 0.0
	for _, c := range buckets {
		total += float64(c*(c+1)) / 2
	}
	return total / float64(len(keys))
}

// TestHashKeyDispersion guards the property Lookup depends on: that the tag
// byte separates keys well enough that a probe rarely lands on the wrong
// field. hashKey reads only the ends of a key, so a regression here would
// show up as extra key comparisons rather than as a wrong answer, which no
// correctness test would catch.
func TestHashKeyDispersion(t *testing.T) {
	// The seed is randomized per process, so this runs against whatever seed
	// the test happens to draw. The bound is loose enough not to flake and
	// tight enough to catch a hash that stops mixing: measured values sit at
	// 1.00-1.20, and a first-byte-and-length tag scores 9.6 on sharedSuffix.
	const maxKeyCompares = 1.6
	for name, keys := range hashCorpora() {
		if got := keyCompares(keys); got > maxKeyCompares {
			t.Errorf("%s: %.2f key compares per lookup, want <= %.2f", name, got, maxKeyCompares)
		}
	}
}

// TestHashKeyDeterministic checks the tag is a pure function of the key: the
// parse-time tag and the Lookup-time tag have to agree or lookups miss.
func TestHashKeyDeterministic(t *testing.T) {
	for _, keys := range hashCorpora() {
		for _, k := range keys {
			want := hashKey(k)
			for range 4 {
				if got := hashKey(k); got != want {
					t.Fatalf("hashKey(%q) not stable: %d then %d", k, want, got)
				}
			}
			// A key that is a substring of a larger buffer must hash the same
			// as a standalone copy, since parsed keys alias the document.
			padded := ("<<<" + k + ">>>")[3 : 3+len(k)]
			if got := hashKey(padded); got != want {
				t.Errorf("hashKey(%q) = %d standalone, %d as a substring", k, want, got)
			}
		}
	}
}

// TestHashKeyLengths walks every key length across the tier boundaries at 4
// and 8 bytes, including the empty key, which no tier covers and whose loads
// must therefore be skipped entirely.
func TestHashKeyLengths(t *testing.T) {
	const alphabet = "abcdefghijklmnopqrstuvwxyz0123456789_-"
	for n := range 24 {
		k := alphabet[:n]
		want := hashKey(k)
		// Hashing a same-length key carved out of a different buffer must
		// agree: the tiers may only read bytes belonging to the key.
		if got := hashKey(strings.Clone(k)); got != want {
			t.Errorf("len %d: %d for the original, %d for a copy", n, want, got)
		}
	}
}

func BenchmarkHashKey(b *testing.B) {
	keys := hashCorpora()["cloudLogging"]
	b.Run("hashKey", func(b *testing.B) {
		var s byte
		for b.Loop() {
			for _, k := range keys {
				s ^= hashKey(k)
			}
		}
		hashSink = s
	})
	// The baseline hashKey replaced, kept so the comparison stays runnable.
	b.Run("maphash", func(b *testing.B) {
		var s byte
		for b.Loop() {
			for _, k := range keys {
				s ^= byte(maphash.String(hashseed, k))
			}
		}
		hashSink = s
	})
}

var hashSink byte
