package server

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/cfpperche/picode/internal/clilaunch"
	"github.com/cfpperche/picode/internal/clisession"
	"github.com/cfpperche/picode/internal/rpc"
	"github.com/cfpperche/picode/internal/session"
	"github.com/cfpperche/picode/internal/store"
	"github.com/cfpperche/picode/internal/tmux"
	"github.com/cfpperche/picode/internal/transcript"
)

// handoffServer is cleanupServer with the Deps kept, so a test can register
// a CLI runtime (the live-source branch) without the wrapper protocol.
func handoffServer(t *testing.T) (ts *httptest.Server, deps Deps, dataDir, home string) {
	t.Helper()
	root := t.TempDir()
	home = filepath.Join(root, "home")
	dataDir = filepath.Join(root, "data")
	for _, d := range []string{home, dataDir} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	st, err := store.Open(filepath.Join(dataDir, "picode.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	deps = Deps{Store: st, Tmux: tmux.New(), Runtime: rpc.NewRuntime("cat", st, nil), AgentCmd: "cat", DataDir: dataDir, TermRuntimes: NewTermRuntimes()}
	ts = httptest.NewServer(New("127.0.0.1:0", deps).Handler)
	t.Cleanup(ts.Close)
	return ts, deps, dataDir, home
}

// cleanupTerm kills the tmux session of a terminal a handoff opened. It is
// registered before anything is asserted about the response: a failed
// assertion must not leak a live pane, which is how 39 of them piled up
// during this feature's own test runs.
func cleanupTerm(t *testing.T, res map[string]any) map[string]any {
	t.Helper()
	term, _ := res["terminal"].(map[string]any)
	if term == nil {
		return nil
	}
	id, _ := term["id"].(string)
	if id != "" {
		t.Cleanup(func() { killTermPane(tmux.ShellSessionName(id)) })
	}
	return term
}

// killTermPane kills a pane and confirms it is gone. A single kill loses to
// tmux under a loaded full-matrix run: panes named after this file's
// fixtures kept surviving a suite that reported clean, one per run or so.
func killTermPane(name string) {
	m := tmux.New()
	for i := 0; i < 5; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		_ = m.KillSession(ctx, name)
		alive, err := m.HasSession(ctx, name)
		cancel()
		if err != nil || !alive {
			return
		}
		time.Sleep(200 * time.Millisecond)
	}
}

// fakeCLI installs a CLI executable that records its argv to $QA_OUTPUT.args
// and then stays attached, and registers it for the catalog id.
func fakeCLI(t *testing.T, ts *httptest.Server, home, cli string) (bin, out string) {
	t.Helper()
	dir := filepath.Join(home, "bin-"+cli)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	bin = filepath.Join(dir, cli)
	out = filepath.Join(home, cli+"-run")
	script := "#!/bin/sh\nif [ \"$1\" = --version ]; then printf '" + cli + " 9.9.9\\n'; exit 0; fi\nprintf '%s\\000' \"$@\" > \"$QA_OUTPUT.args\"\nexec cat\n"
	if err := os.WriteFile(bin, []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	cliRequest(t, ts, "PUT", "/api/clis/"+cli, clilaunch.Config{Executable: bin, Env: map[string]string{"QA_OUTPUT": out}}, 200)
	return bin, out
}

func seedHandoffClaude(t *testing.T, home, proj string) string {
	t.Helper()
	dir := filepath.Join(home, ".claude", "projects", strings.NewReplacer("/", "-", ".", "-").Replace(proj))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "cc-1.jsonl")
	body := strings.Join([]string{
		`{"type":"summary","summary":"Race fix"}`,
		`{"type":"user","message":{"role":"user","content":[{"type":"text","text":"fix the race"}]},"timestamp":"2026-09-01T10:00:02.000Z","cwd":"` + proj + `","sessionId":"cc-1","version":"2.1.263","uuid":"u1"}`,
		`{"type":"assistant","message":{"id":"m1","role":"assistant","model":"claude-sonnet-5","content":[{"type":"text","text":"On it."},{"type":"tool_use","id":"toolu_1","name":"Bash","input":{"command":"go test ./..."}}]},"timestamp":"2026-09-01T10:00:03.000Z","sessionId":"cc-1","cwd":"` + proj + `","uuid":"a1"}`,
		`{"type":"user","message":{"role":"user","content":[{"tool_use_id":"toolu_1","type":"tool_result","content":"ok"}]},"timestamp":"2026-09-01T10:00:04.000Z","sessionId":"cc-1","cwd":"` + proj + `","uuid":"u2"}`,
		`{"type":"assistant","message":{"id":"m2","role":"assistant","model":"claude-sonnet-5","content":[{"type":"text","text":"Done."}]},"timestamp":"2026-09-01T10:00:05.000Z","sessionId":"cc-1","cwd":"` + proj + `","uuid":"a2"}`,
	}, "\n")
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func snapshotTree(t *testing.T, root string) map[string]string {
	t.Helper()
	out := map[string]string{}
	_ = filepath.Walk(root, func(p string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() {
			out[p] = info.ModTime().String() + ":" + strconv.FormatInt(info.Size(), 10)
		}
		return nil
	})
	return out
}

func TestCLIsAdvertiseSessionCapabilities(t *testing.T) {
	ts, _, _ := cleanupServer(t)
	res := cliRequest(t, ts, "GET", "/api/clis", nil, 200)
	want := map[string]map[string]any{
		"pi":          {"list": true, "read": true, "write": true, "prompt": true, "fork": false, "agent": true},
		"claude-code": {"list": true, "read": true, "write": true, "prompt": true, "fork": true, "agent": false},
		"codex":       {"list": true, "read": true, "write": true, "prompt": true, "fork": true, "agent": false},
		"grok":        {"list": true, "read": true, "write": true, "prompt": true, "fork": true, "agent": false},
		"hermes":      {"list": true, "read": true, "write": true, "prompt": false, "fork": false, "agent": false},
		"opencode":    {"list": true, "read": true, "write": true, "prompt": true, "fork": true, "agent": false},
	}
	seen := 0
	for _, row := range res["clis"].([]any) {
		c := row.(map[string]any)
		exp, ok := want[c["id"].(string)]
		if !ok {
			continue
		}
		seen++
		if !reflect.DeepEqual(c["sessions"], exp) {
			t.Errorf("%s sessions = %v, want %v", c["id"], c["sessions"], exp)
		}
	}
	if seen != len(want) {
		t.Fatalf("saw %d catalog rows with capabilities, want %d", seen, len(want))
	}
}

func TestHandoffDecisionTable(t *testing.T) {
	ts, _, dataDir, home := handoffServer(t)
	proj := filepath.Join(t.TempDir(), "proj")
	if err := os.MkdirAll(proj, 0o755); err != nil {
		t.Fatal(err)
	}
	path := seedHandoffClaude(t, home, proj)
	base := func() map[string]any { return map[string]any{"id": "cc-1", "path": path, "cwd": proj} }
	with := func(kv ...any) map[string]any {
		m := base()
		for i := 0; i+1 < len(kv); i += 2 {
			m[kv[i].(string)] = kv[i+1]
		}
		return m
	}
	preview := "/api/clis/claude-code/sessions/handoff/preview"

	cliRequest(t, ts, "POST", "/api/clis/nope/sessions/handoff/preview", with("to", "codex"), 404)
	cliRequest(t, ts, "POST", preview, with("to", "nope"), 404)
	cliRequest(t, ts, "POST", preview, with("to", "claude-code"), 400)
	cliRequest(t, ts, "POST", preview, map[string]any{"to": "codex", "cwd": proj}, 400) // no id/path
	cliRequest(t, ts, "POST", preview, with("to", "codex", "window", "yesterday"), 400)
	cliRequest(t, ts, "POST", preview, with("to", "codex", "mode", "magic"), 400)
	cliRequest(t, ts, "POST", preview, with("to", "codex", "extra", 1), 400) // DisallowUnknownFields
	cliRequest(t, ts, "POST", preview, with("to", "codex", "path", filepath.Join(t.TempDir(), "x.jsonl")), 400)

	// Hermes imports but cannot be started with a prompt, so it offers the
	// native mode only and refuses a brief.
	fakeCLI(t, ts, home, "hermes")
	r := cliRequestFull(t, ts, "POST", preview, with("to", "hermes", "mode", "brief"))
	if r["status"] != "409" || !strings.Contains(r["body"].(map[string]any)["error"].(string), "cannot start from a brief") {
		t.Fatalf("hermes brief: %v", r)
	}
	if modes := r["body"].(map[string]any)["modes"]; !reflect.DeepEqual(modes, []any{"native"}) {
		t.Fatalf("hermes modes = %v", modes)
	}
	// Grok takes either, and native is the default when both are offered.
	fakeCLI(t, ts, home, "grok")
	p := cliRequest(t, ts, "POST", preview, with("to", "grok"), 200)
	if p["mode"] != "native" || !reflect.DeepEqual(p["modes"], []any{"native", "brief"}) {
		t.Fatalf("grok default mode: %v", p)
	}
	p = cliRequest(t, ts, "POST", preview, with("to", "grok", "mode", "brief"), 200)
	if p["mode"] != "brief" || p["briefPreview"] == nil || !strings.Contains(p["briefPreview"].(string), "## Last request\nfix the race") {
		t.Fatalf("grok brief: %v", p)
	}
	// Codex is not installed yet: 400 before anything is created.
	cliRequest(t, ts, "PUT", "/api/clis/codex", clilaunch.Config{Executable: "/definitely/missing"}, 200)
	cliRequest(t, ts, "POST", preview, with("to", "codex"), 400)

	// Preview writes nothing, anywhere.
	fakeCLI(t, ts, home, "codex")
	before := snapshotTree(t, home)
	beforeData := snapshotTree(t, dataDir)
	p = cliRequest(t, ts, "POST", preview, with("to", "codex"), 200)
	if !reflect.DeepEqual(snapshotTree(t, home), before) {
		t.Fatal("preview touched the home tree")
	}
	delete(beforeData, filepath.Join(dataDir, "picode.db"))
	after := snapshotTree(t, dataDir)
	delete(after, filepath.Join(dataDir, "picode.db"))
	for k := range after {
		if strings.Contains(k, "handoffs") {
			t.Fatalf("preview wrote %s", k)
		}
	}
	if p["mode"] != "native" || !reflect.DeepEqual(p["modes"], []any{"native", "brief"}) || p["window"] != "recent" || p["tools"] != "native" || p["hasCompaction"] != false {
		t.Fatalf("codex preview: %v", p)
	}
	counts := p["counts"].(map[string]any)
	// note + user + assistant + assistant = 4 messages, one tool pair
	if counts["messages"] != 4.0 || counts["toolCalls"] != 1.0 || counts["toolResults"] != 1.0 || counts["thinking"] != 0.0 {
		t.Fatalf("counts = %v", counts)
	}
	if p["formatVersion"] != "9.9.9" {
		t.Fatalf("formatVersion must come from the setup check: %v", p["formatVersion"])
	}
}

func TestHandoffOversizeFallsBackToBrief(t *testing.T) {
	ts, _, dataDir, home := handoffServer(t)
	proj := filepath.Join(t.TempDir(), "proj")
	if err := os.MkdirAll(proj, 0o755); err != nil {
		t.Fatal(err)
	}
	path := seedHandoffClaude(t, home, proj)
	f, err := os.OpenFile(path, os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.Seek(clisession.MaxReadBytes, 0); err != nil {
		t.Fatal(err)
	}
	late := `{"type":"user","message":{"role":"user","content":[{"type":"text","text":"the last request"}]},"timestamp":"2026-09-01T12:00:00.000Z","cwd":"` + proj + `","sessionId":"cc-1","uuid":"u-late"}` + "\n"
	if _, err := f.WriteString(late); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	fakeCLI(t, ts, home, "grok")
	body := map[string]any{"id": "cc-1", "path": path, "cwd": proj, "to": "grok"}
	p := cliRequest(t, ts, "POST", "/api/clis/claude-code/sessions/handoff/preview", body, 200)
	if p["mode"] != "brief" || p["tooLarge"] != true || !reflect.DeepEqual(p["modes"], []any{"brief"}) {
		t.Fatalf("oversize preview: %v", p)
	}
	if p["briefPreview"] == nil || !strings.Contains(p["briefPreview"].(string), "the last request") {
		t.Fatalf("brief missed the tail: %v", p)
	}
	warnings, _ := p["manifest"].(map[string]any)["warnings"].([]any)
	if len(warnings) == 0 {
		t.Fatalf("expected a too-large warning: %v", p["manifest"])
	}
	body["mode"] = "native"
	r := cliRequestFull(t, ts, "POST", "/api/clis/claude-code/sessions/handoff", body)
	if r["status"] != "413" {
		t.Fatalf("native execute of oversize: %v", r)
	}
	body["mode"] = "brief"
	res := cliRequest(t, ts, "POST", "/api/clis/claude-code/sessions/handoff", body, 201)
	cleanupTerm(t, res)
	brief, _ := res["brief"].(map[string]any)
	if brief == nil || brief["path"] == nil {
		t.Fatalf("brief execute: %v", res)
	}
	if _, err := os.Stat(brief["path"].(string)); err != nil {
		t.Fatal(err)
	}
	_ = dataDir
}

func TestHandoffNativeClaudeToCodexCreatesRolloutAndTerminal(t *testing.T) {
	ts, _, _, home := handoffServer(t)
	proj := filepath.Join(t.TempDir(), "proj")
	if err := os.MkdirAll(proj, 0o755); err != nil {
		t.Fatal(err)
	}
	cliRequest(t, ts, "POST", "/api/workspaces", map[string]any{"name": "proj", "path": proj}, 201)
	path := seedHandoffClaude(t, home, proj)
	_, out := fakeCLI(t, ts, home, "codex")

	res := cliRequest(t, ts, "POST", "/api/clis/claude-code/sessions/handoff", map[string]any{"id": "cc-1", "path": path, "cwd": proj, "to": "codex"}, 201)
	term := cleanupTerm(t, res)
	target := res["target"].(map[string]any)
	if term == nil || term["launchError"] != nil {
		t.Fatalf("launch error: %v", term["launchError"])
	}
	rollout := target["path"].(string)
	if !strings.HasPrefix(rollout, filepath.Join(home, ".codex", "sessions")) || !strings.HasSuffix(rollout, "-"+target["id"].(string)+".jsonl") {
		t.Fatalf("rollout path = %s", rollout)
	}
	raw, err := os.ReadFile(rollout)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), `"cli_version":"9.9.9"`) || !strings.Contains(string(raw), "Handoff from Claude Code running claude-sonnet-5, session cc-1") || !strings.Contains(string(raw), `"type":"function_call"`) {
		t.Fatalf("rollout body:\n%s", raw)
	}
	argv := strings.Split(strings.TrimSuffix(string(waitCLIFile(t, out+".args")), "\x00"), "\x00")
	// The handoff lands as an agent (ADR-0184), so PiCode's tool families
	// follow the resume arguments like on any agent's launch.
	if len(argv) < 2 || !reflect.DeepEqual(argv[:2], []string{"resume", target["id"].(string)}) {
		t.Fatalf("codex launched with %q", argv)
	}
	if a, _ := res["agent"].(map[string]any); a == nil || a["cli"] != "codex" || a["terminalId"] != term["id"] {
		t.Fatalf("handoff agent = %v", res["agent"])
	}
	h := res["handoff"].(map[string]any)
	if h["mode"] != "native" || h["sourceId"] != "cc-1" || h["targetCli"] != "codex" || h["targetId"] != target["id"] || h["terminalId"] != term["id"] {
		t.Fatalf("handoff row = %v", h)
	}
	if res["manifest"].(map[string]any)["dropped"] == nil {
		t.Fatalf("manifest missing: %v", res["manifest"])
	}
	// Lineage on both listings.
	src := cliRequest(t, ts, "GET", "/api/clis/claude-code/sessions?cwd="+proj, nil, 200)
	row := src["sessions"].([]any)[0].(map[string]any)
	to := row["handoff"].(map[string]any)["to"].([]any)[0].(map[string]any)
	if to["cli"] != "codex" || to["id"] != target["id"] || to["terminalId"] != term["id"] {
		t.Fatalf("source lineage = %v", row["handoff"])
	}
	dst := cliRequest(t, ts, "GET", "/api/clis/codex/sessions?cwd="+proj, nil, 200)
	drow := dst["sessions"].([]any)[0].(map[string]any)
	from := drow["handoff"].(map[string]any)["from"].(map[string]any)
	if from["cli"] != "claude-code" || from["id"] != "cc-1" || drow["workspace"] != "proj" {
		t.Fatalf("target lineage = %v", drow)
	}

}

func TestHandoffBriefWritesFileAndPromptArgs(t *testing.T) {
	ts, _, dataDir, home := handoffServer(t)
	proj := filepath.Join(t.TempDir(), "proj")
	if err := os.MkdirAll(proj, 0o755); err != nil {
		t.Fatal(err)
	}
	path := seedHandoffClaude(t, home, proj)
	_, grokOut := fakeCLI(t, ts, home, "grok")
	_, codexOut := fakeCLI(t, ts, home, "codex")

	res := cliRequest(t, ts, "POST", "/api/clis/claude-code/sessions/handoff", map[string]any{"id": "cc-1", "path": path, "cwd": proj, "to": "grok", "mode": "brief"}, 201)
	cleanupTerm(t, res)
	brief := res["brief"].(map[string]any)["path"].(string)
	if !strings.HasPrefix(brief, filepath.Join(dataDir, "handoffs")) || !strings.HasSuffix(brief, "brief.md") {
		t.Fatalf("brief path = %s", brief)
	}
	body, err := os.ReadFile(brief)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(body), "# Handoff from Claude Code") || !strings.Contains(string(body), "## Commands run\n- go test ./...") {
		t.Fatalf("brief:\n%s", body)
	}
	argv := strings.Split(strings.TrimSuffix(string(waitCLIFile(t, grokOut+".args")), "\x00"), "\x00")
	h := res["handoff"].(map[string]any)
	if len(argv) != 3 || argv[0] != "--session-id" || argv[1] != h["targetId"] || !strings.HasPrefix(argv[2], "Continue a session handed off from Claude Code: read "+brief) {
		t.Fatalf("grok launched with %q (handoff %v)", argv, h)
	}
	if h["mode"] != "brief" || h["targetId"] == "" {
		t.Fatalf("brief handoff row = %v", h)
	}
	if entries, _ := os.ReadDir(filepath.Join(home, ".grok")); len(entries) != 0 {
		t.Fatal("a brief handoff must not write the target's home")
	}

	// Codex cannot pre-assign an id: no targetId, positional prompt only.
	res = cliRequest(t, ts, "POST", "/api/clis/claude-code/sessions/handoff", map[string]any{"id": "cc-1", "path": path, "cwd": proj, "to": "codex", "mode": "brief"}, 201)
	cleanupTerm(t, res)
	argv = strings.Split(strings.TrimSuffix(string(waitCLIFile(t, codexOut+".args")), "\x00"), "\x00")
	if len(argv) < 1 || !strings.HasPrefix(argv[0], "Continue a session handed off") || (len(argv) > 1 && argv[1] != "-c") {
		t.Fatalf("codex launched with %q", argv)
	}
	if res["handoff"].(map[string]any)["targetId"] != nil {
		t.Fatalf("codex brief must leave targetId empty: %v", res["handoff"])
	}
}

// The landing choice (agent vs terminal) is a user decision for pi, the
// only CLI that is also a managed agent. These tests cover the decision
// table's landing rows; the pi-native default (agent) and pi-brief
// default (terminal) stay covered by TestHandoffToPiAdoptsAgent and
// TestHandoffBriefWritesFileAndPromptArgs's shape.
func TestHandoffPiTerminalLandingNative(t *testing.T) {
	ts, _, _, home := handoffServer(t)
	old := session.TestRoot
	session.TestRoot = filepath.Join(home, ".pi", "agent", "sessions")
	defer func() { session.TestRoot = old }()
	proj := filepath.Join(t.TempDir(), "proj")
	if err := os.MkdirAll(proj, 0o755); err != nil {
		t.Fatal(err)
	}
	path := seedHandoffClaude(t, home, proj)
	fakeCLI(t, ts, home, "pi")

	res := cliRequest(t, ts, "POST", "/api/clis/claude-code/sessions/handoff", map[string]any{"id": "cc-1", "path": path, "cwd": proj, "to": "pi", "landing": "terminal"}, 201)
	// A terminal landing is still an agent (ADR-0184): a Pi agent whose
	// bound terminal runs `pi --session`, not a managed one.
	if a, _ := res["agent"].(map[string]any); a == nil || a["terminalId"] == nil {
		t.Fatalf("a terminal landing is an agent bound to its terminal: %v", res["agent"])
	}
	if res["landing"] != "terminal" {
		t.Fatalf("landing = %v", res["landing"])
	}
	sum := res["target"].(map[string]any)
	if !strings.HasPrefix(sum["path"].(string), session.Dir(proj)) {
		t.Fatalf("session path %v is not in pi's cwd bucket", sum["path"])
	}
	argv := strings.Split(strings.TrimSuffix(string(waitCLIFile(t, home+"/pi-run.args")), "\x00"), "\x00")
	// The agent's own spawn flags (session dir, extensions) follow.
	if len(argv) < 2 || argv[0] != "--session" || argv[1] != sum["path"] {
		t.Fatalf("pi launched with %q (target %v)", argv, sum)
	}
	if res["handoff"].(map[string]any)["agentId"] != nil {
		t.Fatalf("handoff row = %v", res["handoff"])
	}
}

func TestHandoffPiAgentLandingBrief(t *testing.T) {
	ts, _, dataDir, home := handoffServer(t)
	proj := filepath.Join(t.TempDir(), "proj")
	if err := os.MkdirAll(proj, 0o755); err != nil {
		t.Fatal(err)
	}
	path := seedHandoffClaude(t, home, proj)

	// No pi CLI binary installed: the agent landing needs no terminal, so
	// the handoff must still succeed.
	res := cliRequest(t, ts, "POST", "/api/clis/claude-code/sessions/handoff", map[string]any{"id": "cc-1", "path": path, "cwd": proj, "to": "pi", "landing": "agent", "mode": "brief"}, 201)
	if res["terminal"] != nil {
		t.Fatal("an agent landing must not open a terminal")
	}
	if res["landing"] != "agent" {
		t.Fatalf("landing = %v", res["landing"])
	}
	agent := res["agent"].(map[string]any)
	if agent["mode"] != "stopped" || agent["name"] != "Race fix" {
		t.Fatalf("agent = %v", agent)
	}
	brief := res["brief"].(map[string]any)["path"].(string)
	if !strings.HasPrefix(brief, filepath.Join(dataDir, "handoffs")) {
		t.Fatalf("brief path = %s", brief)
	}
	req, _ := http.NewRequest("GET", ts.URL+"/api/agents/"+agent["id"].(string)+"/tasks", nil)
	res2 := do(t, ts.Client(), req)
	b, _ := io.ReadAll(res2.Body)
	if res2.StatusCode != 200 {
		t.Fatalf("tasks: %d: %s", res2.StatusCode, b)
	}
	var tasks []map[string]any
	if err := json.Unmarshal(b, &tasks); err != nil {
		t.Fatal(err)
	}
	if len(tasks) != 1 {
		t.Fatalf("tasks = %v", tasks)
	}
	task := tasks[0]
	if task["kind"] != "prompt" || task["status"] != "queued" || !strings.Contains(task["payload"].(string), brief) {
		t.Fatalf("task = %v", task)
	}
	if res["handoff"].(map[string]any)["agentId"] != agent["id"] {
		t.Fatalf("handoff row = %v", res["handoff"])
	}
}

func TestHandoffLandingValidation(t *testing.T) {
	ts, _, _, home := handoffServer(t)
	proj := filepath.Join(t.TempDir(), "proj")
	if err := os.MkdirAll(proj, 0o755); err != nil {
		t.Fatal(err)
	}
	path := seedHandoffClaude(t, home, proj)
	body := map[string]any{"id": "cc-1", "path": path, "cwd": proj, "to": "codex", "landing": "agent"}
	cliRequest(t, ts, "POST", "/api/clis/claude-code/sessions/handoff/preview", body, 400)
	body["landing"] = "middlet"
	body["to"] = "pi"
	cliRequest(t, ts, "POST", "/api/clis/claude-code/sessions/handoff/preview", body, 400)
}

func TestHandoffToPiAdoptsAgent(t *testing.T) {
	ts, _, _, home := handoffServer(t)
	old := session.TestRoot
	session.TestRoot = filepath.Join(home, ".pi", "agent", "sessions")
	defer func() { session.TestRoot = old }()
	proj := filepath.Join(t.TempDir(), "proj")
	if err := os.MkdirAll(proj, 0o755); err != nil {
		t.Fatal(err)
	}
	ws := cliRequest(t, ts, "POST", "/api/workspaces", map[string]any{"name": "proj", "path": proj}, 201)
	path := seedHandoffClaude(t, home, proj)

	res := cliRequest(t, ts, "POST", "/api/clis/claude-code/sessions/handoff", map[string]any{"id": "cc-1", "path": path, "cwd": proj, "to": "pi"}, 201)
	if res["terminal"] != nil {
		t.Fatal("a pi handoff must not open a terminal")
	}
	agent := res["agent"].(map[string]any)
	if agent["mode"] != "stopped" || agent["workspaceId"] != ws["id"] || agent["name"] != "Race fix" {
		t.Fatalf("agent = %v", agent)
	}
	sp := agent["sessionPath"].(string)
	if !strings.HasPrefix(sp, session.AgentDir(agent["id"].(string))) {
		t.Fatalf("session path %s is not under the agent's private dir", sp)
	}
	sum, err := session.Summarize(sp)
	if err != nil || sum.Cwd != proj || sum.Name != "Race fix" {
		t.Fatalf("pi summary = %+v %v", sum, err)
	}
	h := res["handoff"].(map[string]any)
	if h["agentId"] != agent["id"] || h["targetCli"] != "pi" || h["terminalId"] != nil {
		t.Fatalf("handoff row = %v", h)
	}
	// The pi listing for the workspace shows it in use by that agent, with lineage.
	list := cliRequest(t, ts, "GET", "/api/clis/pi/sessions?workspace="+ws["id"].(string), nil, 200)
	row := list["sessions"].([]any)[0].(map[string]any)
	if row["inUseBy"] == nil || row["handoff"].(map[string]any)["from"].(map[string]any)["cli"] != "claude-code" {
		t.Fatalf("pi row = %v", row)
	}
}

func TestLiveHolderForInteractivePi(t *testing.T) {
	manager := tmux.New()
	if !manager.Available() {
		t.Skip("tmux missing")
	}
	st := testStore(t)
	dir := t.TempDir()
	_, agent, err := storeWorkspaceWithAgent(st, "App", dir)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "session.jsonl")
	if err := os.WriteFile(path, []byte(`{"type":"session","id":"s1"}`+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := st.UpdateAgent(agent.ID, store.AgentPatch{SessionPath: &path}); err != nil {
		t.Fatal(err)
	}
	name := tmux.SessionName(agent.ID)
	if err := manager.NewSessionEnv(context.Background(), name, dir, nil, "/bin/sh", "-c", "sleep 300"); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = manager.KillSession(context.Background(), name) })
	deps := Deps{Store: st, Tmux: manager, Runtime: rpc.NewRuntime("cat", st, nil)}
	live := liveHolderFor(deps, "pi", clisession.Ref{Path: path}, transcript.Timeline{})
	if live == nil || live.Kind != "agent" || live.ID != agent.ID {
		t.Fatalf("interactive live = %+v", live)
	}
	_ = manager.KillSession(context.Background(), name)
	if liveHolderFor(deps, "pi", clisession.Ref{Path: path}, transcript.Timeline{}) != nil {
		t.Fatal("stopped interactive still counted as live")
	}
}

func TestHandoffLiveSourceNeedsForce(t *testing.T) {
	ts, deps, _, home := handoffServer(t)
	proj := filepath.Join(t.TempDir(), "proj")
	if err := os.MkdirAll(proj, 0o755); err != nil {
		t.Fatal(err)
	}
	path := seedHandoffClaude(t, home, proj)
	fakeCLI(t, ts, home, "codex")
	_, claudeOut := fakeCLI(t, ts, home, "claude-code")

	// A running Claude Code terminal pinned to cc-1 (ADR-0084).
	created := launchFixture(t, ts, "claude-code", map[string]any{"name": "live claude", "cwd": proj}, 201)
	termID := created["id"].(string)
	t.Cleanup(func() { killTermPane(tmux.ShellSessionName(termID)) })
	waitCLIFile(t, claudeOut+".args")
	if _, ok := deps.TermRuntimes.Start(termID, TermRuntime{CLI: "claude-code", Source: "wrapper", RunID: "run-1", StartedAt: time.Now()}); !ok {
		t.Fatal("runtime not registered")
	}
	if err := deps.Store.SetTerminalLastSession(termID, store.TerminalLastSession{CLI: "claude-code", SessionID: "cc-1", Path: path, Cwd: proj, UpdatedAt: "2026-09-01T10:00:05Z"}); err != nil {
		t.Fatal(err)
	}

	body := map[string]any{"id": "cc-1", "path": path, "cwd": proj, "to": "codex"}
	p := cliRequest(t, ts, "POST", "/api/clis/claude-code/sessions/handoff/preview", body, 200)
	live := p["live"].(map[string]any)
	if live["kind"] != "terminal" || live["id"] != termID || live["name"] != "live claude" {
		t.Fatalf("preview live = %v", p["live"])
	}
	r := cliRequestFull(t, ts, "POST", "/api/clis/claude-code/sessions/handoff", body)
	if r["status"] != "409" || r["body"].(map[string]any)["live"] == nil {
		t.Fatalf("live source without force: %v", r)
	}
	if rollouts, _ := filepath.Glob(filepath.Join(home, ".codex", "sessions", "*", "*", "*", "*.jsonl")); len(rollouts) != 0 {
		t.Fatal("a refused handoff wrote a rollout")
	}
	body["force"] = true
	res := cliRequest(t, ts, "POST", "/api/clis/claude-code/sessions/handoff", body, 201)
	cleanupTerm(t, res)
	warnings := res["manifest"].(map[string]any)["warnings"].([]any)
	if len(warnings) != 1 || !strings.Contains(warnings[0].(string), "live claude was still writing") {
		t.Fatalf("forced handoff must warn: %v", warnings)
	}
}
