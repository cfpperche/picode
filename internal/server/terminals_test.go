package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/cfpperche/picode/internal/tmux"
)

func TestTerminalsNeedTmuxOrCRUD(t *testing.T) {
	ts, _, _ := cleanupServer(t)
	listed := do(t, ts.Client(), mustGet(t, ts.URL+"/api/terminals"))
	if listed.StatusCode != http.StatusOK {
		t.Fatalf("list = %d", listed.StatusCode)
	}
	if !tmux.New().Available() {
		t.Skip("tmux not installed — create/open gated (accepted)")
	}
	created := postJSON(t, ts, "/api/terminals", map[string]any{})
	if created.StatusCode != http.StatusCreated {
		t.Fatalf("create = %d", created.StatusCode)
	}
	var page map[string]any
	if err := json.NewDecoder(created.Body).Decode(&page); err != nil {
		t.Fatal(err)
	}
	id, _ := page["id"].(string)
	sess, _ := page["session"].(string)
	if id == "" || sess == "" {
		t.Fatalf("page=%+v", page)
	}
	t.Cleanup(func() { _ = tmux.New().KillSession(context.Background(), sess) })

	again := postJSON(t, ts, "/api/terminals/"+id+"/open", map[string]any{})
	if again.StatusCode != http.StatusOK {
		t.Fatalf("open = %d", again.StatusCode)
	}
	raw, _ := json.Marshal(map[string]string{"name": "build"})
	preq, _ := http.NewRequest(http.MethodPatch, ts.URL+"/api/terminals/"+id, bytes.NewReader(raw))
	preq.Header.Set("Content-Type", "application/json")
	renamed := do(t, ts.Client(), preq)
	if renamed.StatusCode != http.StatusOK {
		t.Fatalf("rename = %d", renamed.StatusCode)
	}
	var renamedPage map[string]any
	_ = json.NewDecoder(renamed.Body).Decode(&renamedPage)
	if renamedPage["name"] != "build" {
		t.Fatalf("name=%v", renamedPage["name"])
	}

	req, _ := http.NewRequest(http.MethodDelete, ts.URL+"/api/terminals/"+id, nil)
	killed := do(t, ts.Client(), req)
	if killed.StatusCode != http.StatusNoContent {
		t.Fatalf("delete = %d", killed.StatusCode)
	}
	listed = do(t, ts.Client(), mustGet(t, ts.URL+"/api/terminals"))
	var bag struct {
		Terminals []map[string]any `json:"terminals"`
	}
	_ = json.NewDecoder(listed.Body).Decode(&bag)
	if len(bag.Terminals) != 0 {
		t.Fatalf("after delete %+v", bag.Terminals)
	}
}

func TestTerminalsGone(t *testing.T) {
	ts, _, _ := cleanupServer(t)
	got := do(t, ts.Client(), mustGet(t, ts.URL+"/api/terminals/nope"))
	if got.StatusCode != http.StatusNotFound && got.StatusCode != http.StatusMethodNotAllowed {
		// GET by id is not a route; DELETE gone is.
	}
	req, _ := http.NewRequest(http.MethodDelete, ts.URL+"/api/terminals/nope", nil)
	del := do(t, ts.Client(), req)
	if del.StatusCode != http.StatusNotFound {
		t.Fatalf("delete missing = %d", del.StatusCode)
	}
}
