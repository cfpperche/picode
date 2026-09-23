package desktop

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestRecordSampleOneLinePerDay(t *testing.T) {
	path := filepath.Join(t.TempDir(), "h.jsonl")
	if err := RecordSample(path, Sample{Day: "2026-09-20", WinFreeBytes: 27}); err != nil {
		t.Fatal(err)
	}
	RecordSample(path, Sample{Day: "2026-09-23", WinFreeBytes: 12})
	RecordSample(path, Sample{Day: "2026-09-23", WinFreeBytes: 10}) // same day replaces
	got := ReadHistory(path)
	if len(got) != 2 || got[1].WinFreeBytes != 10 || got[0].Day != "2026-09-20" {
		t.Fatalf("history = %+v", got)
	}
}

func TestRecordSampleBoundsTheFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "h.jsonl")
	start := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	for i := 0; i < HistoryMax+10; i++ {
		RecordSample(path, Sample{Day: start.AddDate(0, 0, i).Format("2006-01-02")})
	}
	got := ReadHistory(path)
	if len(got) != HistoryMax || got[0].Day != start.AddDate(0, 0, 10).Format("2006-01-02") {
		t.Fatalf("len %d, first %s", len(got), got[0].Day)
	}
}

func TestReadHistorySkipsBrokenLines(t *testing.T) {
	path := filepath.Join(t.TempDir(), "h.jsonl")
	os.WriteFile(path, []byte("{\"day\":\"2026-09-21\"}\nnot json\n{}\n{\"day\":\"<img src=x>\"}\n{\"day\":\"2026-09-22\"}\n"), 0o644)
	if got := ReadHistory(path); len(got) != 2 {
		t.Errorf("got %+v", got)
	}
	if got := ReadHistory(filepath.Join(t.TempDir(), "missing")); got != nil {
		t.Errorf("missing file = %+v", got)
	}
}

// TestGrowthSince is the decision table for "what grew this week":
//
//	samples                        | → growth
//	fewer than two                 | none
//	none at least 7 days old       | none (a trend needs a baseline)
//	baseline 7+ days old           | per cache, largest first, zero changes left out
func TestGrowthSince(t *testing.T) {
	if GrowthSince([]Sample{{Day: "2026-09-23"}}, 7) != nil {
		t.Error("one sample has no growth")
	}
	recent := []Sample{{Day: "2026-09-20", Caches: map[string]int64{"go-build": 1}}, {Day: "2026-09-23", Caches: map[string]int64{"go-build": 9}}}
	if GrowthSince(recent, 7) != nil {
		t.Error("no baseline 7 days old")
	}
	s := []Sample{
		{Day: "2026-09-10", Caches: map[string]int64{"go-build": 100, "npm": 50, "uv": 5}},
		{Day: "2026-09-15", Caches: map[string]int64{"go-build": 200, "npm": 50, "uv": 5}},
		{Day: "2026-09-20", Caches: map[string]int64{"go-build": 900}},
		{Day: "2026-09-23", Caches: map[string]int64{"go-build": 1000, "npm": 40, "uv": 5, "new": 7}},
	}
	g := GrowthSince(s, 7)
	got := fmt.Sprint(g)
	// "new" was not measured on the baseline day: no growth claimed.
	want := "[{go-build 800 2026-09-15} {npm -10 2026-09-15}]"
	if got != want {
		t.Errorf("growth = %s, want %s", got, want)
	}
}
