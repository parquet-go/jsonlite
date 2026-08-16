package benchdata_test

import (
	"encoding/json"
	"testing"

	"github.com/parquet-go/jsonlite/internal/benchdata"
)

func TestCorpusIsValidJSON(t *testing.T) {
	for _, d := range benchdata.Corpus() {
		var v any
		if err := json.Unmarshal([]byte(d.JSON), &v); err != nil {
			t.Errorf("%s: invalid JSON: %v", d.Name, err)
			continue
		}
		t.Logf("%-16s %8d bytes  %s", d.Name, len(d.JSON), d.Traits)
	}
	jsonl := benchdata.JSONLines(100)
	t.Logf("%-16s %8d bytes  (100 records)", "jsonl", len(jsonl))
}
