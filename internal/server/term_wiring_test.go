package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cfpperche/picode/internal/store"
)

func wiringTestServer(t *testing.T) (*httptest.Server, string) {
	t.Helper()
	root := t.TempDir()
	t.Setenv("HOME", root)
	t.Setenv("USERPROFILE", root)
	dataDir := filepath.Join(root, "data")
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		t.Fatal(err)
	}
	st, err := store.Open(filepath.Join(dataDir, "picode.db"))
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	ts := httptest.NewServer(New("127.0.0.1:0", Deps{
		Store: st, DataDir: dataDir, TermStates: NewTermStates(),
	}).Handler)
	t.Cleanup(ts.Close)
	return ts, dataDir
}

func TestInterceptDoesNotWriteUserClaudeSettings(t *testing.T) {
	ts, dataDir := wiringTestServer(t)
	home, _ := os.UserHomeDir()
	userSettings := filepath.Join(home, ".claude", "settings.json")

	if res := postJSON(t, ts, "/api/terminals/wiring/claude-code/enable", map[string]any{}); res.StatusCode != http.StatusOK {
		t.Fatalf("enable = %d", res.StatusCode)
	}
	if _, err := os.Stat(userSettings); !os.IsNotExist(err) {
		t.Fatalf("must not create %s: %v", userSettings, err)
	}
	wrap := wrapperPath(dataDir, "claude")
	body, err := os.ReadFile(wrap)
	if err != nil {
		t.Fatalf("wrapper missing: %v", err)
	}
	if !strings.Contains(string(body), "--settings") {
		t.Fatalf("wrapper does not inject --settings:\n%s", body)
	}
	if !strings.Contains(string(body), claudeSettingsFile(dataDir)) {
		t.Fatal("wrapper must point at the data-dir settings file")
	}
	rawSettings, err := os.ReadFile(claudeSettingsFile(dataDir))
	if err != nil {
		t.Fatalf("intercept settings missing: %v", err)
	}
	if !strings.Contains(string(rawSettings), "TaskCompleted") || !strings.Contains(string(rawSettings), " auto claude-code") {
		t.Fatalf("settings should map Stop/TaskCompleted via auto:\n%s", rawSettings)
	}
	pathEnv := interceptSessionPath(dataDir)
	if !strings.HasPrefix(pathEnv, "PATH="+interceptBinDir(dataDir)) {
		t.Fatalf("session PATH = %q", pathEnv)
	}

	var page struct {
		Clis []wiringRow `json:"clis"`
	}
	res := do(t, ts.Client(), mustGet(t, ts.URL+"/api/terminals/wiring"))
	_ = json.NewDecoder(res.Body).Decode(&page)
	var row *wiringRow
	for i := range page.Clis {
		if page.Clis[i].ID == "claude-code" {
			row = &page.Clis[i]
		}
	}
	if row == nil || !row.Wired {
		t.Fatalf("status = %+v, want wired", page.Clis)
	}

	if res := postJSON(t, ts, "/api/terminals/wiring/claude-code/disable", map[string]any{}); res.StatusCode != http.StatusOK {
		t.Fatalf("disable = %d", res.StatusCode)
	}
	if _, err := os.Stat(wrap); !os.IsNotExist(err) {
		t.Fatal("wrapper should be gone after disable")
	}
	if interceptSessionPath(dataDir) != "" {
		t.Fatal("PATH must not be prepended when nothing is intercepting")
	}
	if interceptBinEnv(dataDir) != "" {
		t.Fatal("PICODE_INTERCEPT_BIN must be empty when nothing is intercepting")
	}
}

func TestInterceptCodexAndGrok(t *testing.T) {
	ts, dataDir := wiringTestServer(t)
	if res := postJSON(t, ts, "/api/terminals/wiring/codex/enable", map[string]any{}); res.StatusCode != http.StatusOK {
		t.Fatalf("codex enable = %d", res.StatusCode)
	}
	body, _ := os.ReadFile(wrapperPath(dataDir, "codex"))
	for _, want := range []string{"hooks.UserPromptSubmit", "hooks.PermissionRequest", "hooks.Interrupt", "hooks.state=", "notify="} {
		if !strings.Contains(string(body), want) {
			t.Fatalf("codex wrapper missing %q:\n%s", want, body)
		}
	}
	if strings.Contains(string(body), "exec \"$real\" --dangerously-bypass-hook-trust") {
		t.Fatalf("codex wrapper must trust only PiCode hooks, not bypass all hook trust:\n%s", body)
	}
	if res := postJSON(t, ts, "/api/terminals/wiring/grok/enable", map[string]any{}); res.StatusCode != http.StatusOK {
		t.Fatalf("grok enable = %d", res.StatusCode)
	}
	body, _ = os.ReadFile(wrapperPath(dataDir, "grok"))
	if !strings.Contains(string(body), "GROK_HOME=") {
		t.Fatalf("grok wrapper:\n%s", body)
	}
	hookJSON := filepath.Join(grokHomeDir(dataDir), "hooks", "picode.json")
	raw, err := os.ReadFile(hookJSON)
	if err != nil {
		t.Fatalf("grok hooks missing: %v", err)
	}
	if !strings.Contains(string(raw), "UserPromptSubmit") {
		t.Fatalf("grok hooks: %s", raw)
	}
}

func TestInterceptHermesPythonPathHooks(t *testing.T) {
	ts, dataDir := wiringTestServer(t)
	if res := postJSON(t, ts, "/api/terminals/wiring/hermes/enable", map[string]any{}); res.StatusCode != http.StatusOK {
		t.Fatalf("hermes enable = %d", res.StatusCode)
	}
	body, err := os.ReadFile(wrapperPath(dataDir, "hermes"))
	if err != nil {
		t.Fatalf("hermes wrapper missing: %v", err)
	}
	got := string(body)
	if !strings.Contains(got, "name=hermes") || !strings.Contains(got, "runtime-start") {
		t.Fatalf("hermes wrapper missing presence lease:\n%s", got)
	}
	if !strings.Contains(got, "PYTHONPATH=") || !strings.Contains(got, "HERMES_ACCEPT_HOOKS=1") || !strings.Contains(got, "unset PYTHONPATH") {
		t.Fatalf("hermes wrapper missing PYTHONPATH trampoline unwrap:\n%s", got)
	}
	if strings.Contains(got, "HERMES_HOME=") {
		t.Fatalf("hermes wrapper must not overlay HERMES_HOME:\n%s", got)
	}
	home, _ := os.UserHomeDir()
	if _, err := os.Stat(filepath.Join(home, ".hermes", "config.yaml")); !os.IsNotExist(err) {
		t.Fatalf("enable must not write ~/.hermes/config.yaml: %v", err)
	}
	site := filepath.Join(hermesPythonDir(dataDir), "sitecustomize.py")
	raw, err := os.ReadFile(site)
	if err != nil {
		t.Fatalf("sitecustomize missing: %v", err)
	}
	src := string(raw)
	if !strings.Contains(src, "pre_llm_call") || !strings.Contains(src, "PICODE_HERMES_HOOK") {
		t.Fatalf("sitecustomize: %s", src)
	}
	if !strings.Contains(src, "register_from_config") || strings.Contains(src, "load_config_readonly") {
		t.Fatalf("sitecustomize must patch register_from_config, not load_config:\n%s", src)
	}
	var page struct {
		Clis []wiringRow `json:"clis"`
	}
	res := do(t, ts.Client(), mustGet(t, ts.URL+"/api/terminals/wiring"))
	_ = json.NewDecoder(res.Body).Decode(&page)
	var row *wiringRow
	for i := range page.Clis {
		if page.Clis[i].ID == "hermes" {
			row = &page.Clis[i]
		}
	}
	if row == nil || !row.Wired || !strings.Contains(row.Note, "PYTHONPATH") {
		t.Fatalf("hermes wiring = %+v", page.Clis)
	}
	if strings.Contains(row.Note, "Presence lease only") {
		t.Fatalf("stale Hermes note: %q", row.Note)
	}
}

func TestHermesTrampolineUnwrapSetsPYTHONPATH(t *testing.T) {
	if _, err := exec.LookPath("sh"); err != nil {
		t.Skip("sh not on PATH")
	}
	root := t.TempDir()
	dataDir := filepath.Join(root, "data")
	wrapper, hookLog, envLog := writeHermesInnerProbe(t, dataDir, root)
	s := runHermesWrapper(t, wrapper, root, hookLog, envLog, "--tui")
	if !strings.Contains(s, "PYTHONPATH="+hermesPythonDir(dataDir)) {
		t.Fatalf("inner PYTHONPATH:\n%s", s)
	}
	if !strings.Contains(s, "auto hermes") || !strings.Contains(s, "ARGS=--accept-hooks --tui") {
		t.Fatalf("inner hook/args:\n%s", s)
	}
}

func writeHermesInnerProbe(t *testing.T, dataDir, root string) (wrapper, hookLog, envLog string) {
	t.Helper()
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		t.Fatal(err)
	}
	hookLog = filepath.Join(root, "hooks.log")
	envLog = filepath.Join(root, "inner.env")
	if _, err := ensureHookScript(dataDir); err != nil {
		t.Fatal(err)
	}
	if err := writeHermesIntercept(dataDir, hookScriptPath(dataDir)); err != nil {
		t.Fatal(err)
	}
	if err := writeExecutable(hookScriptPath(dataDir), "#!/bin/sh\nprintf '%s|%s\\n' \"$1\" \"$2\" >> \"$PICODE_TEST_HOOK_LOG\"\n"); err != nil {
		t.Fatal(err)
	}
	inner := filepath.Join(root, "inner-hermes")
	if err := writeExecutable(inner, "#!/bin/sh\nprintf 'PYTHONPATH=%s\\nHOOK=%s\\nARGS=%s\\n' \"$PYTHONPATH\" \"$PICODE_HERMES_HOOK\" \"$*\" > \"$PICODE_TEST_INNER\"\n"); err != nil {
		t.Fatal(err)
	}
	outer := filepath.Join(root, "hermes")
	if err := writeExecutable(outer, "#!/usr/bin/env bash\nunset PYTHONPATH\nunset PYTHONHOME\nexec \""+inner+"\" \"$@\"\n"); err != nil {
		t.Fatal(err)
	}
	return wrapperPath(dataDir, "hermes"), hookLog, envLog
}

func runHermesWrapper(t *testing.T, wrapper, root, hookLog, envLog string, args ...string) string {
	t.Helper()
	_ = os.Remove(envLog)
	cmd := exec.Command(wrapper, args...)
	cmd.Env = append(os.Environ(),
		"PATH="+filepath.Dir(wrapper)+string(os.PathListSeparator)+root+string(os.PathListSeparator)+"/usr/bin:/bin",
		"PICODE_TERM_ID=terminal-test",
		"PICODE_TEST_HOOK_LOG="+hookLog,
		"PICODE_TEST_INNER="+envLog,
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("wrapper %v: %v: %s", args, err, out)
	}
	got, err := os.ReadFile(envLog)
	if err != nil {
		t.Fatal(err)
	}
	return string(got)
}

func TestHermesSubcommandSkipsPYTHONPATH(t *testing.T) {
	if _, err := exec.LookPath("sh"); err != nil {
		t.Skip("sh not on PATH")
	}
	root := t.TempDir()
	dataDir := filepath.Join(root, "data")
	wrapper, hookLog, envLog := writeHermesInnerProbe(t, dataDir, root)

	cases := []struct {
		name    string
		args    []string
		inject  bool
		wantArg string
	}{
		{name: "bare tui", args: []string{"--tui"}, inject: true, wantArg: "--accept-hooks --tui"},
		{name: "resume", args: []string{"--resume", "sess-1"}, inject: true, wantArg: "--accept-hooks --resume sess-1"},
		{name: "chat", args: []string{"chat"}, inject: true, wantArg: "--accept-hooks chat"},
		{name: "prompt args", args: []string{"two words", "$literal"}, inject: true, wantArg: "--accept-hooks two words $literal"},
		{name: "setup", args: []string{"setup"}, wantArg: "setup"},
		{name: "profile then setup", args: []string{"-p", "private", "setup"}, wantArg: "-p private setup"},
		{name: "version", args: []string{"version"}, wantArg: "version"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := runHermesWrapper(t, wrapper, root, hookLog, envLog, tc.args...)
			if !strings.Contains(s, "ARGS="+tc.wantArg) {
				t.Fatalf("args:\n%s", s)
			}
			hasPy := strings.Contains(s, "PYTHONPATH="+hermesPythonDir(dataDir))
			if hasPy != tc.inject {
				t.Fatalf("inject=%v PYTHONPATH:\n%s", tc.inject, s)
			}
		})
	}
}

func TestHermesSitecustomizeDecisionTable(t *testing.T) {
	if _, err := exec.LookPath("python3"); err != nil {
		t.Skip("python3 not on PATH")
	}
	root := t.TempDir()
	dataDir := filepath.Join(root, "data")
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := writeHermesIntercept(dataDir, "/tmp/picode-hook"); err != nil {
		t.Fatal(err)
	}
	pyDir := hermesPythonDir(dataDir)
	agentDir := filepath.Join(pyDir, "agent")
	if err := os.MkdirAll(agentDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(agentDir, "__init__.py"), []byte(""), 0o600); err != nil {
		t.Fatal(err)
	}
	fake := `def register_from_config(cfg=None, *args, **kwargs):
    return {"cfg": cfg, "kwargs": kwargs}
`
	if err := os.WriteFile(filepath.Join(agentDir, "shell_hooks.py"), []byte(fake), 0o600); err != nil {
		t.Fatal(err)
	}
	harness := filepath.Join(root, "harness.py")
	body := `import json, os, sys
import agent.shell_hooks as sh
orig = {"hooks": {"pre_llm_call": [{"command": "user-hook"}]}, "model": "keep"}
got = sh.register_from_config(orig, accept_hooks=True)
print(json.dumps({"orig": orig, "got": got, "pythonpath": os.environ.get("PYTHONPATH")}))
`
	if err := os.WriteFile(harness, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}

	run := func(hook string) map[string]any {
		t.Helper()
		cmd := exec.Command("python3", harness)
		cmd.Dir = root
		env := []string{
			"PATH=/usr/bin:/bin",
			"PYTHONPATH=" + pyDir,
			"HOME=" + root,
		}
		if hook != "" {
			env = append(env, "PICODE_HERMES_HOOK="+hook)
		}
		cmd.Env = env
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("python harness hook=%q: %v: %s", hook, err, out)
		}
		var v map[string]any
		if err := json.Unmarshal(out, &v); err != nil {
			t.Fatalf("json %q: %v", out, err)
		}
		return v
	}

	// Decision table: empty hook is a no-op; a set hook merges a deepcopy
	// and strips this sitecustomize directory from child PYTHONPATH.
	empty := run("")
	if empty["pythonpath"] != nil && empty["pythonpath"] != "" {
		t.Fatalf("PYTHONPATH leaked without hook: %v", empty["pythonpath"])
	}
	orig := empty["orig"].(map[string]any)
	if orig["model"] != "keep" {
		t.Fatalf("orig lost: %v", orig)
	}

	hook := "/tmp/picode-hook auto hermes"
	hit := run(hook)
	if hit["pythonpath"] != nil && hit["pythonpath"] != "" {
		t.Fatalf("PYTHONPATH leaked after sitecustomize: %v", hit["pythonpath"])
	}
	orig = hit["orig"].(map[string]any)
	hooks := orig["hooks"].(map[string]any)["pre_llm_call"].([]any)
	if len(hooks) != 1 || hooks[0].(map[string]any)["command"] != "user-hook" {
		t.Fatalf("original cfg mutated: %v", orig)
	}
	got := hit["got"].(map[string]any)["cfg"].(map[string]any)
	if got["hooks_auto_accept"] != true {
		t.Fatalf("merged cfg missing auto-accept: %v", got)
	}
	merged := got["hooks"].(map[string]any)["pre_llm_call"].([]any)
	if len(merged) != 2 {
		t.Fatalf("merged hooks = %v", merged)
	}
	cmds := []string{merged[0].(map[string]any)["command"].(string), merged[1].(map[string]any)["command"].(string)}
	if cmds[0] != "user-hook" || cmds[1] != hook {
		t.Fatalf("merged commands = %v", cmds)
	}
	if _, ok := got["hooks"].(map[string]any)["pre_approval_request"]; !ok {
		t.Fatalf("missing approval hook: %v", got["hooks"])
	}
}

func TestInterceptPi(t *testing.T) {
	ts, dataDir := wiringTestServer(t)
	home, _ := os.UserHomeDir()
	piHome := filepath.Join(home, ".pi")
	if err := os.MkdirAll(piHome, 0o755); err != nil {
		t.Fatal(err)
	}
	sentinel := filepath.Join(piHome, "keep-me")
	if err := os.WriteFile(sentinel, []byte("user-owned\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	if res := postJSON(t, ts, "/api/terminals/wiring/pi/enable", map[string]any{}); res.StatusCode != http.StatusOK {
		t.Fatalf("pi enable = %d", res.StatusCode)
	}
	wrapper := wrapperPath(dataDir, "pi")
	extension := piTerminalStateExtensionFile(dataDir)
	wrapperBody, err := os.ReadFile(wrapper)
	if err != nil {
		t.Fatalf("pi wrapper missing: %v", err)
	}
	if !strings.Contains(string(wrapperBody), quotedCLIArgs([]string{"-e", extension})+" \"$@\"") {
		t.Fatalf("pi wrapper does not prepend the state extension and preserve argv:\n%s", wrapperBody)
	}
	if !strings.Contains(string(wrapperBody), `auth|config|install|list|remove|uninstall|update`) {
		t.Fatalf("pi wrapper does not preserve subcommand dispatch:\n%s", wrapperBody)
	}
	extensionBody, err := os.ReadFile(extension)
	if err != nil {
		t.Fatalf("pi extension missing: %v", err)
	}
	for _, want := range []string{
		`ctx.mode !== "tui"`, `process.env.PICODE_TERM_ID`,
		`pi.on("agent_start"`, `pi.on("ui_prompt_start"`,
		`pi.on("ui_prompt_end"`, `ctx.isIdle()`, `pi.on("agent_settled"`,
	} {
		if !strings.Contains(string(extensionBody), want) {
			t.Fatalf("pi extension missing %q:\n%s", want, extensionBody)
		}
	}
	if got, err := os.ReadFile(sentinel); err != nil || string(got) != "user-owned\n" {
		t.Fatalf("enable changed ~/.pi sentinel: %q, %v", got, err)
	}

	// The wrapper prepends only PiCode's extension. Subcommands, flags, and
	// user-supplied extensions remain byte-for-byte arguments to the real pi.
	realDir := t.TempDir()
	argLog := filepath.Join(t.TempDir(), "argv")
	realPi := filepath.Join(realDir, "pi")
	if err := writeExecutable(realPi, "#!/bin/sh\nprintf '%s\\n' \"$@\" > \"$PICODE_TEST_ARGV\"\n"); err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name   string
		args   []string
		inject bool
	}{
		{name: "version", args: []string{"--version"}},
		{name: "auth command", args: []string{"auth", "check"}},
		{name: "install command", args: []string{"install", "git:example/pi-package"}},
		{name: "TUI prompt", args: []string{"hello"}, inject: true},
		{name: "user extension", args: []string{"-e", "/tmp/user extension.ts", "--version"}, inject: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_ = os.Remove(argLog)
			cmd := exec.Command(wrapper, tc.args...)
			cmd.Env = []string{
				"PATH=" + interceptBinDir(dataDir) + string(os.PathListSeparator) + realDir + string(os.PathListSeparator) + "/usr/bin:/bin",
				"PICODE_TEST_ARGV=" + argLog,
			}
			if out, err := cmd.CombinedOutput(); err != nil {
				t.Fatalf("wrapper: %v: %s", err, out)
			}
			got, err := os.ReadFile(argLog)
			if err != nil {
				t.Fatal(err)
			}
			wantArgs := append([]string(nil), tc.args...)
			if tc.inject {
				wantArgs = append([]string{"-e", extension}, wantArgs...)
			}
			want := strings.Join(wantArgs, "\n") + "\n"
			if string(got) != want {
				t.Fatalf("argv = %q, want %q", got, want)
			}
		})
	}

	var page struct {
		Clis []wiringRow `json:"clis"`
	}
	res := do(t, ts.Client(), mustGet(t, ts.URL+"/api/terminals/wiring"))
	_ = json.NewDecoder(res.Body).Decode(&page)
	var piRow *wiringRow
	for i := range page.Clis {
		if page.Clis[i].ID == "pi" {
			piRow = &page.Clis[i]
		}
		if page.Clis[i].ID == "codex" && strings.Contains(page.Clis[i].Note, "End-of-turn only") {
			t.Fatalf("stale Codex note: %q", page.Clis[i].Note)
		}
	}
	if piRow == nil || !piRow.Wired {
		t.Fatalf("Pi status = %+v, want wired", page.Clis)
	}

	if res := postJSON(t, ts, "/api/terminals/wiring/pi/disable", map[string]any{}); res.StatusCode != http.StatusOK {
		t.Fatalf("pi disable = %d", res.StatusCode)
	}
	// ADR-0069 keeps generated support files for existing processes; future
	// manual launches lose their wrapper entry point.
	for _, path := range []string{wrapper} {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatalf("%s should be gone after disable: %v", path, err)
		}
	}
	if _, err := os.Stat(extension); err != nil {
		t.Fatalf("existing process extension removed: %v", err)
	}
	if got, err := os.ReadFile(sentinel); err != nil || string(got) != "user-owned\n" {
		t.Fatalf("disable changed ~/.pi sentinel: %q, %v", got, err)
	}
}

func TestPiTerminalStateExtensionDecisionTable(t *testing.T) {
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("node not on PATH")
	}
	ts, dataDir := wiringTestServer(t)
	if res := postJSON(t, ts, "/api/terminals/wiring/pi/enable", map[string]any{}); res.StatusCode != http.StatusOK {
		t.Fatalf("pi enable = %d", res.StatusCode)
	}

	// Replace the normal HTTP reporter with a deterministic recorder, then
	// load the generated JavaScript-compatible .ts source as an ES module.
	logPath := filepath.Join(t.TempDir(), "states.log")
	recorder := "#!/bin/sh\nprintf '%s|%s\\n' \"$1\" \"$2\" >> \"$PICODE_TEST_LOG\"\n"
	if err := writeExecutable(hookScriptPath(dataDir), recorder); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(piTerminalStateExtensionFile(dataDir))
	if err != nil {
		t.Fatal(err)
	}
	modulePath := filepath.Join(t.TempDir(), "pi-terminal-state.mjs")
	if err := os.WriteFile(modulePath, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	harnessPath := filepath.Join(t.TempDir(), "fire-event.mjs")
	harness := `import { pathToFileURL } from "node:url";
const [modulePath, event, mode, idle] = process.argv.slice(2);
const { default: load } = await import(pathToFileURL(modulePath).href);
const handlers = new Map();
load({ on(name, handler) { handlers.set(name, handler); } });
const handler = handlers.get(event);
if (!handler) throw new Error(` + "`missing handler: ${event}`" + `);
await handler({}, { mode, isIdle: () => idle === "true" });
`
	if err := os.WriteFile(harnessPath, []byte(harness), 0o600); err != nil {
		t.Fatal(err)
	}

	// Decision table: both guards must pass; each native lifecycle event then
	// maps to exactly one terminal state. ui_prompt_end resumes the state that
	// was active before the blocking prompt.
	cases := []struct {
		name    string
		event   string
		mode    string
		hasTerm bool
		idle    bool
		want    string
	}{
		{name: "missing terminal id", event: "agent_start", mode: "tui"},
		{name: "managed RPC agent", event: "agent_start", mode: "rpc", hasTerm: true},
		{name: "noninteractive print", event: "agent_start", mode: "print", hasTerm: true},
		{name: "JSON stream", event: "agent_start", mode: "json", hasTerm: true},
		{name: "session starts quiet", event: "session_start", mode: "tui", hasTerm: true, want: "idle|pi\n"},
		{name: "agent starts working", event: "agent_start", mode: "tui", hasTerm: true, want: "working|pi\n"},
		{name: "UI prompt needs user", event: "ui_prompt_start", mode: "tui", hasTerm: true, want: "needs-you|pi\n"},
		{name: "UI prompt returns to work", event: "ui_prompt_end", mode: "tui", hasTerm: true, want: "working|pi\n"},
		{name: "idle UI prompt stays idle", event: "ui_prompt_end", mode: "tui", hasTerm: true, idle: true, want: "idle|pi\n"},
		{name: "agent settles idle", event: "agent_settled", mode: "tui", hasTerm: true, want: "idle|pi\n"},
		{name: "session shutdown idle", event: "session_shutdown", mode: "tui", hasTerm: true, want: "idle|pi\n"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_ = os.Remove(logPath)
			cmd := exec.Command("node", harnessPath, modulePath, tc.event, tc.mode, fmt.Sprint(tc.idle))
			cmd.Env = []string{"PATH=/usr/bin:/bin", "PICODE_TEST_LOG=" + logPath}
			if tc.hasTerm {
				cmd.Env = append(cmd.Env, "PICODE_TERM_ID=terminal-test")
			}
			if out, err := cmd.CombinedOutput(); err != nil {
				t.Fatalf("node harness: %v: %s", err, out)
			}
			got, err := os.ReadFile(logPath)
			if os.IsNotExist(err) {
				got = nil
			} else if err != nil {
				t.Fatal(err)
			}
			if string(got) != tc.want {
				t.Fatalf("report = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestInterceptBashrcRefusesUnknownDataDir(t *testing.T) {
	if path, err := ensureInterceptBashrc(""); err == nil || path != "" {
		t.Fatalf("empty data dir = %q, %v; want refusal", path, err)
	}
}

func TestInterceptRefusesUnknownCLI(t *testing.T) {
	ts, _ := wiringTestServer(t)
	res := postJSON(t, ts, "/api/terminals/wiring/antigravity/enable", map[string]any{})
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("unknown CLI = %d, want 400", res.StatusCode)
	}
}

func TestStripLegacyUserClaudeHooks(t *testing.T) {
	ts, _ := wiringTestServer(t)
	p, err := claudeSettingsPath()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	doc := map[string]any{
		"model": "claude-opus-4-8",
		"hooks": map[string]any{
			"Stop": []any{
				map[string]any{"hooks": []any{map[string]any{"type": "command", "command": "/usr/bin/say done"}}},
				map[string]any{"hooks": []any{map[string]any{"type": "command", "command": "/x/picode-hook idle claude-code"}}},
			},
		},
	}
	raw, _ := json.MarshalIndent(doc, "", "  ")
	if err := os.WriteFile(p, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	if res := postJSON(t, ts, "/api/terminals/wiring/claude-code/enable", map[string]any{}); res.StatusCode != http.StatusOK {
		t.Fatalf("enable = %d", res.StatusCode)
	}
	got, _ := os.ReadFile(p)
	var after map[string]any
	_ = json.Unmarshal(got, &after)
	if after["model"] != "claude-opus-4-8" {
		t.Fatalf("user model lost: %s", got)
	}
	if strings.Contains(string(got), wiringMarker) {
		t.Fatalf("legacy marker still in user settings: %s", got)
	}
	stop := after["hooks"].(map[string]any)["Stop"].([]any)
	if len(stop) != 1 {
		t.Fatalf("Stop groups = %d, want only the user group", len(stop))
	}
}

func TestHookMapPy(t *testing.T) {
	if _, err := exec.LookPath("python3"); err != nil {
		t.Skip("python3 not on PATH")
	}
	dir := t.TempDir()
	if _, err := ensureHookScript(dir); err != nil {
		t.Fatal(err)
	}
	run := func(in string) string {
		t.Helper()
		cmd := exec.Command("python3", filepath.Join(dir, "picode-hook-map.py"))
		cmd.Stdin = strings.NewReader(in)
		out, err := cmd.Output()
		if err != nil {
			t.Fatalf("map %q: %v", in, err)
		}
		return string(out)
	}
	cases := []struct {
		in, want string
	}{
		{`{"hook_event_name":"UserPromptSubmit"}`, "working\n"},
		{`{"hook_event_name":"SessionStart"}`, "idle\n"},
		{`{"hook_event_name":"Stop"}`, "idle\n"},
		{`{"hook_event_name":"TaskCompleted"}`, "idle\n"},
		{`{"type":"agent-turn-complete"}`, "idle\n"},
		{`{"hook_event_name":"Interrupt"}`, "idle\n"},
		{`{"hook_event_name":"PermissionRequest"}`, "needs-you\n"},
		{`{"hook_event_name":"Notification","notification_type":"permission_prompt"}`, "needs-you\n"},
		{`{"hook_event_name":"pre_llm_call"}`, "working\n"},
		{`{"hook_event_name":"post_approval_response"}`, "working\n"},
		{`{"hook_event_name":"on_session_start"}`, "idle\n"},
		{`{"hook_event_name":"on_session_end"}`, "idle\n"},
		{`{"hook_event_name":"post_llm_call"}`, "idle\n"},
		{`{"hook_event_name":"subagent_stop"}`, "idle\n"},
		{`{"hook_event_name":"on_session_finalize"}`, "idle\n"},
		{`{"hook_event_name":"on_session_reset"}`, "idle\n"},
		{`{"hook_event_name":"pre_approval_request"}`, "needs-you\n"},
		{`{"hook_event_name":"pre_tool_call"}`, ""},
	}
	for _, tc := range cases {
		if got := run(tc.in); got != tc.want {
			t.Fatalf("%s = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestCodexHookHashMatchesCodexFingerprint(t *testing.T) {
	// Captured from Codex 0.153.0 hooks/list for this exact normalized hook.
	spec := codexHookSpec{key: "user_prompt_submit", timeoutSec: 600}
	got := codexHookHash(spec, "/tmp/codex-hook-test.sh UserPromptSubmit")
	want := "sha256:d195511c28b02bd5cb782e8f7e489c9316ac02f7d42e2b64130eff27afe5f1cb"
	if got != want {
		t.Fatalf("hash = %q, want Codex fingerprint %q", got, want)
	}

	// Go's encoding/json escapes HTML by default; serde_json (which Codex
	// fingerprints) does not. A legal data-dir containing these characters
	// must still produce the command hash Codex trusts.
	spec = codexHookSpec{key: "user_prompt_submit", timeoutSec: 5}
	got = codexHookHash(spec, "/tmp/<picode>& hook")
	want = "sha256:5328cb425a3aeb63b4eb7c137e1cb84d42f52f12165fe7a4b5f03d7e76731a35"
	if got != want {
		t.Fatalf("HTML-character hash = %q, want %q", got, want)
	}
}

func TestHookScriptTalksToLocalhostInsecure(t *testing.T) {
	if !strings.Contains(hookScriptTmpl, "curl -fsSk") {
		t.Fatal("reporter must curl -k: mkcert is localhost, WSL has no CA")
	}
	if !strings.Contains(hookScriptTmpl, "https://localhost:") {
		t.Fatal("reporter must rewrite 127.0.0.1 → localhost for the cert SAN")
	}
}

func TestInterceptWrappersReportRuntimeLifecycle(t *testing.T) {
	if _, err := exec.LookPath("sh"); err != nil {
		t.Skip("sh not on PATH")
	}
	root := t.TempDir()
	dataDir := filepath.Join(root, "data")
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		t.Fatal(err)
	}
	hookLog := filepath.Join(root, "hooks.log")
	realLog := filepath.Join(root, "real.log")
	for _, cli := range []string{"claude-code", "codex", "grok", "hermes", "pi"} {
		if err := installIntercept(dataDir, cli); err != nil {
			t.Fatalf("install %s: %v", cli, err)
		}
	}
	hook := hookScriptPath(dataDir)
	if err := writeExecutable(hook, "#!/bin/sh\nprintf '%s|%s|%s|%s\\n' \"$1\" \"$2\" \"$3\" \"$4\" >> \"$PICODE_TEST_HOOK_LOG\"\n"); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"claude", "codex", "grok", "hermes", "pi"} {
		path := filepath.Join(root, name)
		body := "#!/bin/sh\nprintf '%s|%s\\n' \"" + name + "\" \"$*\" >> \"$PICODE_TEST_REAL_LOG\"\n"
		if name == "grok" {
			// Still executable, but its interpreter is absent: the wrapper's
			// direct launch fails and must still report runtime-end.
			body = "#!/no/such/picode-interpreter\n"
		}
		if name == "codex" {
			// The wrapper probes --help before the real invocation. Returning
			// no marker selects the normal hook override branch.
			body += "exit 0\n"
		}
		if err := writeExecutable(path, body); err != nil {
			t.Fatal(err)
		}
		if out, err := exec.Command("sh", "-n", wrapperPath(dataDir, name)).CombinedOutput(); err != nil {
			t.Fatalf("%s wrapper syntax: %v: %s", name, err, out)
		}
	}

	cases := []struct {
		cli     string
		args    []string
		wantErr bool
	}{
		{cli: "claude-code", args: []string{"hello"}},
		{cli: "codex", args: []string{"hello"}},
		{cli: "grok", args: []string{"hello"}, wantErr: true},
		{cli: "hermes", args: []string{"--tui"}},
		{cli: "pi", args: []string{"hello"}},
	}
	for _, tc := range cases {
		t.Run(tc.cli, func(t *testing.T) {
			name := map[string]string{"claude-code": "claude", "codex": "codex", "grok": "grok", "hermes": "hermes", "pi": "pi"}[tc.cli]
			cmd := exec.Command(wrapperPath(dataDir, name), tc.args...)
			cmd.Env = append(os.Environ(),
				"PATH="+interceptBinDir(dataDir)+string(os.PathListSeparator)+root+string(os.PathListSeparator)+"/usr/bin:/bin",
				"PICODE_TERM_ID=terminal-test",
				"PICODE_TEST_HOOK_LOG="+hookLog,
				"PICODE_TEST_REAL_LOG="+realLog,
			)
			if out, err := cmd.CombinedOutput(); (err != nil) != tc.wantErr {
				t.Fatalf("wrapper error = %v, wantErr=%v: %s", err, tc.wantErr, out)
			}
		})
	}
	got, err := os.ReadFile(hookLog)
	if err != nil {
		t.Fatal(err)
	}
	for _, cli := range []string{"claude", "codex", "grok", "hermes", "pi"} {
		if !strings.Contains(string(got), "runtime-start|"+cli+"|") || !strings.Contains(string(got), "runtime-end|"+cli+"|") {
			t.Fatalf("%s lifecycle = %q", cli, got)
		}
	}
	if real, err := os.ReadFile(realLog); err != nil {
		t.Fatal(err)
	} else if !strings.Contains(string(real), "pi|-e "+piTerminalStateExtensionFile(dataDir)+" hello") {
		t.Fatalf("real argv did not preserve Pi injection: %q", real)
	}
}

func TestClaudeSetWiringRefusesEnable(t *testing.T) {
	_, err := claudeSetWiring("/nope", "", true)
	if err == nil || !strings.Contains(err.Error(), "refusing") {
		t.Fatalf("enable must refuse writing user settings: %v", err)
	}
}
