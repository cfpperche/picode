package server

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/cfpperche/picode/internal/session"
	"github.com/cfpperche/picode/internal/store"
)

func writeAdoptSession(t *testing.T, cwd string) string {
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

func TestAdoptPiSessionDecisionTable(t *testing.T) {
	ts, _, home := cleanupServer(t)
	known := filepath.Join(home, "known")
	if err := os.MkdirAll(known, 0o755); err != nil {
		t.Fatal(err)
	}
	ws := postJSON(t, ts, "/api/workspaces", map[string]string{"name": "App", "path": known})
	if ws.StatusCode != http.StatusCreated {
		t.Fatalf("workspace = %d", ws.StatusCode)
	}

	t.Run("known cwd becomes workspace agent", func(t *testing.T) {
		src := writeAdoptSession(t, known)
		res := postJSON(t, ts, "/api/pi-sessions/adopt", map[string]string{"path": src})
		if res.StatusCode != http.StatusCreated {
			t.Fatalf("adopt = %d", res.StatusCode)
		}
		var ag agentView
		if err := json.NewDecoder(res.Body).Decode(&ag); err != nil {
			t.Fatal(err)
		}
		if ag.WorkspaceID == store.FreeWorkspaceID {
			t.Fatalf("want workspace agent, got free %+v", ag)
		}
		if ag.SessionPath == nil || *ag.SessionPath == src {
			t.Fatalf("must copy, still %v", ag.SessionPath)
		}
		if _, err := os.Stat(src); err != nil {
			t.Fatal("source must stay")
		}
		if ag.Provider == nil || *ag.Provider != "xai" || ag.Model == nil || *ag.Model != "grok-4.6" || ag.Thinking == nil || *ag.Thinking != "high" {
			t.Fatalf("cfg provider=%v model=%v thinking=%v", ag.Provider, ag.Model, ag.Thinking)
		}
	})

	t.Run("unknown cwd becomes free agent", func(t *testing.T) {
		other := filepath.Join(home, "other")
		if err := os.MkdirAll(other, 0o755); err != nil {
			t.Fatal(err)
		}
		src := writeAdoptSession(t, other)
		res := postJSON(t, ts, "/api/pi-sessions/adopt", map[string]string{"path": src})
		if res.StatusCode != http.StatusCreated {
			t.Fatalf("adopt = %d", res.StatusCode)
		}
		var ag agentView
		if err := json.NewDecoder(res.Body).Decode(&ag); err != nil {
			t.Fatal(err)
		}
		if ag.WorkspaceID != store.FreeWorkspaceID {
			t.Fatalf("want free, got %+v", ag)
		}
		if ag.WorkPath == nil || filepath.Clean(*ag.WorkPath) != filepath.Clean(other) {
			t.Fatalf("workPath=%v", ag.WorkPath)
		}
	})

	t.Run("path outside sessions root is rejected", func(t *testing.T) {
		res := postJSON(t, ts, "/api/pi-sessions/adopt", map[string]string{"path": "/etc/passwd"})
		if res.StatusCode != http.StatusBadRequest {
			t.Fatalf("outside = %d", res.StatusCode)
		}
	})
}

func TestListPiSessions(t *testing.T) {
	ts, _, home := cleanupServer(t)
	proj := filepath.Join(home, "p")
	if err := os.MkdirAll(proj, 0o755); err != nil {
		t.Fatal(err)
	}
	writeAdoptSession(t, proj)
	got := do(t, ts.Client(), mustGet(t, ts.URL+"/api/pi-sessions"))
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
