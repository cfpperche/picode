package session

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func mustWrite(t *testing.T, dir, name, body string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

// unixTS is seconds, for readability at call sites (time.Time.Unix()) — pi's
// real wire format is epoch milliseconds, so the embedded value is scaled.
func msgLine(unixTS int64, cost float64) string {
	if unixTS == 0 {
		return fmt.Sprintf(`{"type":"message","message":{"role":"assistant","usage":{"cost":{"total":%v}}}}`, cost)
	}
	return fmt.Sprintf(`{"type":"message","message":{"role":"assistant","timestamp":%d,"usage":{"cost":{"total":%v}}}}`, unixTS*1000, cost)
}

func TestStatsRootBucketsByMessageDay(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "--tmp--")
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	d25 := time.Date(2026, 8, 25, 12, 0, 0, 0, time.UTC)
	d26 := time.Date(2026, 8, 26, 12, 0, 0, 0, time.UTC)
	d27 := time.Date(2026, 8, 27, 12, 0, 0, 0, time.UTC)
	body := msgLine(d25.Unix(), 1) + "\n" + msgLine(d26.Unix(), 2) + "\n" + msgLine(d27.Unix(), 4) + "\n"
	p := mustWrite(t, dir, "s.jsonl", body)
	// The file's own mtime lands on the LAST message's day. If cost were
	// bucketed by mtime instead of per-message timestamp, all $7 would land
	// on Aug 27 and Aug 25/26 would read zero — the exact bug this design
	// avoids (see stats.go's package doc).
	if err := os.Chtimes(p, d27, d27); err != nil {
		t.Fatal(err)
	}

	from := time.Date(2026, 8, 25, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 8, 28, 0, 0, 0, 0, time.UTC)
	st, err := StatsRoot(root, from, to, time.Time{}, time.UTC)
	if err != nil {
		t.Fatal(err)
	}
	if st.Current.Cost != 7 || st.Current.Messages != 3 || st.Current.Sessions != 1 {
		t.Fatalf("current = %+v", st.Current)
	}
	want := map[string]float64{"2026-08-25": 1, "2026-08-26": 2, "2026-08-27": 4}
	if len(st.Series) != 3 {
		t.Fatalf("series len = %d, want 3: %+v", len(st.Series), st.Series)
	}
	for _, db := range st.Series {
		if db.Cost != want[db.Date] {
			t.Fatalf("day %s cost = %v, want %v (series=%+v)", db.Date, db.Cost, want[db.Date], st.Series)
		}
	}
}

func TestStatsRootProviderAttribution(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "--tmp--")
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 8, 25, 12, 0, 0, 0, time.UTC)
	body := `{"type":"model_change","provider":"anthropic","modelId":"claude"}` + "\n" +
		msgLine(now.Unix(), 1) + "\n" +
		`{"type":"model_change","provider":"xai","modelId":"grok"}` + "\n" +
		msgLine(now.Unix(), 2) + "\n"
	mustWrite(t, dir, "s.jsonl", body)

	from := time.Date(2026, 8, 25, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 8, 26, 0, 0, 0, 0, time.UTC)
	st, err := StatsRoot(root, from, to, time.Time{}, time.UTC)
	if err != nil {
		t.Fatal(err)
	}
	if len(st.ByProvider) != 2 {
		t.Fatalf("byProvider = %+v", st.ByProvider)
	}
	byName := map[string]ProviderBucket{}
	for _, pb := range st.ByProvider {
		byName[pb.Provider] = pb
	}
	if byName["anthropic"].Cost != 1 || byName["xai"].Cost != 2 {
		t.Fatalf("byProvider = %+v", st.ByProvider)
	}
}

func TestStatsRootSkipsFilesOlderThanWindow(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "--tmp--")
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	from := time.Date(2026, 8, 25, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 8, 26, 0, 0, 0, 0, time.UTC)
	priorFrom := time.Date(2026, 8, 18, 0, 0, 0, 0, time.UTC)

	// A message timestamp that would land squarely in [from,to) if read —
	// this only proves the skip fires if the file is never opened at all.
	inWindow := time.Date(2026, 8, 25, 12, 0, 0, 0, time.UTC)
	body := msgLine(inWindow.Unix(), 99) + "\n"
	p := mustWrite(t, dir, "s.jsonl", body)
	old := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC) // well before priorFrom
	if err := os.Chtimes(p, old, old); err != nil {
		t.Fatal(err)
	}

	st, err := StatsRoot(root, from, to, priorFrom, time.UTC)
	if err != nil {
		t.Fatal(err)
	}
	if st.Current.Cost != 0 || st.Current.Sessions != 0 {
		t.Fatalf("expected the stale file to be skipped entirely, got %+v", st.Current)
	}
}

func TestStatsRootMissingTimestampFallsBackToMtime(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "--tmp--")
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	body := msgLine(0, 5) + "\n" // no "timestamp" key at all
	p := mustWrite(t, dir, "s.jsonl", body)
	mtimeDay := time.Date(2026, 8, 26, 15, 0, 0, 0, time.UTC)
	if err := os.Chtimes(p, mtimeDay, mtimeDay); err != nil {
		t.Fatal(err)
	}

	from := time.Date(2026, 8, 25, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 8, 28, 0, 0, 0, 0, time.UTC)
	st, err := StatsRoot(root, from, to, time.Time{}, time.UTC)
	if err != nil {
		t.Fatal(err)
	}
	if st.Current.Cost != 5 {
		t.Fatalf("current.cost = %v, want 5 (fallback to mtime)", st.Current.Cost)
	}
	found := false
	for _, db := range st.Series {
		if db.Date == "2026-08-26" && db.Cost == 5 {
			found = true
		} else if db.Cost != 0 {
			t.Fatalf("cost leaked onto %s: %+v", db.Date, db)
		}
	}
	if !found {
		t.Fatalf("expected the mtime-fallback day (2026-08-26) to carry the cost: %+v", st.Series)
	}
}

func TestStatsRootEmptyRoot(t *testing.T) {
	root := t.TempDir()
	from := time.Date(2026, 8, 25, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 8, 26, 0, 0, 0, 0, time.UTC)
	st, err := StatsRoot(root, from, to, time.Time{}, time.UTC)
	if err != nil {
		t.Fatal(err)
	}
	if st.Current.Cost != 0 || st.Current.Messages != 0 || st.Current.Sessions != 0 {
		t.Fatalf("current = %+v", st.Current)
	}
	if len(st.ByProvider) != 0 {
		t.Fatalf("byProvider = %+v", st.ByProvider)
	}
	if len(st.Series) != 1 || st.Series[0].Date != "2026-08-25" || st.Series[0].Cost != 0 {
		t.Fatalf("series = %+v", st.Series)
	}
}

func TestStatsRootAllRangeOmitsPrior(t *testing.T) {
	root := t.TempDir()
	to := time.Date(2026, 8, 26, 0, 0, 0, 0, time.UTC)
	st, err := StatsRoot(root, time.Time{}, to, time.Time{}, time.UTC)
	if err != nil {
		t.Fatal(err)
	}
	if st.Prior != nil {
		t.Fatalf("Prior = %+v, want nil for range=all", st.Prior)
	}
	if len(st.Series) != 0 {
		t.Fatalf("series on an empty all-time root should be empty, got %+v", st.Series)
	}
}
