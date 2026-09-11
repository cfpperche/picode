package climetrics

import (
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/cfpperche/picode/internal/clisession"

	_ "modernc.org/sqlite"
)

// --- cache ------------------------------------------------------------

func TestCacheReparsesOnlyWhatChanged(t *testing.T) {
	files.reset()
	dir := t.TempDir()
	path := filepath.Join(dir, "s.jsonl")
	if err := os.WriteFile(path, []byte("a\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	calls := 0
	parse := func(string) *parsed { calls++; return &parsed{} }

	cachedParse(path, parse)
	cachedParse(path, parse)
	if calls != 1 {
		t.Fatalf("parsed %d times, want 1 — an unchanged file must be free", calls)
	}

	// Appending changes size and mtime, which is what a live session does.
	if err := os.WriteFile(path, []byte("a\nb\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cachedParse(path, parse)
	if calls != 2 {
		t.Fatalf("parsed %d times, want 2 — a changed file must be re-read", calls)
	}
}

func TestCacheEvictsToItsBudget(t *testing.T) {
	c := &parseCache{max: 10}
	dir := t.TempDir()
	for i := 0; i < 8; i++ {
		p := &parsed{ents: make([]guestEntry, 4)}
		c.put(filepath.Join(dir, itoa(i)), 1, int64(i), p)
	}
	if c.entries > c.max+4 {
		t.Fatalf("cache holds %d entries against a budget of %d", c.entries, c.max)
	}
	if len(c.m) == 0 {
		t.Fatal("eviction emptied the cache instead of trimming it")
	}
}

// TestCacheKeysAMultiFileParseOnEveryFileItSpans is the contract grok's
// session parse rides on: the parse spans summary.json, events.jsonl and
// usage.json, and the cache key must move when any of them does — a turn
// appends to events.jsonl while it runs and writes usage.json when it ends.
func TestCacheKeysAMultiFileParseOnEveryFileItSpans(t *testing.T) {
	files.reset()
	dir := t.TempDir()
	summary := filepath.Join(dir, "summary.json")
	events := filepath.Join(dir, "events.jsonl")
	if err := os.WriteFile(summary, []byte(`{"info":{"id":"g1"}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(events, []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}
	calls := 0
	parse := func() *parsed { calls++; return &parsed{} }

	cachedParseKeyed(statKey(summary, events), parse)
	cachedParseKeyed(statKey(summary, events), parse)
	if calls != 1 {
		t.Fatalf("parsed %d times, want 1 — unchanged files must be free", calls)
	}

	f, err := os.OpenFile(events, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString("{\"ts\":\"x\"}\n"); err != nil {
		f.Close()
		t.Fatal(err)
	}
	f.Close()
	cachedParseKeyed(statKey(summary, events), parse)
	if calls != 2 {
		t.Fatalf("parsed %d times, want 2 — an appended timeline is a new parse", calls)
	}

	// A file that appears later must not be answered by the parse cached
	// while it was absent.
	usage := filepath.Join(dir, "usage.json")
	cachedParseKeyed(statKey(summary, events, usage), parse)
	if calls != 3 {
		t.Fatalf("parsed %d times, want 3 — a file that appears is a new parse", calls)
	}
}

// --- codex ------------------------------------------------------------

func withCodexRoot(t *testing.T) string {
	t.Helper()
	files.reset()
	dir := t.TempDir()
	old := clisession.CodexTestRoot
	clisession.CodexTestRoot = dir
	t.Cleanup(func() { clisession.CodexTestRoot = old })
	return dir
}

func writeRollout(t *testing.T, root, day string, lines []map[string]any) {
	t.Helper()
	parts := strings.Split(day, "-")
	dir := filepath.Join(root, parts[0], parts[1], parts[2])
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	var b strings.Builder
	for _, l := range lines {
		raw, _ := json.Marshal(l)
		b.Write(raw)
		b.WriteByte('\n')
	}
	if err := os.WriteFile(filepath.Join(dir, "rollout-"+day+".jsonl"), []byte(b.String()), 0o644); err != nil {
		t.Fatal(err)
	}
}

func codexTurn(ts string, in, out int64) map[string]any {
	return map[string]any{
		"timestamp": ts, "type": "event_msg",
		"payload": map[string]any{
			"type": "token_count",
			"info": map[string]any{
				"last_token_usage": map[string]any{
					"input_tokens": in, "output_tokens": out,
					"cached_input_tokens": 0, "reasoning_output_tokens": 0,
				},
			},
		},
	}
}

func TestCodexCountsTokensButNeverCost(t *testing.T) {
	root := withCodexRoot(t)
	writeRollout(t, root, "2026-09-06", []map[string]any{
		{"timestamp": day(1), "type": "session_meta",
			"payload": map[string]any{"id": "s1", "cwd": "/repo", "model_provider": "openai"}},
		{"timestamp": day(1), "type": "turn_context", "payload": map[string]any{"model": "gpt-5.6"}},
		codexTurn(day(1), 1000, 200),
	})

	w, err := CodexMeter{}.Meter(req(ScopeMachine, 7))
	if err != nil {
		t.Fatal(err)
	}
	if w.Stats.Tokens.Input != 1000 || w.Stats.Tokens.Output != 200 {
		t.Fatalf("tokens = %+v", w.Stats.Tokens)
	}
	if w.Stats.Current.Cost != 0 {
		t.Fatalf("cost = %v — Codex must never be priced", w.Stats.Current.Cost)
	}
	if w.Coverage.Signals[SigCost] != StateNotReported {
		t.Fatalf("cost coverage = %v, want not-reported so the surface shows a dash", w.Coverage.Signals[SigCost])
	}
}

// TestCodexUsesTheTurnNotTheRunningTotal guards the field choice: info
// carries both a cumulative total and the turn that just ended, and reading
// the cumulative one would multiply every session's tokens by its turn count.
func TestCodexUsesTheTurnNotTheRunningTotal(t *testing.T) {
	root := withCodexRoot(t)
	writeRollout(t, root, "2026-09-06", []map[string]any{
		{"timestamp": day(1), "type": "session_meta", "payload": map[string]any{"id": "s1", "cwd": "/repo"}},
		codexTurn(day(1), 100, 10),
		codexTurn(day(1), 100, 10),
		codexTurn(day(1), 100, 10),
	})
	w, err := CodexMeter{}.Meter(req(ScopeMachine, 7))
	if err != nil {
		t.Fatal(err)
	}
	if w.Stats.Tokens.Input != 300 {
		t.Fatalf("input = %d, want 300 (three turns of 100)", w.Stats.Tokens.Input)
	}
}

func TestCodexReportsQuotaWindows(t *testing.T) {
	root := withCodexRoot(t)
	reset := time.Date(2026, 9, 13, 12, 0, 0, 0, time.UTC).Unix()
	writeRollout(t, root, "2026-09-06", []map[string]any{
		{"timestamp": day(1), "type": "session_meta", "payload": map[string]any{"id": "s1", "cwd": "/repo"}},
		{"timestamp": day(1), "type": "event_msg", "payload": map[string]any{
			"type": "token_count",
			"rate_limits": map[string]any{
				"plan_type": "prolite",
				"primary":   map[string]any{"used_percent": 81.0, "window_minutes": 10080, "resets_at": reset},
			},
		}},
	})
	w, err := CodexMeter{}.Meter(req(ScopeMachine, 7))
	if err != nil {
		t.Fatal(err)
	}
	if len(w.Limits) != 1 {
		t.Fatalf("limits = %+v", w.Limits)
	}
	got := w.Limits[0]
	if got.Label != "weekly" || got.UsedPercent != 81 || got.Plan != "prolite" || got.ResetsAt == "" {
		t.Fatalf("limit = %+v", got)
	}
	if w.Coverage.Signals[SigLimits] != StateReported {
		t.Fatal("a quota reading must show as reported")
	}
}

// TestCodexPrunesDayDirectories is the reason the 2.24 GB tree is
// affordable: a window opens the days it covers, not the archive.
func TestCodexPrunesDayDirectories(t *testing.T) {
	root := withCodexRoot(t)
	writeRollout(t, root, "2020-01-01", []map[string]any{
		{"timestamp": "2020-01-01T10:00:00Z", "type": "session_meta", "payload": map[string]any{"id": "old", "cwd": "/repo"}},
	})
	writeRollout(t, root, "2026-09-06", []map[string]any{
		{"timestamp": day(1), "type": "session_meta", "payload": map[string]any{"id": "new", "cwd": "/repo"}},
	})
	r := req(ScopeMachine, 7)
	got := codexFiles(root, r)
	for _, p := range got {
		if strings.Contains(p, "2020") {
			t.Fatalf("a 7-day window walked into 2020: %v", got)
		}
	}
	if len(got) != 1 {
		t.Fatalf("files = %v, want just the in-range day", got)
	}
}

func TestCodexDirInRangeKeepsPartialPaths(t *testing.T) {
	from := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 9, 8, 0, 0, 0, 0, time.UTC)
	for _, c := range []struct {
		dir  string
		want bool
	}{
		{"2026", true},
		{"2026/09", true},
		{"2026/09/04", true},
		{"2025", false},
		{"2026/01", false},
		{"weird", true}, // unreadable segment: never pruned on a guess
	} {
		if got := codexDirInRange("/root", "/root/"+c.dir, from, to); got != c.want {
			t.Fatalf("codexDirInRange(%q) = %v, want %v", c.dir, got, c.want)
		}
	}
}

// --- opencode / hermes ------------------------------------------------

func newTestDB(t *testing.T, path string, stmts ...string) {
	t.Helper()
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	for _, s := range stmts {
		if _, err := db.Exec(s); err != nil {
			t.Fatalf("%s: %v", s, err)
		}
	}
}

func TestOpenCodeReadsCostStraightOffTheRow(t *testing.T) {
	files.reset()
	path := filepath.Join(t.TempDir(), "opencode.db")
	at := fixtureNow.AddDate(0, 0, -1).UnixMilli()
	data, _ := json.Marshal(map[string]any{
		"role": "assistant", "cost": 1.25, "modelID": "glm-5.3-flash", "providerID": "zai",
		"tokens": map[string]any{"input": 900, "output": 100, "cache": map[string]any{"read": 10, "write": 0}},
	})
	newTestDB(t, path,
		`CREATE TABLE session (id TEXT PRIMARY KEY, directory TEXT, summary_additions INT, summary_deletions INT, time_updated INT)`,
		`CREATE TABLE message (id TEXT PRIMARY KEY, session_id TEXT, time_created INT, data TEXT)`,
		`INSERT INTO session VALUES ('s1','/repo',40,5,`+itoa64(at)+`)`,
		`INSERT INTO message VALUES ('m1','s1',`+itoa64(at)+`,'`+string(data)+`')`,
	)
	old := clisession.OpenCodeTestDB
	clisession.OpenCodeTestDB = path
	t.Cleanup(func() { clisession.OpenCodeTestDB = old })

	w, err := OpenCodeMeter{}.Meter(req(ScopeMachine, 7))
	if err != nil {
		t.Fatal(err)
	}
	if !approx(w.Stats.Current.Cost, 1.25) {
		t.Fatalf("cost = %v, want 1.25 straight off the row", w.Stats.Current.Cost)
	}
	if w.Coverage.Signals[SigCost] != StateReported {
		t.Fatal("OpenCode prices every message; its cost is not partial")
	}
	if w.Impact == nil || w.Impact.LinesAdded != 40 {
		t.Fatalf("impact = %+v", w.Impact)
	}
	if w.Stats.ByModel[0].Provider != "zai" {
		t.Fatalf("byModel = %+v", w.Stats.ByModel)
	}
}

func TestHermesSpreadsSessionTotalsOverItsMessages(t *testing.T) {
	files.reset()
	path := filepath.Join(t.TempDir(), "state.db")
	inWin := float64(fixtureNow.AddDate(0, 0, -1).Unix())
	outWin := float64(fixtureNow.AddDate(0, 0, -40).Unix())
	newTestDB(t, path,
		`CREATE TABLE sessions (id TEXT PRIMARY KEY, cwd TEXT, model TEXT, title TEXT,
			billing_mode TEXT, actual_cost_usd REAL, estimated_cost_usd REAL,
			input_tokens INT, output_tokens INT, cache_read_tokens INT, cache_write_tokens INT, reasoning_tokens INT)`,
		`CREATE TABLE messages (id INTEGER PRIMARY KEY, session_id TEXT, role TEXT, tool_name TEXT,
			timestamp REAL, token_count INT, finish_reason TEXT)`,
		`INSERT INTO sessions VALUES ('h1','/repo','hermes-1','Title','subscription',10.0,0,1000,0,0,0,0)`,
		`INSERT INTO messages VALUES (1,'h1','assistant','bash',`+ftoa(inWin)+`,500,'stop')`,
		`INSERT INTO messages VALUES (2,'h1','assistant','',`+ftoa(outWin)+`,500,'stop')`,
	)
	old := clisession.HermesTestDB
	clisession.HermesTestDB = path
	t.Cleanup(func() { clisession.HermesTestDB = old })

	w, err := HermesMeter{}.Meter(req(ScopeMachine, 7))
	if err != nil {
		t.Fatal(err)
	}
	// Half the session's tokens fall in the window, so half its cost does.
	if !approx(w.Stats.Current.Cost, 5.0) {
		t.Fatalf("cost = %v, want 5.00 — half the session is outside the window", w.Stats.Current.Cost)
	}
	// Hermes states its own billing mode, which outranks the operator's setting.
	if w.Coverage.Billing != BillingSubscription {
		t.Fatalf("billing = %v, want subscription from Hermes' own column", w.Coverage.Billing)
	}
	if len(w.Stats.Tools) != 1 || w.Stats.Tools[0].Name != "bash" {
		t.Fatalf("tools = %+v", w.Stats.Tools)
	}
}

func TestHermesBillingWordsMapOrStayOut(t *testing.T) {
	for word, want := range map[string]Billing{
		"subscription": BillingSubscription,
		"PLAN":         BillingSubscription,
		"api_key":      BillingAPI,
		"metered":      BillingAPI,
		"something":    "", // unrecognised: leave the operator's setting alone
		"":             "",
	} {
		if got := hermesBilling(word); got != want {
			t.Fatalf("hermesBilling(%q) = %q, want %q", word, got, want)
		}
	}
}

// --- grok -------------------------------------------------------------

// withGrokRoot points GROK_HOME at a fresh tree.
func withGrokRoot(t *testing.T) string {
	t.Helper()
	files.reset()
	home := t.TempDir()
	t.Setenv("GROK_HOME", home)
	return home
}

// grokStamp is an RFC3339 timestamp n seconds after the fixture's day-old
// instant, so every event inside one fixture shares a clock.
func grokStamp(n int) string {
	return fixtureNow.AddDate(0, 0, -1).Add(time.Duration(n) * time.Second).Format(time.RFC3339Nano)
}

func grokWrite(t *testing.T, dir, name, body string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

// grokEventLine is one events.jsonl record.
func grokEventLine(n int, typ, extra string) string {
	if extra != "" {
		extra = "," + extra
	}
	return `{"ts":"` + grokStamp(n) + `","type":"` + typ + `"` + extra + "}\n"
}

// grokTurnEvents is the one turn every fixture shares: two model loops (4s
// and 3s), one tool of 500ms that failed, and a 9s turn window.
func grokTurnEvents() string {
	return grokEventLine(1, "turn_started", `"turn_number":0`) +
		grokEventLine(2, "loop_started", `"loop_index":0`) +
		grokEventLine(4, "first_token", "") +
		grokEventLine(6, "tool_started", `"tool_name":"run_terminal_command"`) +
		grokEventLine(6, "tool_completed", `"tool_name":"run_terminal_command","duration_ms":500,"outcome":"error"`) +
		grokEventLine(7, "loop_started", `"loop_index":1`) +
		grokEventLine(8, "first_token", "") +
		grokEventLine(10, "turn_ended", `"outcome":"completed"`)
}

// grokTurnEvents2 is a second turn ten seconds later, with no tool.
func grokTurnEvents2() string {
	return grokEventLine(21, "turn_started", `"turn_number":1`) +
		grokEventLine(22, "loop_started", `"loop_index":0`) +
		grokEventLine(23, "first_token", "") +
		grokEventLine(30, "turn_ended", `"outcome":"completed"`)
}

// grokUsage is what usage.json holds once Grok has priced a turn: 1e9 ticks
// is $0.10 at its own 10^10 ticks per USD.
func grokUsage(endStamp string) string {
	return `{"sessionId":"g1","updatedAt":"` + endStamp + `",` +
		`"session":{"primaryModelId":"grok-4.6"},` +
		`"turns":[{"turnNumber":1,"endedAt":"` + endStamp + `",` +
		`"inputTokens":1000,"outputTokens":250,"cachedReadTokens":500,` +
		`"cacheCreationTokens":0,"reasoningTokens":50,"totalTokens":1800,` +
		`"modelCalls":2,"costUsdTicks":1000000000,"primaryModelId":"grok-4.6"}]}`
}

const grokSummaryJSON = `{"info":{"id":"g1","cwd":"/repo"},"session_summary":"Ship it","current_model_id":"grok-4.6"}`

func TestGrokReadsPromptsTurnsToolsAndUsage(t *testing.T) {
	home := withGrokRoot(t)
	folder := filepath.Join(home, "sessions", "%2Frepo")
	grokWrite(t, folder, "prompt_history.jsonl",
		`{"timestamp":"`+grokStamp(0)+`","session_id":"g1","prompt":"SECRETPROMPT"}`+"\n")
	sess := filepath.Join(folder, "g1")
	grokWrite(t, sess, "summary.json", grokSummaryJSON)
	grokWrite(t, sess, "events.jsonl", grokTurnEvents())
	grokWrite(t, sess, "usage.json", grokUsage(grokStamp(10)))

	w, err := GrokMeter{}.Meter(req(ScopeMachine, 7))
	if err != nil {
		t.Fatal(err)
	}
	if w.Stats.Current.Messages != 2 {
		t.Fatalf("messages = %d, want 2 (one prompt, one turn)", w.Stats.Current.Messages)
	}
	if w.Stats.Turns.Assistant != 1 || w.Stats.Turns.User != 1 {
		t.Fatalf("turns = %+v, want one assistant and one user", w.Stats.Turns)
	}
	if got := w.Stats.Tokens; got.Input != 1000 || got.Output != 250 || got.CacheRead != 500 || got.Reasoning != 50 {
		t.Fatalf("tokens = %+v", got)
	}
	if !approx(w.Stats.Current.Cost, 0.10) {
		t.Fatalf("cost = %v, want 0.10 from costUsdTicks", w.Stats.Current.Cost)
	}
	// Model time is loop_started→tool_started (4s) plus loop_started→turn_ended
	// (3s); the tool is its own duration, and the turn window is 9s.
	if w.Timing == nil || w.Timing.APIMs != 7000 || w.Timing.ToolMs != 500 || w.Timing.SessionMs != 9000 {
		t.Fatalf("timing = %+v", w.Timing)
	}
	if len(w.Stats.Tools) != 1 || w.Stats.Tools[0].Name != "run_terminal_command" || w.Stats.Tools[0].Calls != 1 {
		t.Fatalf("tools = %+v", w.Stats.Tools)
	}
	if w.Stats.Turns.Errors != 1 {
		t.Fatalf("errors = %d, want the failed tool completion", w.Stats.Turns.Errors)
	}
	if len(w.Stats.ByModel) != 1 || w.Stats.ByModel[0].Provider != "xai" || w.Stats.ByModel[0].Model != "grok-4.6" {
		t.Fatalf("byModel = %+v", w.Stats.ByModel)
	}
	if len(w.Stats.TopSessions) != 1 || w.Stats.TopSessions[0].Name != "Ship it" {
		t.Fatalf("topSessions = %+v, want the session title summary.json carries", w.Stats.TopSessions)
	}
	for _, sig := range []Signal{SigCost, SigTokens, SigModel, SigMessages, SigTurns, SigTools, SigErrors, SigTiming} {
		if w.Coverage.Signals[sig] != StateReported {
			t.Fatalf("%s coverage = %v, want reported: %+v", sig, w.Coverage.Signals[sig], w.Coverage.Signals)
		}
	}
	if w.Coverage.Signals[SigImpact] != StateNotReported || w.Coverage.Signals[SigLimits] != StateNotReported {
		t.Fatalf("grok counts no lines and no quota window: %+v", w.Coverage.Signals)
	}
	blob, _ := json.Marshal(w.Stats)
	if strings.Contains(string(blob), "SECRETPROMPT") {
		t.Fatalf("prompt text reached the payload: %s", blob)
	}
	// The url-encoded folder decodes back to a real path, so a scoped
	// window can claim it.
	if len(w.Stats.ByWorkspace) != 1 || w.Stats.ByWorkspace[0].Cwd != "/repo" {
		t.Fatalf("byWorkspace = %+v", w.Stats.ByWorkspace)
	}
}

func TestGrokPromptsAloneSayWhatIsMissing(t *testing.T) {
	home := withGrokRoot(t)
	folder := filepath.Join(home, "sessions", "%2Frepo")
	grokWrite(t, folder, "prompt_history.jsonl",
		`{"timestamp":"`+grokStamp(0)+`","session_id":"g1","prompt":"SECRETPROMPT"}`+"\n")

	w, err := GrokMeter{}.Meter(req(ScopeMachine, 7))
	if err != nil {
		t.Fatal(err)
	}
	if w.Stats.Current.Messages != 1 {
		t.Fatalf("messages = %d, want 1", w.Stats.Current.Messages)
	}
	if w.Coverage.Signals[SigMessages] != StateReported {
		t.Fatal("prompt counts are real and must say so")
	}
	if w.Coverage.Signals[SigTokens] != StateNotReported || w.Coverage.Signals[SigCost] != StateNotReported {
		t.Fatalf("a folder with no session directory has nothing to price: %+v", w.Coverage.Signals)
	}
}

func TestGrokWithoutUsageSaysTokensAreMissing(t *testing.T) {
	home := withGrokRoot(t)
	sess := filepath.Join(home, "sessions", "%2Frepo", "g1")
	grokWrite(t, sess, "summary.json", grokSummaryJSON)
	grokWrite(t, sess, "events.jsonl", grokTurnEvents())

	w, err := GrokMeter{}.Meter(req(ScopeMachine, 7))
	if err != nil {
		t.Fatal(err)
	}
	// The timeline is real: turns, tools and durations exist without usage.json.
	if w.Stats.Turns.Assistant != 1 || len(w.Stats.Tools) != 1 {
		t.Fatalf("turns/tools = %+v / %+v", w.Stats.Turns, w.Stats.Tools)
	}
	if w.Timing == nil || w.Timing.APIMs != 7000 || w.Timing.ToolMs != 500 {
		t.Fatalf("timing = %+v — durations must survive a session with no tokens", w.Timing)
	}
	if w.Coverage.Signals[SigTurns] != StateReported || w.Coverage.Signals[SigTools] != StateReported || w.Coverage.Signals[SigTiming] != StateReported {
		t.Fatalf("timeline coverage = %+v", w.Coverage.Signals)
	}
	if w.Coverage.Signals[SigTokens] != StateNotReported || w.Coverage.Signals[SigCost] != StateNotReported {
		t.Fatalf("tokens/cost must read as unmeasured, not zero: %+v", w.Coverage.Signals)
	}
}

func TestGrokPartialUsageNamesBothCounts(t *testing.T) {
	home := withGrokRoot(t)
	sess := filepath.Join(home, "sessions", "%2Frepo", "g1")
	grokWrite(t, sess, "summary.json", grokSummaryJSON)
	grokWrite(t, sess, "events.jsonl", grokTurnEvents()+grokTurnEvents2())
	grokWrite(t, sess, "usage.json", grokUsage(grokStamp(10)))

	w, err := GrokMeter{}.Meter(req(ScopeMachine, 7))
	if err != nil {
		t.Fatal(err)
	}
	if w.Stats.Turns.Assistant != 2 {
		t.Fatalf("turns = %d, want both turns even though only one is priced", w.Stats.Turns.Assistant)
	}
	if w.Coverage.Signals[SigTokens] != StatePartial || w.Coverage.Signals[SigCost] != StatePartial {
		t.Fatalf("tokens/cost coverage = %+v, want partial", w.Coverage.Signals)
	}
	if !strings.Contains(w.Coverage.Note, "1 of 2") {
		t.Fatalf("note must name both counts: %q", w.Coverage.Note)
	}
}

func TestGrokUsageAloneStillCountsTheTurn(t *testing.T) {
	home := withGrokRoot(t)
	sess := filepath.Join(home, "sessions", "%2Frepo", "g1")
	grokWrite(t, sess, "summary.json", grokSummaryJSON)
	grokWrite(t, sess, "usage.json", grokUsage(grokStamp(10)))

	w, err := GrokMeter{}.Meter(req(ScopeMachine, 7))
	if err != nil {
		t.Fatal(err)
	}
	if w.Stats.Turns.Assistant != 1 || w.Stats.Tokens.Output != 250 {
		t.Fatalf("turns/tokens = %d / %+v", w.Stats.Turns.Assistant, w.Stats.Tokens)
	}
	if w.Timing != nil {
		t.Fatalf("timing = %+v, want none: no events file was written", w.Timing)
	}
	if w.Coverage.Signals[SigTiming] != StateNotReported {
		t.Fatalf("timing coverage = %v", w.Coverage.Signals[SigTiming])
	}
}

func TestAbsentCLIStillGetsARow(t *testing.T) {
	files.reset()
	old := clisession.HermesTestDB
	clisession.HermesTestDB = filepath.Join(t.TempDir(), "never-installed.db")
	t.Cleanup(func() { clisession.HermesTestDB = old })

	w, err := HermesMeter{}.Meter(req(ScopeMachine, 7))
	if err != nil {
		t.Fatalf("an uninstalled CLI is not an error: %v", err)
	}
	if w.Coverage.CLI != "hermes" || w.Coverage.Note == "" {
		t.Fatalf("coverage = %+v, want a row that explains itself", w.Coverage)
	}
}

// ftoa writes a float the way SQLite will read it back. An earlier version
// trimmed trailing zeros and turned 1788696000 into 1788696 — a 1970
// timestamp that silently emptied the window.
func ftoa(f float64) string { return strconv.FormatFloat(f, 'f', -1, 64) }
