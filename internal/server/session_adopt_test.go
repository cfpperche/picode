package server

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/cfpperche/picode/internal/session"
)

// writeSessionFixture mints one pi JSONL in the given cwd's session dir.
func writeSessionFixture(t *testing.T, cwd string) string {
	t.Helper()
	dir := session.Dir(cwd)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(dir, "s.jsonl")
	body := `{"type":"session","version":3,"id":"abc","timestamp":"2026-08-24T01:00:00.000Z","cwd":"` + cwd + `"}
{"type":"session_info","name":"Refactor auth"}
{"type":"model_change","provider":"xai","modelId":"grok-4.6"}
{"type":"thinking_level_change","thinkingLevel":"high"}
{"type":"message","message":{"role":"user","content":[{"type":"text","text":"hello"}]}}
`
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestListPiSessions(t *testing.T) {
	ts, _, home := cleanupServer(t)
	proj := filepath.Join(home, "p")
	if err := os.MkdirAll(proj, 0o755); err != nil {
		t.Fatal(err)
	}
	writeSessionFixture(t, proj)
	got := do(t, ts.Client(), mustGet(t, ts.URL+"/api/clis/pi/sessions"))
	if got.StatusCode != http.StatusOK {
		t.Fatalf("list = %d", got.StatusCode)
	}
	var bag struct {
		Sessions []session.Summary `json:"sessions"`
	}
	if err := json.NewDecoder(got.Body).Decode(&bag); err != nil {
		t.Fatal(err)
	}
	if len(bag.Sessions) != 1 || bag.Sessions[0].Cwd != proj {
		t.Fatalf("%+v", bag.Sessions)
	}
}

// ADR-0126: session adoption is gone — the route is no longer registered,
// so nothing answers it with a created agent (an unknown path falls through
// to the UI fallback, which is 503 in a bare test server).
func TestAdoptRouteIsGone(t *testing.T) {
	ts, _, _ := cleanupServer(t)
	for _, cli := range []string{"pi", "claude-code", "opencode"} {
		res := postJSON(t, ts, "/api/clis/"+cli+"/sessions/adopt", map[string]string{"path": "x"})
		if res.StatusCode == http.StatusCreated || res.StatusCode == http.StatusOK {
			t.Fatalf("adopt on %s = %d, route should be gone", cli, res.StatusCode)
		}
	}
}
