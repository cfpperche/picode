package server

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/cfpperche/picode/internal/rpc"
	"github.com/cfpperche/picode/internal/store"
	"github.com/cfpperche/picode/internal/tmux"
)

// The fake re-execs this test binary as a minimal pi-rpc double that
// answers get_state, bash, and abort_bash (roadmap A3). Same trick as
// internal/rpc/client_test.go, but local: rpc's _test.go fakes are not
// linked into this package's test binary.
func init() {
	if os.Getenv("PICODE_FAKE_RPC") == "1" {
		fakeBashRPCMain()
		os.Exit(0)
	}
}

func fakeBashRPCMain() {
	dec := json.NewDecoder(os.Stdin)
	enc := json.NewEncoder(os.Stdout)
	for {
		var req map[string]any
		if err := dec.Decode(&req); err != nil {
			return
		}
		id, _ := req["id"].(string)
		typ, _ := req["type"].(string)
		switch typ {
		case "get_state":
			_ = enc.Encode(map[string]any{
				"id": id, "type": "response", "command": "get_state", "success": true,
				"data": map[string]any{"model": map[string]any{"id": "fake"}, "isStreaming": false},
			})
		case "prompt":
			msg, _ := req["message"].(string)
			if len(msg) >= 10 && msg[:10] == "/mcp-auth " {
				_ = enc.Encode(map[string]any{
					"type": "extension_ui_request", "id": "ui-auth",
					"method": "input", "title": "Complete OAuth\nhttps://example.test/oauth\nPaste",
				})
				var reply map[string]any
				if err := dec.Decode(&reply); err != nil {
					return
				}
				_ = enc.Encode(map[string]any{
					"id": id, "type": "response", "command": typ, "success": true,
				})
				break
			}
			_ = enc.Encode(map[string]any{
				"id": id, "type": "response", "command": typ, "success": true,
			})
		case "bash":
			cmd, _ := req["command"].(string)
			_ = enc.Encode(map[string]any{"type": "bash_execution_update", "id": id, "delta": "fake: " + cmd + "\n"})
			_ = enc.Encode(map[string]any{
				"id": id, "type": "response", "command": "bash", "success": true,
				"data": map[string]any{"output": "fake: " + cmd + "\n", "exitCode": 0, "cancelled": false, "truncated": false},
			})
		case "abort_bash":
			_ = enc.Encode(map[string]any{"id": id, "type": "response", "command": "abort_bash", "success": true})
		default:
			_ = enc.Encode(map[string]any{"id": id, "type": "response", "command": typ, "success": true})
		}
	}
}

// bashTestServer builds a server whose agent command is this test binary
// re-exec'd as the fake pi rpc double (see internal/rpc/client_test.go).
func bashTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	t.Setenv("PICODE_FAKE_RPC", "1")
	root := t.TempDir()
	home := filepath.Join(root, "home")
	dataDir := filepath.Join(root, "data")
	for _, d := range []string{home, dataDir} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	st, err := store.Open(filepath.Join(dataDir, "picode.db"))
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	ts := httptest.NewServer(New("127.0.0.1:0", Deps{
		Store:    st,
		Tmux:     tmux.New(),
		Runtime:  rpc.NewRuntime(os.Args[0], st, nil),
		AgentCmd: os.Args[0],
		DataDir:  dataDir,
	}).Handler)
	t.Cleanup(ts.Close)
	return ts
}

func TestAgentBashMatrix(t *testing.T) {
	ts := bashTestServer(t)
	proj := t.TempDir()
	res := postJSON(t, ts, "/api/workspaces", map[string]string{"name": "App", "path": proj})
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("add = %d", res.StatusCode)
	}
	var wk workspaceView
	_ = json.NewDecoder(res.Body).Decode(&wk)
	id := wk.Agent.ID

	rows := []struct {
		name string
		cmd  string
		code int
	}{
		{"empty", "", http.StatusBadRequest},
		{"double bang", "!!ls", http.StatusBadRequest},
		{"not running", "ls", http.StatusConflict},
	}
	for _, r := range rows {
		got := postJSON(t, ts, "/api/agents/"+id+"/bash", map[string]string{"command": r.cmd})
		if got.StatusCode != r.code {
			t.Fatalf("%s = %d want %d", r.name, got.StatusCode, r.code)
		}
	}

	missing := postJSON(t, ts, "/api/agents/ag_nope/bash", map[string]string{"command": "ls"})
	if missing.StatusCode != http.StatusNotFound {
		t.Fatalf("missing agent = %d", missing.StatusCode)
	}

	// Abort while nothing runs: agent not started yet.
	ab := postJSON(t, ts, "/api/agents/"+id+"/bash/abort", map[string]string{})
	if ab.StatusCode != http.StatusConflict {
		t.Fatalf("abort stopped = %d", ab.StatusCode)
	}
}

func TestAgentBashRunsWithFakeRPC(t *testing.T) {
	ts := bashTestServer(t)
	proj := t.TempDir()
	if err := os.WriteFile(filepath.Join(proj, "x.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	res := postJSON(t, ts, "/api/workspaces", map[string]string{"name": "App", "path": proj})
	var wk workspaceView
	_ = json.NewDecoder(res.Body).Decode(&wk)
	id := wk.Agent.ID

	start := postJSON(t, ts, "/api/agents/"+id+"/managed/start", map[string]string{})
	if start.StatusCode != http.StatusCreated {
		t.Fatalf("start = %d", start.StatusCode)
	}

	got := postJSON(t, ts, "/api/agents/"+id+"/bash", map[string]string{"command": "echo hi"})
	if got.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(got.Body)
		t.Fatalf("bash = %d body=%s", got.StatusCode, body)
	}
	var out struct {
		Output   string `json:"output"`
		ExitCode int    `json:"exitCode"`
	}
	if err := json.NewDecoder(got.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	if out.ExitCode != 0 || out.Output == "" {
		t.Fatalf("out = %+v", out)
	}

	ab := postJSON(t, ts, "/api/agents/"+id+"/bash/abort", map[string]string{})
	if ab.StatusCode != http.StatusOK {
		t.Fatalf("abort = %d", ab.StatusCode)
	}
}
