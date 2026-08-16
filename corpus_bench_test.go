package jsonlite_test

import (
	"testing"

	"github.com/parquet-go/jsonlite"
	"github.com/parquet-go/jsonlite/internal/benchdata"
)

// BenchmarkCorpusParse parses each corpus document into the Value tree.
func BenchmarkCorpusParse(b *testing.B) {
	for _, d := range benchdata.Corpus() {
		b.Run(d.Name, func(b *testing.B) {
			b.ReportAllocs()
			b.SetBytes(int64(len(d.JSON)))
			for b.Loop() {
				if _, err := jsonlite.Parse(d.JSON); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// BenchmarkCorpusIterate walks each document with the streaming Iterator,
// which never materializes a Value. Comparing it against BenchmarkCorpusParse
// separates the cost of scanning from the cost of building the tree.
func BenchmarkCorpusIterate(b *testing.B) {
	for _, d := range benchdata.Corpus() {
		b.Run(d.Name, func(b *testing.B) {
			b.ReportAllocs()
			b.SetBytes(int64(len(d.JSON)))
			it := jsonlite.Iterate(d.JSON)
			for b.Loop() {
				it.Reset(d.JSON)
				for it.Next() {
				}
				if err := it.Err(); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// BenchmarkCorpusValid is the cheapest full pass over a document: no tree, no
// token materialization. It is the floor any parse path can aim at.
func BenchmarkCorpusValid(b *testing.B) {
	for _, d := range benchdata.Corpus() {
		b.Run(d.Name, func(b *testing.B) {
			b.SetBytes(int64(len(d.JSON)))
			for b.Loop() {
				if !jsonlite.Valid(d.JSON) {
					b.Fatal("invalid")
				}
			}
		})
	}
}

// BenchmarkJSONLines covers the JSON Lines shape, where each record is parsed
// separately and the per-document setup cost is paid many times over.
func BenchmarkJSONLines(b *testing.B) {
	const records = 1000
	input := benchdata.JSONLines(records)
	b.Run("ParseSeq", func(b *testing.B) {
		b.ReportAllocs()
		b.SetBytes(int64(len(input)))
		for b.Loop() {
			n := 0
			for _, err := range jsonlite.ParseSeq(input) {
				if err != nil {
					b.Fatal(err)
				}
				n++
			}
			if n != records {
				b.Fatalf("got %d records, want %d", n, records)
			}
		}
	})
	b.Run("Iterate", func(b *testing.B) {
		b.ReportAllocs()
		b.SetBytes(int64(len(input)))
		it := jsonlite.Iterate(input)
		for b.Loop() {
			it.Reset(input)
			for it.Next() {
			}
			if err := it.Err(); err != nil {
				b.Fatal(err)
			}
		}
	})
}

// BenchmarkCorpusIterateKeys walks each document and reads every key, which
// is what a real consumer does. BenchmarkCorpusIterate never touches Key(),
// so the two together separate the cost of producing keys from the cost of
// walking structure.
func BenchmarkCorpusIterateKeys(b *testing.B) {
	for _, d := range benchdata.Corpus() {
		b.Run(d.Name, func(b *testing.B) {
			b.ReportAllocs()
			b.SetBytes(int64(len(d.JSON)))
			it := jsonlite.Iterate(d.JSON)
			var sink int
			for b.Loop() {
				it.Reset(d.JSON)
				for it.Next() {
					sink += len(it.Key())
				}
			}
			if err := it.Err(); err != nil {
				b.Fatal(err)
			}
			_ = sink
		})
	}
}
