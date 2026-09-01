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

// assistantLine is a full-fidelity assistant message the way pi >= 0.50
// writes it: provider/model inline, token usage, stopReason, tool calls.
func assistantLine(unixTS int64, provider, model, stop string, cost float64, in, out, cr, cw int, tools ...string) string {
	calls := `{"type":"text","text":"ok"}`
	for i, t := range tools {
		calls += fmt.Sprintf(`,{"type":"toolCall","id":"c%d","name":"%s","arguments":{}}`, i, t)
	}
	return fmt.Sprintf(`{"type":"message","message":{"role":"assistant","provider":"%s","model":"%s","stopReason":"%s","timestamp":%d,"content":[%s],"usage":{"input":%d,"output":%d,"cacheRead":%d,"cacheWrite":%d,"reasoning":3,"cost":{"total":%v}}}}`,
		provider, model, stop, unixTS*1000, calls, in, out, cr, cw, cost)
}

func userLine(unixTS int64) string {
	return fmt.Sprintf(`{"type":"message","message":{"role":"user","timestamp":%d,"content":[{"type":"text","text":"hi"}]}}`, unixTS*1000)
}

func TestStatsRootBreakdowns(t *testing.T) {
	root := t.TempDir()
	dirA := filepath.Join(root, "--a--")
	dirB := filepath.Join(root, "--b--")
	for _, d := range []string{dirA, dirB} {
		if err := os.Mkdir(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	now := time.Date(2026, 8, 25, 12, 0, 0, 0, time.UTC)
	ts := now.Unix()
	// Session 1: model_change says anthropic/claude, but the assistant
	// messages name xai/grok inline — inline wins (no "unknown", no stale
	// provider after a mid-session switch pi didn't log).
	s1 := `{"type":"session","id":"s1","cwd":"/work/a"}` + "\n" +
		`{"type":"session_info","name":"first"}` + "\n" +
		`{"type":"model_change","provider":"anthropic","modelId":"claude"}` + "\n" +
		assistantLine(ts, "xai", "grok", "toolUse", 1, 100, 10, 900, 0, "bash", "read") + "\n" +
		userLine(ts) + "\n" +
		assistantLine(ts, "xai", "grok", "error", 2, 100, 10, 0, 50, "bash") + "\n" +
		`{"type":"compaction","summary":"..."}` + "\n"
	// Session 2: no inline provider at all — falls back to model_change.
	s2 := `{"type":"session","id":"s2","cwd":"/work/b"}` + "\n" +
		`{"type":"model_change","provider":"zai","modelId":"glm"}` + "\n" +
		assistantLine(ts, "", "", "aborted", 4, 0, 0, 0, 0) + "\n"
	// Session 3: same folder as s1, cheapest, no header lines at all.
	s3 := assistantLine(ts, "xai", "grok", "stop", 0.5, 10, 1, 0, 0) + "\n"
	p1 := mustWrite(t, dirA, "s1.jsonl", s1)
	mustWrite(t, dirB, "s2.jsonl", s2)
	mustWrite(t, dirA, "s3.jsonl", s3)
	if err := os.Chtimes(p1, now, now); err != nil {
		t.Fatal(err)
	}

	from := time.Date(2026, 8, 25, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 8, 26, 0, 0, 0, 0, time.UTC)
	st, err := StatsRoot(root, from, to, time.Time{}, time.UTC)
	if err != nil {
		t.Fatal(err)
	}

	if st.Current.Cost != 7.5 || st.Current.Messages != 5 || st.Current.Sessions != 3 {
		t.Fatalf("current = %+v", st.Current)
	}
	// Provider: inline beats model_change, and the user line after the
	// xai reply follows xai (the running truth), not the stale
	// model_change; nothing lands in "unknown" or "anthropic".
	if len(st.ByProvider) != 2 || st.ByProvider[0].Provider != "zai" || st.ByProvider[0].Cost != 4 || st.ByProvider[1].Provider != "xai" || st.ByProvider[1].Cost != 3.5 || st.ByProvider[1].Messages != 4 {
		t.Fatalf("byProvider = %+v", st.ByProvider)
	}
	if len(st.ByModel) != 2 || st.ByModel[0].Model != "glm" || st.ByModel[0].Cost != 4 || st.ByModel[1].Model != "grok" || st.ByModel[1].Messages != 3 {
		t.Fatalf("byModel = %+v", st.ByModel)
	}
	var sum float64
	for _, m := range st.ByModel {
		sum += m.Cost
	}
	if sum != st.Current.Cost {
		t.Fatalf("byModel cost %v != current %v", sum, st.Current.Cost)
	}
	// Workspace: cwd from the session line; s3 (no header) falls back to the
	// folder name, so it is its own bucket rather than silently merging.
	if len(st.ByWorkspace) != 3 || st.ByWorkspace[0].Cwd != "/work/b" || st.ByWorkspace[1].Cwd != "/work/a" || st.ByWorkspace[1].Sessions != 1 || st.ByWorkspace[2].Cwd != "--a--" {
		t.Fatalf("byWorkspace = %+v", st.ByWorkspace)
	}
	if st.Tokens.Input != 210 || st.Tokens.Output != 21 || st.Tokens.CacheRead != 900 || st.Tokens.CacheWrite != 50 || st.Tokens.Reasoning != 12 {
		t.Fatalf("tokens = %+v", st.Tokens)
	}
	if st.Tokens.CacheHit == nil || int(*st.Tokens.CacheHit*100) != 8108 { // 900/(210+900)
		t.Fatalf("cacheHit = %v", st.Tokens.CacheHit)
	}
	if len(st.Tools) != 2 || st.Tools[0].Name != "bash" || st.Tools[0].Calls != 2 || st.Tools[1].Name != "read" {
		t.Fatalf("tools = %+v", st.Tools)
	}
	want := TurnStats{Assistant: 4, User: 1, Errors: 1, Aborted: 1, Compactions: 1}
	if st.Turns != want {
		t.Fatalf("turns = %+v, want %+v", st.Turns, want)
	}
	if len(st.TopSessions) != 3 || st.TopSessions[0].Name != "" || st.TopSessions[0].Cost != 4 || st.TopSessions[1].Name != "first" || st.TopSessions[1].Cost != 3 {
		t.Fatalf("topSessions = %+v", st.TopSessions)
	}
	if st.TopSessions[1].LastAt != now.Format(time.RFC3339) {
		t.Fatalf("lastAt = %q, want %q", st.TopSessions[1].LastAt, now.Format(time.RFC3339))
	}
	if len(st.Series) != 1 || st.Series[0].Turns != 4 || st.Series[0].Messages != 5 {
		t.Fatalf("series = %+v", st.Series)
	}
}

func TestStatsRootTopListsAreCapped(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "--tmp--")
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	ts := time.Date(2026, 8, 25, 12, 0, 0, 0, time.UTC).Unix()
	for i := 0; i < TopSessionsN+3; i++ {
		tools := make([]string, 0, TopTools+2)
		for j := 0; j < TopTools+2; j++ {
			tools = append(tools, fmt.Sprintf("tool%d", j))
		}
		mustWrite(t, dir, fmt.Sprintf("s%d.jsonl", i), assistantLine(ts, "p", "m", "stop", float64(i), 1, 1, 0, 0, tools...)+"\n")
	}
	from := time.Date(2026, 8, 25, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 8, 26, 0, 0, 0, 0, time.UTC)
	st, err := StatsRoot(root, from, to, time.Time{}, time.UTC)
	if err != nil {
		t.Fatal(err)
	}
	if len(st.TopSessions) != TopSessionsN || st.TopSessions[0].Cost != float64(TopSessionsN+2) {
		t.Fatalf("topSessions = %+v", st.TopSessions)
	}
	if len(st.Tools) != TopTools {
		t.Fatalf("tools len = %d", len(st.Tools))
	}
	if st.Current.Sessions != TopSessionsN+3 {
		t.Fatalf("sessions = %d", st.Current.Sessions)
	}
}

func TestStatsRootPriorWindowGetsNoBreakdowns(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "--tmp--")
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	prior := time.Date(2026, 8, 20, 12, 0, 0, 0, time.UTC).Unix()
	cur := time.Date(2026, 8, 25, 12, 0, 0, 0, time.UTC).Unix()
	body := assistantLine(prior, "xai", "grok", "error", 10, 500, 5, 0, 0, "bash") + "\n" +
		assistantLine(cur, "zai", "glm", "stop", 1, 5, 5, 0, 0) + "\n"
	mustWrite(t, dir, "s.jsonl", body)
	from := time.Date(2026, 8, 25, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 8, 26, 0, 0, 0, 0, time.UTC)
	priorFrom := time.Date(2026, 8, 18, 0, 0, 0, 0, time.UTC)
	st, err := StatsRoot(root, from, to, priorFrom, time.UTC)
	if err != nil {
		t.Fatal(err)
	}
	if st.Prior == nil || st.Prior.Cost != 10 || st.Current.Cost != 1 {
		t.Fatalf("prior = %+v current = %+v", st.Prior, st.Current)
	}
	if len(st.ByModel) != 1 || st.ByModel[0].Model != "glm" || st.Tokens.Input != 5 || st.Turns.Errors != 0 || len(st.Tools) != 0 {
		t.Fatalf("prior-window facts leaked into current breakdowns: byModel=%+v tokens=%+v turns=%+v tools=%+v", st.ByModel, st.Tokens, st.Turns, st.Tools)
	}
}

func TestFingerprintTracksAppendsAndNewFiles(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "--tmp--")
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	empty := Fingerprint(root)
	p := mustWrite(t, dir, "s.jsonl", msgLine(0, 1)+"\n")
	one := Fingerprint(root)
	if one == empty {
		t.Fatal("a new file must change the fingerprint")
	}
	if Fingerprint(root) != one {
		t.Fatal("fingerprint must be stable while nothing changes")
	}
	f, err := os.OpenFile(p, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	f.WriteString(msgLine(0, 2) + "\n")
	f.Close()
	if Fingerprint(root) == one {
		t.Fatal("an append must change the fingerprint")
	}
	if Fingerprint("") != "" {
		t.Fatal("empty root fingerprints as empty")
	}
}
