package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"
)

// The omp checklist mirror is a plain JS-compatible extension module POSTing
// omp's committed todo phases to the checklist routes (ADR-0055). The decision
// table runs the generated template under node against a recording daemon:
// each row is one event with one context, and the requests it produced are the
// verdict. omp's own event names and payload shapes are pinned by
// TestWrapperInstallShape; this table pins what the mirror publishes.

func TestOmpChecklistExtensionDecisionTable(t *testing.T) {
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("node not on PATH")
	}
	_, dataDir := wiringTestServer(t)
	if err := writeOmpIntercept(dataDir, hookScriptPath(dataDir)); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(ompChecklistExtensionFile(dataDir))
	if err != nil {
		t.Fatal(err)
	}
	modulePath := filepath.Join(t.TempDir(), "omp-checklist.mjs")
	if err := os.WriteFile(modulePath, raw, 0o600); err != nil {
		t.Fatal(err)
	}

	var mu sync.Mutex
	type recorded struct {
		path string
		body map[string]any
	}
	var got []recorded
	recorder := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("unparseable checklist POST: %v", err)
		}
		mu.Lock()
		got = append(got, recorded{path: r.URL.Path, body: body})
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	defer recorder.Close()

	// The harness loads the extension with a fake pi whose on() collects the
	// handlers, fires one event with the argv payload, and answers ctx with a
	// fixed mode and session manager. The serialized POST chain holds the
	// node process open until the recorder has every body, so whatever the
	// CombinedOutput returns is the whole verdict.
	harnessPath := filepath.Join(t.TempDir(), "fire-event.mjs")
	harness := `import { pathToFileURL } from "node:url";
const [modulePath, event, payload, mode, branch] = process.argv.slice(2);
const { default: load } = await import(pathToFileURL(modulePath).href);
const handlers = new Map();
load({ on(name, handler) { handlers.set(name, handler); } });
const handler = handlers.get(event);
if (!handler) throw new Error("missing handler: " + event);
const ctx = {
  mode,
  sessionManager: {
    getSessionId: () => "native-omp-1",
    getBranch: () => JSON.parse(branch || "[]"),
  },
};
await handler(payload ? JSON.parse(payload) : {}, ctx);
`
	if err := os.WriteFile(harnessPath, []byte(harness), 0o600); err != nil {
		t.Fatal(err)
	}

	phases := func(tasks ...map[string]any) map[string]any {
		return map[string]any{"toolName": "todo", "details": map[string]any{"phases": []any{map[string]any{"name": "Setup", "tasks": tasks}}}}
	}
	task := func(content, status string) map[string]any {
		return map[string]any{"content": content, "status": status}
	}
	item := func(text, status string) map[string]any {
		return map[string]any{"text": text, "status": status}
	}
	branchEntry := func(status string) string {
		entry := map[string]any{
			"type": "message",
			"message": map[string]any{
				"role":     "toolResult",
				"toolName": "todo",
				"details":  map[string]any{"phases": []any{map[string]any{"name": "Setup", "tasks": []any{task("only", status)}}}},
			},
		}
		raw, _ := json.Marshal([]any{entry})
		return string(raw)
	}

	cases := []struct {
		name    string
		event   string
		payload map[string]any
		mode    string
		branch  string
		agentID string
		termID  string
		want    []recorded
	}{
		{
			name:    "todo result publishes the mapped list",
			event:   "tool_result",
			payload: phases(task("read the code", "completed"), task("edit", "in_progress"), task("test", "pending")),
			mode:    "tui", termID: "t1",
			want: []recorded{{path: "/api/terminals/t1/checklist", body: map[string]any{
				"items": []any{item("read the code", "completed"), item("edit", "in-progress"), item("test", "pending")},
			}}},
		},
		{
			name:    "bound agent wins over the terminal",
			event:   "tool_result",
			payload: phases(task("only", "in_progress")),
			mode:    "tui", agentID: "a1", termID: "t1",
			want: []recorded{{path: "/api/agents/a1/checklist", body: map[string]any{
				"items": []any{item("only", "in-progress")},
			}}},
		},
		{
			name: "no identity publishes nothing", event: "tool_result", mode: "tui",
			payload: phases(task("only", "pending")),
		},
		{
			name:    "blocked task stays visible and names its blocker",
			event:   "tool_result",
			payload: phases(map[string]any{"content": "ship", "status": "blocked", "blocker": "ci red"}),
			mode:    "tui", termID: "t1",
			want: []recorded{{path: "/api/terminals/t1/checklist", body: map[string]any{
				"items": []any{item("ship (blocked: ci red)", "pending")},
			}}},
		},
		{
			name:    "abandoned leaves the plan",
			event:   "tool_result",
			payload: phases(task("done thing", "completed"), task("given up", "abandoned")),
			mode:    "tui", termID: "t1",
			want: []recorded{{path: "/api/terminals/t1/checklist", body: map[string]any{
				"items": []any{item("done thing", "completed")},
			}}},
		},
		{
			name:  "multi-phase plans carry the phase name",
			event: "tool_result",
			payload: map[string]any{"toolName": "todo", "details": map[string]any{"phases": []any{
				map[string]any{"name": "Setup", "tasks": []any{task("read", "completed")}},
				map[string]any{"name": "Ship", "tasks": []any{task("land", "pending")}},
			}}},
			mode: "tui", termID: "t1",
			want: []recorded{{path: "/api/terminals/t1/checklist", body: map[string]any{
				"items": []any{item("Setup — read", "completed"), item("Ship — land", "pending")},
			}}},
		},
		{name: "non-todo results are ignored", event: "tool_result", mode: "tui", termID: "t1",
			payload: map[string]any{"toolName": "bash", "details": map[string]any{"phases": []any{}}}},
		{name: "todo results without phases are ignored", event: "tool_result", mode: "tui", termID: "t1",
			payload: map[string]any{"toolName": "todo", "details": map[string]any{}}},
		{
			name: "empty plan resets the row", event: "tool_result",
			payload: phases(), mode: "tui", termID: "t1",
			want: []recorded{{path: "/api/terminals/t1/checklist", body: map[string]any{
				"reset": true,
			}}},
		},
		{
			name: "session start replays the branch snapshot", event: "session_start",
			branch: branchEntry("in_progress"), mode: "tui", termID: "t1",
			want: []recorded{{path: "/api/terminals/t1/checklist", body: map[string]any{
				"sessionId": "native-omp-1",
				"items":     []any{item("only", "in-progress")},
			}}},
		},
		{
			name: "session start with an empty branch resets", event: "session_start",
			mode: "tui", termID: "t1",
			want: []recorded{{path: "/api/terminals/t1/checklist", body: map[string]any{
				"sessionId": "native-omp-1", "reset": true,
			}}},
		},
		{
			name: "headless print runs are silent", event: "tool_result",
			payload: phases(task("only", "pending")), mode: "print", termID: "t1",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mu.Lock()
			offset := len(got)
			mu.Unlock()
			isolated := t.TempDir() // PICODE_DATA: no server.json, no token leaks in
			cmd := exec.Command("node", harnessPath, modulePath, tc.event, marshalArg(t, tc.payload), tc.mode, tc.branch)
			cmd.Env = []string{
				"PATH=/usr/bin:/bin",
				"HOME=" + isolated,
				"PICODE_DATA=" + isolated,
				"PICODE_TERM_URL=" + recorder.URL,
			}
			if tc.agentID != "" {
				cmd.Env = append(cmd.Env, "PICODE_AGENT_ID="+tc.agentID)
			}
			if tc.termID != "" {
				cmd.Env = append(cmd.Env, "PICODE_TERM_ID="+tc.termID)
			}
			if out, err := cmd.CombinedOutput(); err != nil {
				t.Fatalf("node harness: %v: %s", err, out)
			}
			mu.Lock()
			produced := append([]recorded{}, got[offset:]...)
			mu.Unlock()
			if len(produced) != len(tc.want) {
				t.Fatalf("requests = %d, want %d (%v)", len(produced), len(tc.want), produced)
			}
			for i, want := range tc.want {
				if produced[i].path != want.path {
					t.Fatalf("request %d path = %s, want %s", i, produced[i].path, want.path)
				}
				wantJSON, _ := json.Marshal(want.body)
				gotJSON, _ := json.Marshal(produced[i].body)
				if string(gotJSON) != string(wantJSON) {
					t.Fatalf("request %d body = %s, want %s", i, gotJSON, wantJSON)
				}
			}
		})
	}
}

// marshalArg encodes a harness payload, or "" when there is none.
func marshalArg(t *testing.T, payload map[string]any) string {
	t.Helper()
	if payload == nil {
		return ""
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}
