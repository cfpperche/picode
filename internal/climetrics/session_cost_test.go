package climetrics

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// An exit prices the sessions it points at (ADR-0194): the CLI's own record,
// and "not measured" — never zero — for a CLI whose sessions are not files.
func TestMeterSessionFile(t *testing.T) {
	cc, ok := MeterSessionFile("claude-code", filepath.Join("testdata", "claude-parent.jsonl"), nil)
	if !ok || cc.Tokens == 0 || cc.Turns == 0 {
		t.Fatalf("claude-code fixture = %+v, %v", cc, ok)
	}
	cx, ok := MeterSessionFile("codex", filepath.Join("testdata", "codex-rollout.jsonl"), nil)
	if !ok || cx.Tokens == 0 {
		t.Fatalf("codex fixture = %+v, %v", cx, ok)
	}
	// The models each assistant turn names (an exit's model for a terminal
	// CLI). Codex names it in turn_context, which this fixture predates; the
	// line below is the shape a 2026-09-23 rollout carries.
	if cc.Models["claude-opus-5"] != 1 || cc.Models["claude-fable-5-1"] != 1 {
		t.Fatalf("claude-code models = %v", cc.Models)
	}
	raw, err := os.ReadFile(filepath.Join("testdata", "codex-rollout.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.SplitN(string(raw), "\n", 2)
	ctx := `{"timestamp":"2026-09-06T00:32:35.000Z","type":"turn_context","payload":{"cwd":"/home/goat/picode","model":"gpt-6-astra"}}`
	withCtx := filepath.Join(t.TempDir(), "rollout.jsonl")
	if err := os.WriteFile(withCtx, []byte(lines[0]+"\n"+ctx+"\n"+lines[1]), 0o644); err != nil {
		t.Fatal(err)
	}
	if cm, ok := MeterSessionFile("codex", withCtx, nil); !ok || cm.Models["gpt-6-astra"] == 0 {
		t.Fatalf("codex models = %v, %v", cm.Models, ok)
	}
	// Without a price table nothing is estimated; Codex never writes a price.
	if cx.Estimated != 0 || cx.Cost != 0 {
		t.Fatalf("codex without prices must not invent a cost: %+v", cx)
	}
	for _, c := range []struct{ cli, path string }{
		{"grok", filepath.Join("testdata", "claude-parent.jsonl")},
		{"claude-code", filepath.Join("testdata", "missing.jsonl")},
		{"claude-code", "testdata"},
	} {
		if _, ok := MeterSessionFile(c.cli, c.path, nil); ok {
			t.Errorf("%s %s: measured, want not measured", c.cli, c.path)
		}
	}
}

// OpenCode, Hermes and Grok keep a conversation in a store or a folder, not
// one file: MeterSession prices it there, and an unknown session is "not
// measured", never zero.
func TestMeterSessionReadsStoresAndFolders(t *testing.T) {
	oc := filepath.Join(t.TempDir(), "opencode.db")
	data, _ := json.Marshal(map[string]any{
		"role": "assistant", "cost": 1.25, "modelID": "glm-5.3-flash",
		"tokens": map[string]any{"input": 900, "output": 100, "cache": map[string]any{"read": 10}},
	})
	other, _ := json.Marshal(map[string]any{"role": "assistant", "cost": 7.0, "modelID": "x"})
	newTestDB(t, oc,
		`CREATE TABLE message (id TEXT PRIMARY KEY, session_id TEXT, time_created INT, data TEXT)`,
		`INSERT INTO message VALUES ('m1','s1',1,'`+string(data)+`')`,
		`INSERT INTO message VALUES ('m2','s2',1,'`+string(other)+`')`,
	)
	c, ok := MeterSession("opencode", oc, "s1", nil)
	if !ok || !approx(c.Cost, 1.25) || c.Tokens != 1010 || c.Turns != 1 || c.Models["glm-5.3-flash"] != 1 {
		t.Fatalf("opencode s1 = %+v, %v; want only its own row", c, ok)
	}

	hm := filepath.Join(t.TempDir(), "state.db")
	newTestDB(t, hm,
		`CREATE TABLE sessions (id TEXT PRIMARY KEY, cwd TEXT, model TEXT, title TEXT, cost_status TEXT,
			actual_cost_usd REAL, estimated_cost_usd REAL, input_tokens INT, output_tokens INT,
			cache_read_tokens INT, cache_write_tokens INT, reasoning_tokens INT)`,
		`CREATE TABLE messages (id INTEGER PRIMARY KEY, session_id TEXT, role TEXT, timestamp REAL)`,
		`INSERT INTO sessions VALUES ('h1','/repo','hermes-1','T','',0,2.5,1000,200,0,0,0)`,
		`INSERT INTO sessions VALUES ('h2','/repo','no-such-model','T','',0,0,1000,200,0,0,0)`,
		`INSERT INTO messages VALUES (1,'h1','user',1)`,
		`INSERT INTO messages VALUES (2,'h1','assistant',2)`,
		`INSERT INTO messages VALUES (3,'h2','assistant',2)`,
	)
	c, ok = MeterSession("hermes", hm, "h1", nil)
	if !ok || !approx(c.Cost, 2.5) || c.Tokens != 1200 || c.Turns != 1 || c.Models["hermes-1"] != 1 {
		t.Fatalf("hermes h1 = %+v, %v", c, ok)
	}
	// Tokens with no cost and no price: counted as unpriced, not as free.
	if c, ok = MeterSession("hermes", hm, "h2", nil); !ok || c.Cost != 0 || c.Unpriced != 1 {
		t.Fatalf("hermes h2 = %+v, %v; want one unpriced turn", c, ok)
	}

	home := withGrokRoot(t)
	folder := filepath.Join(home, "sessions", "%2Frepo")
	grokWrite(t, folder, "prompt_history.jsonl", `{"timestamp":"`+grokStamp(0)+`","session_id":"g1","prompt":"p"}`+"\n")
	sess := filepath.Join(folder, "g1")
	grokWrite(t, sess, "summary.json", grokSummaryJSON)
	grokWrite(t, sess, "usage.json", grokUsage(grokStamp(10)))
	// The listing names the session folder, or the prompt history plus the id.
	for _, p := range []struct{ path, id string }{{sess, "g1"}, {filepath.Join(folder, "prompt_history.jsonl"), "g1"}} {
		c, ok := MeterSession("grok", p.path, p.id, nil)
		if !ok || c.Cost <= 0 || c.Tokens == 0 {
			t.Fatalf("grok %s = %+v, %v", p.path, c, ok)
		}
	}

	for _, n := range []struct{ cli, path, id string }{
		{"opencode", oc, "missing"},
		{"hermes", hm, "missing"},
		{"hermes", filepath.Join(t.TempDir(), "absent.db"), "h1"},
		{"grok", filepath.Join(folder, "prompt_history.jsonl"), "missing"},
		{"opencode", oc, ""},
	} {
		if c, ok := MeterSession(n.cli, n.path, n.id, nil); ok {
			t.Errorf("%s %s/%s = %+v: measured, want not measured", n.cli, n.path, n.id, c)
		}
	}
}
