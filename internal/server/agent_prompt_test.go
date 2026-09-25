package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/cfpperche/picode/internal/rpc"
	"github.com/cfpperche/picode/internal/store"
	"github.com/cfpperche/picode/internal/tmux"
)

func TestCheckPromptImages(t *testing.T) {
	ok := promptImage{MimeType: "image/png", Data: "aaa"}
	rows := []struct {
		name string
		in   []promptImage
		err  string
	}{
		{"none", nil, ""},
		{"one", []promptImage{ok}, ""},
		{"five", []promptImage{ok, ok, ok, ok, ok}, "at most 4"},
		{"bad mime", []promptImage{{MimeType: "application/pdf", Data: "x"}}, "unsupported"},
		{"empty data", []promptImage{{MimeType: "image/png"}}, "required"},
		{"too big", []promptImage{{MimeType: "image/png", Data: strings.Repeat("a", maxImageB64+1)}}, "4 MB"},
	}
	for _, r := range rows {
		err := checkPromptImages(r.in)
		if r.err == "" {
			if err != nil {
				t.Fatalf("%s: %v", r.name, err)
			}
			continue
		}
		if err == nil || !strings.Contains(err.Error(), r.err) {
			t.Fatalf("%s: %v want %s", r.name, err, r.err)
		}
	}
}

func TestAgentPromptHTTP(t *testing.T) {
	ts, _, _ := cleanupServer(t)
	proj := t.TempDir()
	if err := os.WriteFile(proj+"/a.txt", []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	wk := addWorkspaceWithAgent(t, ts, "App", proj)
	id := wk.Agents[0].ID

	body, _ := json.Marshal(map[string]any{
		"kind":    "prompt",
		"message": "see",
		"images":  []map[string]string{{"mimeType": "image/png", "data": "aaa"}},
	})
	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/api/agents/"+id+"/prompt", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	got := do(t, ts.Client(), req)
	if got.StatusCode != http.StatusConflict {
		t.Fatalf("stopped = %d want 409", got.StatusCode)
	}

	five := make([]map[string]string, 5)
	for i := range five {
		five[i] = map[string]string{"mimeType": "image/png", "data": "aaa"}
	}
	body, _ = json.Marshal(map[string]any{"message": "x", "images": five})
	req, _ = http.NewRequest(http.MethodPost, ts.URL+"/api/agents/"+id+"/prompt", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	got = do(t, ts.Client(), req)
	if got.StatusCode != http.StatusBadRequest {
		t.Fatalf("five = %d", got.StatusCode)
	}

	none := do(t, ts.Client(), mustGet(t, ts.URL+"/api/agents/ag_missing/prompt"))
	// GET not registered — method not allowed or 404. POST missing agent:
	req, _ = http.NewRequest(http.MethodPost, ts.URL+"/api/agents/ag_missing/prompt", bytes.NewReader([]byte(`{"message":"hi"}`)))
	req.Header.Set("Content-Type", "application/json")
	none = do(t, ts.Client(), req)
	if none.StatusCode != http.StatusNotFound {
		t.Fatalf("missing = %d", none.StatusCode)
	}
}

func TestAgentPromptInteractiveAndDrop(t *testing.T) {
	manager := tmux.New()
	if !manager.Available() {
		t.Skip("tmux missing")
	}
	st := testStore(t)
	deps := Deps{
		Store: st, Tmux: manager, DataDir: t.TempDir(),
		Runtime: rpc.NewRuntime("cat", st, nil), AgentCmd: "cat",
		Replies: NewTuiReplies(),
	}
	ts := httptest.NewServer(New("127.0.0.1:0", deps).Handler)
	t.Cleanup(ts.Close)
	t.Setenv("HOME", t.TempDir())
	t.Setenv("USERPROFILE", t.TempDir())
	dir := t.TempDir()
	_, agent, err := storeWorkspaceWithAgent(st, "App", dir)
	if err != nil {
		t.Fatal(err)
	}
	name := tmux.SessionName(agent.ID)
	if err := manager.NewSessionEnv(context.Background(), name, dir, nil, "/bin/sh", "-c", "sleep 300"); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = manager.KillSession(context.Background(), name) })
	if err := st.SetAgentRuntimeMode(agent.ID, store.StatusRunning, "interactive"); err != nil {
		t.Fatal(err)
	}

	code, page := postRaw(t, ts, "/api/agents/"+agent.ID+"/prompt", `{"kind":"prompt","message":"hi from attach","images":[{"mimeType":"image/png","data":"aaa"}]}`)
	if code != http.StatusConflict || page["reason"] != "images" {
		t.Fatalf("interactive+images = %d %v", code, page)
	}

	code, page = postRaw(t, ts, "/api/agents/"+agent.ID+"/prompt", `{"message":"hello tui"}`)
	if code != http.StatusOK || page["typed"] != true {
		t.Fatalf("interactive text = %d %v", code, page)
	}
	pane, err := manager.CaptureTail(context.Background(), name, 20)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(pane, "hello tui") {
		t.Fatalf("pane = %q", pane)
	}

	png := "iVBORw0KGgo="
	code, drop := postRaw(t, ts, "/api/agents/"+agent.ID+"/drop", `{"name":"shot.png","data":"`+png+`"}`)
	if code != http.StatusOK {
		t.Fatalf("drop = %d %v", code, drop)
	}
	rel, _ := drop["path"].(string)
	if rel == "" {
		t.Fatalf("drop path missing: %v", drop)
	}
	code, page = postRaw(t, ts, "/api/agents/"+agent.ID+"/prompt", `{"message":"see","paths":["`+rel+`"]}`)
	if code != http.StatusOK {
		t.Fatalf("prompt+path = %d %v", code, page)
	}

	t.Setenv("PICODE_FAKE_RPC", "1")
	st2 := testStore(t)
	ts2, _ := askServer(t, st2, os.Args[0])
	dir2 := t.TempDir()
	_, managed, err := storeWorkspaceWithAgent(st2, "App", dir2)
	if err != nil {
		t.Fatal(err)
	}
	if code, _ := postRaw(t, ts2, "/api/agents/"+managed.ID+"/managed/start", "{}"); code != http.StatusCreated && code != http.StatusOK {
		t.Fatalf("managed/start = %d", code)
	}
	body, _ := json.Marshal(map[string]any{"kind": "prompt", "message": "see", "images": []map[string]string{{"mimeType": "image/png", "data": "aaa"}}})
	req, _ := http.NewRequest(http.MethodPost, ts2.URL+"/api/agents/"+managed.ID+"/prompt", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	got := do(t, ts2.Client(), req)
	if got.StatusCode != http.StatusOK {
		t.Fatalf("managed images still 200 = %d", got.StatusCode)
	}
	code, page = postRaw(t, ts2, "/api/agents/"+managed.ID+"/prompt", `{"message":"x","paths":[".picode/drop/a.png"]}`)
	if code != http.StatusBadRequest {
		t.Fatalf("managed+paths = %d %v", code, page)
	}
}
