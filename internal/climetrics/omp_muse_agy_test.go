package climetrics

import (
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/cfpperche/picode/internal/clisession"

	_ "modernc.org/sqlite"
)

// These three meters are what the "What each CLI reports" panel was
// missing: omp, muse and agy had session listings but no coverage row, so
// the panel silently understated the fleet. Fixtures follow the real-store
// shapes quoted in the meter docs (omp schema v3 with usage/cost,
// muse export_schema_version 1 envelopes, the agy conversation index).

func writeLines(t *testing.T, path string, lines []string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
}

func withOmpRoot(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	old := clisession.OmpTestRoot
	clisession.OmpTestRoot = dir
	t.Cleanup(func() { clisession.OmpTestRoot = old })
	return dir
}

func withMuseDB(t *testing.T, db string) {
	t.Helper()
	old := clisession.MuseTestDB
	clisession.MuseTestDB = db
	t.Cleanup(func() { clisession.MuseTestDB = old })
}

func withAgyDB(t *testing.T, db string) {
	t.Helper()
	old := clisession.AgyTestDB
	clisession.AgyTestDB = db
	t.Cleanup(func() { clisession.AgyTestDB = old })
}

// --- omp ---------------------------------------------------------------

func ompFixture() []string {
	enc := func(v any) string {
		raw, err := json.Marshal(v)
		if err != nil {
			panic(err)
		}
		return string(raw)
	}
	ms := int64(1788652800000) // 2026-09-01T00:00:00Z
	return []string{
		enc(map[string]any{"type": "session", "version": 3, "id": "s1", "timestamp": "2026-09-01T00:00:00Z", "cwd": "/repo"}),
		enc(map[string]any{"type": "title", "title": "Fix it"}),
		enc(map[string]any{"type": "model_change", "model": "zai/glm-5.3-flash"}),
		enc(map[string]any{"type": "message", "message": map[string]any{
			"role": "user", "timestamp": ms, "content": []any{map[string]any{"type": "text", "text": "hi"}},
		}}),
		enc(map[string]any{"type": "message", "message": map[string]any{
			"role": "assistant", "provider": "zai", "model": "glm-5.3-flash",
			"stopReason": "stop", "timestamp": ms + 1000, "duration": 7288.5,
			"usage": map[string]any{
				"input": 100, "output": 20, "cacheRead": 10, "cacheWrite": 5, "reasoningTokens": 7,
				"cost": map[string]any{"input": 0.001, "output": 0.0002, "cacheRead": 0.0001, "cacheWrite": 0.00005, "total": 0.00135},
			},
			"content": []any{
				map[string]any{"type": "text", "text": "done"},
				map[string]any{"type": "toolCall", "name": "Bash"},
			},
		}}),
	}
}

func TestOmpParseReadsCostTokensToolsTiming(t *testing.T) {
	files.reset()
	dir := t.TempDir()
	path := filepath.Join(dir, "-repo", "s.jsonl")
	writeLines(t, path, ompFixture())

	p := ompParse(path, time.Now())
	if len(p.ents) != 2 {
		t.Fatalf("entries = %d, want 2", len(p.ents))
	}
	var asst *cliEntry
	for i := range p.ents {
		if p.ents[i].role == "assistant" {
			asst = &p.ents[i]
		}
	}
	if asst == nil {
		t.Fatal("no assistant entry")
	}
	if asst.prov != "zai" || asst.model != "glm-5.3-flash" {
		t.Fatalf("model = %s/%s, want zai/glm-5.3-flash", asst.prov, asst.model)
	}
	if asst.cost != 0.00135 {
		t.Fatalf("cost = %v, want 0.00135", asst.cost)
	}
	if asst.toks.Input != 100 || asst.toks.CacheRead != 10 || asst.toks.Reasoning != 7 {
		t.Fatalf("tokens = %+v, want input=100 cacheRead=10 reasoning=7", asst.toks)
	}
	if asst.split.CacheWrite != 0.00005 {
		t.Fatalf("split = %+v, want cacheWrite=0.00005", asst.split)
	}
	if len(asst.tools) != 1 || asst.tools[0] != "Bash" {
		t.Fatalf("tools = %v, want [Bash]", asst.tools)
	}
	if p.timing.SessionMs != 7288 {
		t.Fatalf("sessionMs = %d, want 7288", p.timing.SessionMs)
	}
}

func TestOmpMeterWindow(t *testing.T) {
	files.reset()
	root := withOmpRoot(t)
	writeLines(t, filepath.Join(root, "-repo", "s.jsonl"), ompFixture())

	w, err := OmpMeter{}.Meter(realReq())
	if err != nil {
		t.Fatal(err)
	}
	if w.Stats.Current.Messages != 2 {
		t.Fatalf("messages = %d, want 2", w.Stats.Current.Messages)
	}
	if w.Stats.Current.Cost != 0.00135 {
		t.Fatalf("cost = %v, want 0.00135", w.Stats.Current.Cost)
	}
	if w.Stats.Turns.Assistant != 1 || w.Stats.Turns.User != 1 {
		t.Fatalf("turns = %+v, want 1 assistant 1 user", w.Stats.Turns)
	}
	if w.Timing == nil || w.Timing.SessionMs != 7288 {
		t.Fatalf("timing = %+v, want sessionMs=7288", w.Timing)
	}
	sig := w.Coverage.Signals
	if sig[SigCost] != StateReported || sig[SigTiming] != StateReported {
		t.Fatalf("signals = %v, want cost+timing reported", sig)
	}
	if sig[SigImpact] != StateNotReported || sig[SigLimits] != StateNotReported {
		t.Fatalf("signals = %v, want impact+limits not-reported", sig)
	}
}

func TestOmpMeterAbsentWithoutStore(t *testing.T) {
	withOmpRoot(t) // empty temp dir exists but holds no buckets: no sessions, still present
	// Point at a root that does not exist at all.
	old := clisession.OmpTestRoot
	clisession.OmpTestRoot = filepath.Join(old, "missing")
	defer func() { clisession.OmpTestRoot = old }()

	w, err := OmpMeter{}.Meter(realReq())
	if err != nil {
		t.Fatal(err)
	}
	if w.Stats.Current.Messages != 0 {
		t.Fatalf("messages = %d, want 0", w.Stats.Current.Messages)
	}
	if w.Coverage.Signals[SigCost] != StateUnavailable {
		t.Fatalf("cost = %v, want unavailable for a missing store", w.Coverage.Signals[SigCost])
	}
}

// --- muse --------------------------------------------------------------

func museFixture() []string {
	enc := func(v any) string {
		raw, err := json.Marshal(v)
		if err != nil {
			panic(err)
		}
		return string(raw)
	}
	us := int64(1788652800000000) // 2026-09-01T00:00:00Z in micros
	meta := enc(map[string]any{
		"schema_version": 1, "recorded_at": us, "record_type": "event",
		"payload_type": "runtime.session",
		"payload": map[string]any{
			"kind":   "metadata",
			"record": map[string]any{"workspace_root": "/repo", "model_id": "muse-spark"},
		},
	})
	user := enc(map[string]any{
		"schema_version": 1, "recorded_at": us + 1000, "record_type": "event",
		"payload_type": "runtime.user_intent.accepted",
		"payload":      map[string]any{"model_messages": []any{}},
	})
	asst := enc(map[string]any{
		"schema_version": 1, "recorded_at": us + 2000, "record_type": "event",
		"payload_type": "runtime.session",
		"payload": map[string]any{
			"kind":  "run",
			"event": map[string]any{"kind": "assistant_message_committed", "text": "done"},
		},
	})
	tools := enc(map[string]any{
		"schema_version": 1, "recorded_at": us + 3000, "record_type": "event",
		"payload_type": "runtime.session",
		"payload": map[string]any{
			"kind":  "run",
			"event": map[string]any{"kind": "assistant_tool_calls_committed", "tool_calls": []any{map[string]any{"call_id": "c1", "name": "Read"}}},
		},
	})
	// One record arrives top-level, the rest wrapped in a retained frame's
	// children — both framings occur in real logs.
	frame := enc(map[string]any{
		"retained_frame": "x",
		"children": []any{
			map[string]any{"record_json": user},
			map[string]any{"record_json": asst},
			map[string]any{"record_json": tools},
		},
	})
	return []string{meta, frame}
}

func TestMuseParseReadsEnvelopes(t *testing.T) {
	files.reset()
	dir := t.TempDir()
	path := filepath.Join(dir, "2026", "09", "01", "sid-1", "session.jsonl")
	writeLines(t, path, museFixture())

	p := museParse(path, time.Now())
	if len(p.ents) != 2 {
		t.Fatalf("entries = %d, want 1 user + 1 assistant", len(p.ents))
	}
	var user, asst *cliEntry
	for i := range p.ents {
		switch p.ents[i].role {
		case "user":
			user = &p.ents[i]
		case "assistant":
			asst = &p.ents[i]
		}
	}
	if user == nil || asst == nil {
		t.Fatalf("roles = %v, want user+assistant", p.ents)
	}
	if asst.model != "muse-spark" || asst.cwd != "/repo" {
		t.Fatalf("assistant = %+v, want model muse-spark in /repo", asst)
	}
	// The tool call event follows the message that made it, so it attaches
	// back to that turn rather than forward to the next one.
	if len(asst.tools) != 1 || asst.tools[0] != "Read" {
		t.Fatalf("tools = %v, want [Read]", asst.tools)
	}
	if p.key != "sid-1" {
		t.Fatalf("key = %q, want the session dir name", p.key)
	}
}

func TestMuseMeterWindow(t *testing.T) {
	files.reset()
	dir := t.TempDir()
	withMuseDB(t, filepath.Join(dir, "session-index.db"))
	writeLines(t, filepath.Join(dir, "sessions", "2026", "09", "01", "sid-1", "session.jsonl"), museFixture())

	w, err := MuseMeter{}.Meter(realReq())
	if err != nil {
		t.Fatal(err)
	}
	if w.Stats.Current.Messages != 2 {
		t.Fatalf("messages = %d, want 2", w.Stats.Current.Messages)
	}
	if w.Stats.Turns.Assistant != 1 {
		t.Fatalf("turns = %+v, want 1 assistant", w.Stats.Turns)
	}
	sig := w.Coverage.Signals
	if sig[SigTurns] != StateReported || sig[SigTools] != StateReported || sig[SigModel] != StateReported {
		t.Fatalf("signals = %v, want turns+tools+model reported", sig)
	}
	if sig[SigCost] != StateNotReported || sig[SigTokens] != StateNotReported {
		t.Fatalf("signals = %v, want cost+tokens not-reported", sig)
	}
}

func TestMuseMeterAbsentWithoutStore(t *testing.T) {
	withMuseDB(t, filepath.Join(t.TempDir(), "missing", "session-index.db"))
	w, err := MuseMeter{}.Meter(realReq())
	if err != nil {
		t.Fatal(err)
	}
	if w.Coverage.Signals[SigMessages] != StateUnavailable {
		t.Fatalf("messages = %v, want unavailable for a missing store", w.Coverage.Signals[SigMessages])
	}
}

// --- agy ---------------------------------------------------------------

func writeAgyIndex(t *testing.T, path string) {
	t.Helper()
	db, err := sql.Open("sqlite", "file:"+path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(`CREATE TABLE conversation_summaries (
		conversation_id TEXT PRIMARY KEY,
		title TEXT NOT NULL DEFAULT '',
		preview TEXT NOT NULL DEFAULT '',
		step_count INTEGER NOT NULL DEFAULT 0,
		last_modified_time datetime NOT NULL DEFAULT '',
		workspace_uris TEXT NOT NULL DEFAULT '',
		parent_conversation_id TEXT NOT NULL DEFAULT '',
		agent_name TEXT NOT NULL DEFAULT ''
	)`); err != nil {
		t.Fatal(err)
	}
	ins := `INSERT INTO conversation_summaries (conversation_id, title, preview, step_count, last_modified_time, workspace_uris, parent_conversation_id, agent_name) VALUES (?,?,?,?,?,?,?,?)`
	// Counted: 4 steps in this folder.
	if _, err := db.Exec(ins, "c1", "Build", "Build", 4, "2026-09-01 12:00:00+00:00", `["file:///repo"]`, "", ""); err != nil {
		t.Fatal(err)
	}
	// Kept out: a subagent run, a stepless row, a folderless row.
	if _, err := db.Exec(ins, "c2", "Sub", "Sub", 3, "2026-09-01 12:00:00+00:00", `["file:///repo"]`, "c1", ""); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(ins, "c3", "", "", 0, "2026-09-01 12:00:00+00:00", `["file:///repo"]`, "", ""); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(ins, "c4", "Nowhere", "Nowhere", 2, "2026-09-01 12:00:00+00:00", ``, "", ""); err != nil {
		t.Fatal(err)
	}
}

func TestAgyMeterReadsIndex(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "conversation_summaries.db")
	writeAgyIndex(t, path)
	withAgyDB(t, path)

	w, err := AgyMeter{}.Meter(realReq())
	if err != nil {
		t.Fatal(err)
	}
	if w.Stats.Current.Messages != 4 {
		t.Fatalf("messages = %d, want the 4 steps of the one kept conversation", w.Stats.Current.Messages)
	}
	if w.Stats.Current.Sessions != 1 {
		t.Fatalf("sessions = %d, want 1", w.Stats.Current.Sessions)
	}
	if w.Stats.Turns.Assistant != 4 {
		t.Fatalf("turns = %+v, want 4 assistant", w.Stats.Turns)
	}
	sig := w.Coverage.Signals
	if sig[SigMessages] != StateReported || sig[SigTurns] != StateReported {
		t.Fatalf("signals = %v, want messages+turns reported", sig)
	}
	if sig[SigCost] != StateNotReported || sig[SigModel] != StateNotReported || sig[SigTools] != StateNotReported {
		t.Fatalf("signals = %v, want cost+model+tools not-reported", sig)
	}
}

func TestAgyMeterAbsentWithoutStore(t *testing.T) {
	withAgyDB(t, filepath.Join(t.TempDir(), "missing.db"))
	w, err := AgyMeter{}.Meter(realReq())
	if err != nil {
		t.Fatal(err)
	}
	if w.Coverage.Signals[SigMessages] != StateUnavailable {
		t.Fatalf("messages = %v, want unavailable for a missing store", w.Coverage.Signals[SigMessages])
	}
}

// --- registry ----------------------------------------------------------

func TestMetersIncludesNewCLIs(t *testing.T) {
	have := map[string]bool{}
	for _, m := range Meters() {
		have[m.CLI()] = true
	}
	for _, cli := range []string{"pi", "claude-code", "codex", "opencode", "hermes", "grok", "omp", "muse", "agy"} {
		if !have[cli] {
			t.Fatalf("Meters() is missing %s — its coverage row never renders", cli)
		}
	}
}
