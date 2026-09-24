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
	"time"

	"github.com/cfpperche/picode/internal/clilaunch"
	"github.com/cfpperche/picode/internal/pimission"
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
	if strings.Contains(string(rawSettings), "TaskCompleted") || strings.Contains(string(rawSettings), "SubagentStop") || !strings.Contains(string(rawSettings), "Stop") || !strings.Contains(string(rawSettings), " auto claude-code") {
		t.Fatalf("settings should map parent Stop via auto:\n%s", rawSettings)
	}
	for _, want := range []string{"PostToolUse", "PostToolUseFailure", "PreCompact", "PostCompact"} {
		if !strings.Contains(string(rawSettings), want) {
			t.Fatalf("settings must resume an approved turn via tool hooks, missing %s:\n%s", want, rawSettings)
		}
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

	for _, cli := range clilaunch.Catalog() {
		if res := postJSON(t, ts, "/api/terminals/wiring/"+cli.ID+"/disable", map[string]any{}); res.StatusCode != http.StatusOK {
			t.Fatalf("disable %s = %d", cli.ID, res.StatusCode)
		}
	}
	if _, err := os.Stat(wrap); !os.IsNotExist(err) {
		t.Fatal("wrapper should be gone after disable")
	}
	// Two surface policies default on: the tmux guard (ADR-0138) and the
	// browser hand-off (ADR-0180). With the CLI wrappers gone they are what
	// remains, and they keep the intercept bin dir on PATH.
	if interceptSessionPath(dataDir) == "" {
		t.Fatal("the default-on wrappers must keep the intercept bin dir on PATH")
	}
	entries, err := os.ReadDir(interceptBinDir(dataDir))
	if err != nil {
		t.Fatal(err)
	}
	names := []string{}
	for _, e := range entries {
		names = append(names, e.Name())
	}
	want := strings.Join([]string{"picode-open", "tmux", "wslview", "xdg-open"}, ",")
	if strings.Join(names, ",") != want {
		t.Fatalf("intercept dir = %v, want %s", names, want)
	}
	// Disabling the hand-off leaves only the guard.
	if res := postJSON(t, ts, "/api/terminals/wiring/open-url/disable", map[string]any{}); res.StatusCode != http.StatusOK {
		t.Fatalf("disable open-url = %d", res.StatusCode)
	}
	entries, err = os.ReadDir(interceptBinDir(dataDir))
	if err != nil || len(entries) != 1 || entries[0].Name() != "tmux" {
		t.Fatalf("intercept dir = %v (err %v), want only the tmux guard", entries, err)
	}
	// Disabling the guard as well restores the no-interception state.
	if res := postJSON(t, ts, "/api/terminals/wiring/tmux-guard/disable", map[string]any{}); res.StatusCode != http.StatusOK {
		t.Fatalf("disable tmux-guard = %d", res.StatusCode)
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
	for _, want := range []string{"hooks.SessionStart", "hooks.UserPromptSubmit", "hooks.PreCompact", "hooks.PostCompact", "hooks.PermissionRequest", "hooks.PostToolUse", "hooks.Interrupt", "hooks.state=", "notify="} {
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
	if !strings.Contains(string(body), "PICODE_NATIVE_HOOK=") {
		t.Fatalf("grok wrapper:\n%s", body)
	}
	hookJSON := filepath.Join(nativeAssetsDir(dataDir), "grok.json")
	raw, err := os.ReadFile(hookJSON)
	if err != nil {
		t.Fatalf("grok hooks missing: %v", err)
	}
	for _, want := range []string{"SessionStart", "UserPromptSubmit", "Notification", "PreCompact", "PostCompact", "PostToolUse", "PostToolUseFailure", "timeout\":10"} {
		if !strings.Contains(string(raw), want) {
			t.Fatalf("grok hooks missing %s: %s", want, raw)
		}
	}
	// Grok has no PermissionRequest event (1.0.30): a group it skips silently
	// must not be shipped as if it reported a permission prompt.
	if strings.Contains(string(raw), "PermissionRequest") {
		t.Fatalf("grok hooks carry an event Grok does not dispatch: %s", raw)
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
	if err := writeExecutable(inner, "#!/bin/sh\nfor arg in \"$@\"; do if [ \"$arg\" = path ]; then printf '%s\\n' \"$HOME/.hermes/config.yaml\"; exit; fi; if [ \"$arg\" = plugins ]; then exit 0; fi; done\nprintf 'PYTHONPATH=%s\\nHOOK=%s\\nARGS=%s\\n' \"$PYTHONPATH\" \"$PICODE_NATIVE_HOOK\" \"$*\" > \"$PICODE_TEST_INNER\"\n"); err != nil {
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
		"HERMES_HOME="+filepath.Join(root, "hermes-home"),
		"GROK_HOME="+filepath.Join(root, "grok-home"),
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

func TestHermesSubcommandSkipsNativeIntegration(t *testing.T) {
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
		{name: "bare tui", args: []string{"--tui"}, inject: true, wantArg: "--tui"},
		{name: "resume", args: []string{"--resume", "sess-1"}, inject: true, wantArg: "--resume sess-1"},
		{name: "chat", args: []string{"chat"}, inject: true, wantArg: "chat"},
		{name: "prompt args", args: []string{"two words", "$literal"}, inject: true, wantArg: "two words $literal"},
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
			hasPy := strings.Contains(s, "HOOK="+hookScriptPath(dataDir))
			if hasPy != tc.inject {
				t.Fatalf("inject=%v PYTHONPATH:\n%s", tc.inject, s)
			}
		})
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
	// Activity, Ask and native Missions each have one PiCode-owned extension.
	if !strings.Contains(string(wrapperBody), quotedCLIArgs([]string{"-e", extension, "-e", piReplyExtensionFile(dataDir), "-e", pimission.Path(dataDir)})+" \"$@\"") {
		t.Fatalf("pi wrapper does not prepend the state, receiver and Missions extensions and preserve argv:\n%s", wrapperBody)
	}
	if _, err := os.Stat(piReplyExtensionFile(dataDir)); err != nil {
		t.Fatalf("the receiver extension the wrapper names does not exist: %v", err)
	}
	if _, err := os.Stat(pimission.Path(dataDir)); err != nil {
		t.Fatalf("native mission extension missing: %v", err)
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
				wantArgs = append([]string{"-e", extension, "-e", piReplyExtensionFile(dataDir), "-e", pimission.Path(dataDir)}, wantArgs...)
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
await handler({}, { mode, isIdle: () => idle === "true", sessionManager: { getSessionId: () => "native-pi", getSessionFile: () => "/fixture/pi.jsonl" } });
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
		{name: "compaction starts", event: "session_before_compact", mode: "tui", hasTerm: true, want: "compacting|pi\n"},
		{name: "manual compaction ends idle", event: "session_compact", mode: "tui", hasTerm: true, idle: true, want: "idle|pi\n"},
		{name: "auto compaction resumes work", event: "session_compact", mode: "tui", hasTerm: true, want: "working|pi\n"},
		{name: "failed compaction returns idle", event: "session_compact_failed", mode: "tui", hasTerm: true, idle: true, want: "idle|pi\n"},
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
	run := func(in, cli string) string {
		t.Helper()
		cmd := exec.Command("python3", filepath.Join(dir, "picode-hook-map.py"))
		cmd.Stdin = strings.NewReader(in)
		if cli != "" {
			cmd.Env = append(os.Environ(), "PICODE_HOOK_CLI="+cli)
		}
		out, err := cmd.Output()
		if err != nil {
			t.Fatalf("map %q: %v", in, err)
		}
		return string(out)
	}
	cases := []struct {
		in, cli, want string
	}{
		{`{"hook_event_name":"UserPromptSubmit"}`, "", "working\n"},
		{`{"hook_event_name":"SessionStart"}`, "", "idle\n"},
		{`{"hook_event_name":"SessionStart","source":"compact"}`, "", ""},
		{`{"hook_event_name":"PreCompact","trigger":"manual"}`, "codex", "compacting\n"},
		{`{"hook_event_name":"PostCompact","trigger":"manual"}`, "codex", "idle\n"},
		{`{"hook_event_name":"PreCompact","trigger":"auto"}`, "claude-code", "compacting\n"},
		{`{"hook_event_name":"PostCompact","trigger":"auto"}`, "claude-code", "working\n"},
		{`{"hook_event_name":"PreCompact","trigger":"manual"}`, "grok", "compacting\n"},
		{`{"hook_event_name":"Stop"}`, "", "idle\n"},
		{`{"hook_event_name":"TaskCompleted"}`, "", ""},
		{`{"type":"agent-turn-complete"}`, "", "idle\n"},
		{`{"hook_event_name":"Interrupt"}`, "", "idle\n"},
		{`{"hook_event_name":"PermissionRequest"}`, "", "needs-you\n"},
		{`{"hook_event_name":"Notification","notification_type":"permission_prompt"}`, "", "needs-you\n"},
		{`{"hook_event_name":"pre_llm_call"}`, "", "working\n"},
		{`{"hook_event_name":"post_approval_response"}`, "", "working\n"},
		{`{"hook_event_name":"on_session_start"}`, "", "idle\n"},
		{`{"hook_event_name":"on_session_end"}`, "", "idle\n"},
		{`{"hook_event_name":"post_llm_call"}`, "", "idle\n"},
		{`{"hook_event_name":"subagent_stop"}`, "", ""},
		{`{"hook_event_name":"on_session_finalize"}`, "", "working\n"},
		{`{"hook_event_name":"on_session_reset"}`, "", "idle\n"},
		{`{"hook_event_name":"pre_approval_request"}`, "", "needs-you\n"},
		{`{"hook_event_name":"pre_tool_call"}`, "", ""},
		// Tool lifecycle resumes a turn after an approved permission prompt:
		// the CLI has no "permission resolved" event, so PostToolUse (or a
		// failed dispatch, where the model keeps going) is the working signal.
		{`{"hook_event_name":"PostToolUse"}`, "claude-code", "working\n"},
		{`{"hook_event_name":"PostToolUseFailure"}`, "claude-code", "working\n"},
		{`{"hook_event_name":"post_tool_use"}`, "claude-code", "working\n"},
		{`{"hook_event_name":"PostToolUse"}`, "codex", "working\n"},
		{`{"hook_event_name":"PermissionRequest"}`, "grok", "needs-you\n"},
		{`{"hook_event_name":"PostToolUse"}`, "grok", "working\n"},
		{`{"hook_event_name":"Notification","notification_type":"permission_prompt"}`, "grok", "needs-you\n"},
		{`{"hook_event_name":"Notification","notification_type":"idle_prompt"}`, "grok", "idle\n"},
		// Grok's other notifications carry no attention meaning; a
		// task_complete must not turn the row into a false "Needs you".
		{`{"hook_event_name":"Notification","notification_type":"task_complete"}`, "grok", ""},
		// Antigravity title/statusline payloads (Fatia 5): the CLI reports
		// its own lifecycle as agent_state. Unknown states claim nothing.
		{`{"agent_state":"idle","conversation_id":"c1"}`, "agy", "idle\n"},
		{`{"agent_state":"working","conversation_id":"c1"}`, "agy", "working\n"},
		{`{"agent_state":"thinking","conversation_id":"c1"}`, "agy", "working\n"},
		{`{"agent_state":"tool_use","conversation_id":"c1"}`, "agy", "working\n"},
		{`{"agent_state":"initializing","conversation_id":"c1"}`, "agy", "working\n"},
		{`{"agent_state":"exploding","conversation_id":"c1"}`, "agy", ""},
		{`{"conversation_id":"c1"}`, "agy", ""},
	}
	for _, tc := range cases {
		if got := run(tc.in, tc.cli); got != tc.want {
			t.Fatalf("[%s] %s = %q, want %q", tc.cli, tc.in, got, tc.want)
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

	// Captured from Codex 0.153.0 app-server hooks/list with the exact
	// timeout PiCode injects: the tool event carries the same trust formula.
	spec = codexHookSpec{key: "post_tool_use", timeoutSec: 5}
	got = codexHookHash(spec, "/tmp/codex-hook-probe/hook.sh")
	want = "sha256:c77160dccd8204d3e78a3c200c47b4d963c1484225618c61867d807d0fbda249"
	if got != want {
		t.Fatalf("tool hook hash = %q, want Codex fingerprint %q", got, want)
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
	for _, cli := range []string{"claude-code", "codex", "grok", "hermes", "opencode", "pi"} {
		if err := installIntercept(dataDir, cli); err != nil {
			t.Fatalf("install %s: %v", cli, err)
		}
	}
	hook := hookScriptPath(dataDir)
	if err := writeExecutable(hook, "#!/bin/sh\nprintf '%s|%s|%s|%s\\n' \"$1\" \"$2\" \"$3\" \"$4\" >> \"$PICODE_TEST_HOOK_LOG\"\n"); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"claude", "codex", "grok", "hermes", "opencode", "pi"} {
		path := filepath.Join(root, name)
		body := "#!/bin/sh\nif [ \"$1\" = config ] && [ \"$2\" = path ]; then printf '%s\\n' \"$HOME/.hermes/config.yaml\"; exit; fi\nif [ \"$1\" = plugins ]; then exit 0; fi\nprintf '%s|%s\\n' \"" + name + "\" \"$*\" >> \"$PICODE_TEST_REAL_LOG\"\n"
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
		{cli: "opencode", args: []string{"--session", "ses_test"}},
		{cli: "pi", args: []string{"hello"}},
	}
	for _, tc := range cases {
		t.Run(tc.cli, func(t *testing.T) {
			name := map[string]string{"claude-code": "claude", "codex": "codex", "grok": "grok", "hermes": "hermes", "opencode": "opencode", "pi": "pi"}[tc.cli]
			cmd := exec.Command(wrapperPath(dataDir, name), tc.args...)
			cmd.Env = append(os.Environ(),
				"PATH="+interceptBinDir(dataDir)+string(os.PathListSeparator)+root+string(os.PathListSeparator)+"/usr/bin:/bin",
				"PICODE_TERM_ID=terminal-test",
				"HERMES_HOME="+filepath.Join(root, "hermes-home"),
				"GROK_HOME="+filepath.Join(root, "grok-home"),
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
	for _, cli := range []string{"claude", "codex", "grok", "hermes", "opencode", "pi"} {
		if !strings.Contains(string(got), "runtime-start|"+cli+"|") || !strings.Contains(string(got), "runtime-end|"+cli+"|") {
			t.Fatalf("%s lifecycle = %q", cli, got)
		}
	}
	if real, err := os.ReadFile(realLog); err != nil {
		t.Fatal(err)
	} else if !strings.Contains(string(real), "pi|-e "+piTerminalStateExtensionFile(dataDir)+" -e "+piReplyExtensionFile(dataDir)+" -e "+pimission.Path(dataDir)+" hello") {
		t.Fatalf("real argv did not preserve Pi injection: %q", real)
	}
}

func TestInterceptOpencodeConfigPlugin(t *testing.T) {
	ts, dataDir := wiringTestServer(t)
	home, _ := os.UserHomeDir()
	userDir := filepath.Join(home, ".config", "opencode")
	if err := os.MkdirAll(userDir, 0o755); err != nil {
		t.Fatal(err)
	}
	userJSONC := filepath.Join(userDir, "opencode.jsonc")
	original := []byte("{\n  // user comment\n  \"$schema\": \"https://opencode.ai/config.json\"\n}\n")
	if err := os.WriteFile(userJSONC, original, 0o644); err != nil {
		t.Fatal(err)
	}
	past := time.Unix(1_700_000_000, 0)
	if err := os.Chtimes(userJSONC, past, past); err != nil {
		t.Fatal(err)
	}
	before, err := os.Stat(userJSONC)
	if err != nil {
		t.Fatal(err)
	}

	if res := postJSON(t, ts, "/api/terminals/wiring/opencode/enable", map[string]any{}); res.StatusCode != http.StatusOK {
		t.Fatalf("opencode enable = %d", res.StatusCode)
	}
	body, err := os.ReadFile(wrapperPath(dataDir, "opencode"))
	if err != nil {
		t.Fatalf("opencode wrapper missing: %v", err)
	}
	got := string(body)
	if !strings.Contains(got, "name=opencode") || !strings.Contains(got, "runtime-start") {
		t.Fatalf("opencode wrapper missing presence lease:\n%s", got)
	}
	if !strings.Contains(got, "OPENCODE_CONFIG=") || !strings.Contains(got, opencodeConfigFile(dataDir)) {
		t.Fatalf("opencode wrapper missing OPENCODE_CONFIG:\n%s", got)
	}
	if !strings.Contains(got, "PICODE_OPENCODE_HOOK=") {
		t.Fatalf("opencode wrapper missing hook env:\n%s", got)
	}
	if !strings.Contains(got, "Starting OpenCode...") {
		t.Fatalf("opencode wrapper missing start banner:\n%s", got)
	}
	if strings.Contains(got, "XDG_DATA_HOME=") || strings.Contains(got, "OPENCODE_CONFIG_DIR=") || strings.Contains(got, "OPENCODE_CONFIG_CONTENT=") || strings.Contains(got, "--pure") {
		t.Fatalf("opencode wrapper must not overlay data dir, steal config-dir, or disable plugins:\n%s", got)
	}
	after, err := os.Stat(userJSONC)
	if err != nil {
		t.Fatal(err)
	}
	if after.ModTime() != before.ModTime() {
		t.Fatalf("enable must not touch user jsonc mtime: before %v after %v", before.ModTime(), after.ModTime())
	}
	raw, err := os.ReadFile(userJSONC)
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != string(original) {
		t.Fatalf("enable must not rewrite user jsonc: %s", raw)
	}
	if _, err := os.Stat(filepath.Join(userDir, "plugins")); !os.IsNotExist(err) {
		t.Fatalf("enable must not create ~/.config/opencode/plugins: %v", err)
	}
	plugin, err := os.ReadFile(opencodePluginFile(dataDir))
	if err != nil {
		t.Fatalf("plugin missing: %v", err)
	}
	src := string(plugin)
	if !strings.Contains(src, "session.status") || !strings.Contains(src, "permission.asked") || !strings.Contains(src, "export default") {
		t.Fatalf("plugin: %s", src)
	}
	if strings.Contains(src, "console.log") {
		t.Fatal("plugin must not write to the TUI via console.log")
	}
	if strings.Count(src, "export ") != 1 {
		t.Fatalf("plugin must export only the default function:\n%s", src)
	}
	cfg, err := os.ReadFile(opencodeConfigFile(dataDir))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(cfg), "./picode-activity.js") || !strings.Contains(string(cfg), "$schema") {
		t.Fatalf("session config: %s", cfg)
	}
	var page struct {
		Clis []wiringRow `json:"clis"`
	}
	res := do(t, ts.Client(), mustGet(t, ts.URL+"/api/terminals/wiring"))
	_ = json.NewDecoder(res.Body).Decode(&page)
	var row *wiringRow
	for i := range page.Clis {
		if page.Clis[i].ID == "opencode" {
			row = &page.Clis[i]
		}
	}
	if row == nil || !row.Wired || !strings.Contains(row.Note, "plugin") {
		t.Fatalf("opencode wiring = %+v", page.Clis)
	}
	if strings.Contains(row.Note, "Presence lease only") {
		t.Fatalf("stale OpenCode note: %q", row.Note)
	}
}

func TestOpencodeMaintenanceSkipsPresenceLease(t *testing.T) {
	if _, err := exec.LookPath("sh"); err != nil {
		t.Skip("sh not on PATH")
	}
	root := t.TempDir()
	dataDir := filepath.Join(root, "data")
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := writeOpencodeIntercept(dataDir, hookScriptPath(dataDir)); err != nil {
		t.Fatal(err)
	}
	hookLog := filepath.Join(root, "hooks.log")
	realLog := filepath.Join(root, "real.log")
	if err := writeExecutable(hookScriptPath(dataDir), "#!/bin/sh\nprintf '%s|%s\\n' \"$1\" \"$2\" >> \"$PICODE_TEST_HOOK_LOG\"\n"); err != nil {
		t.Fatal(err)
	}
	if err := writeExecutable(filepath.Join(root, "opencode"), "#!/bin/sh\nprintf 'OPENCODE_CONFIG=%s|ARGS=%s\\n' \"${OPENCODE_CONFIG-}\" \"$*\" >> \"$PICODE_TEST_REAL_LOG\"\n"); err != nil {
		t.Fatal(err)
	}
	if out, err := exec.Command("sh", "-n", wrapperPath(dataDir, "opencode")).CombinedOutput(); err != nil {
		t.Fatalf("wrapper syntax: %v: %s", err, out)
	}
	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command(wrapperPath(dataDir, "opencode"), args...)
		cmd.Env = append(os.Environ(),
			"PATH="+interceptBinDir(dataDir)+string(os.PathListSeparator)+root+string(os.PathListSeparator)+"/usr/bin:/bin",
			"PICODE_TERM_ID=terminal-test",
			"HERMES_HOME="+filepath.Join(root, "hermes-home"),
			"GROK_HOME="+filepath.Join(root, "grok-home"),
			"PICODE_TEST_HOOK_LOG="+hookLog,
			"PICODE_TEST_REAL_LOG="+realLog,
		)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("wrapper %v: %v: %s", args, err, out)
		}
	}
	run("session", "list")
	run("--log-level", "DEBUG", "auth", "list")
	got, err := os.ReadFile(hookLog)
	if err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	if strings.Contains(string(got), "runtime-start") {
		t.Fatalf("maintenance must not take a presence lease:\n%s", got)
	}
	real, err := os.ReadFile(realLog)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(real), "session list") || !strings.Contains(string(real), "auth list") {
		t.Fatalf("real argv: %q", real)
	}
	if strings.Contains(string(real), opencodeConfigFile(dataDir)) {
		t.Fatalf("maintenance must not set OPENCODE_CONFIG:\n%s", real)
	}
	run("/tmp/some-project")
	run("--session", "ses_x")
	got, err = os.ReadFile(hookLog)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(got), "runtime-start|opencode") {
		t.Fatalf("TUI project path and --session must take a presence lease:\n%s", got)
	}
	real, err = os.ReadFile(realLog)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(real), opencodeConfigFile(dataDir)) {
		t.Fatalf("TUI must set OPENCODE_CONFIG:\n%s", real)
	}
}

func TestOpencodePluginListedByDebugInfo(t *testing.T) {
	if _, err := exec.LookPath("opencode"); err != nil {
		t.Skip("opencode not on PATH")
	}
	root := t.TempDir()
	dataDir := filepath.Join(root, "data")
	if err := writeOpencodeIntercept(dataDir, filepath.Join(dataDir, "picode-hook")); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("opencode", "debug", "info")
	cmd.Dir = root
	cmd.Env = append(os.Environ(),
		"HOME="+root,
		"XDG_CONFIG_HOME="+filepath.Join(root, "config"),
		"XDG_DATA_HOME="+filepath.Join(root, "xdg"),
		"OPENCODE_CONFIG="+opencodeConfigFile(dataDir),
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("opencode debug info: %v: %s", err, out)
	}
	if !strings.Contains(string(out), "picode-activity.js") {
		t.Fatalf("plugin not loaded:\n%s", out)
	}
}

func TestOpencodeActivityMap(t *testing.T) {
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("node not on PATH")
	}
	dir := t.TempDir()
	script := filepath.Join(dir, "map.js")
	body := opencodeMapEventJS + `
const rows = [
  ["session.status", "busy", "working"],
  ["session.status", "retry", "working"],
  ["session.status", "idle", "idle"],
  ["session.status", "other", ""],
  ["session.idle", "", "idle"],
  ["permission.asked", "", "needs-you"],
  ["permission.v2.asked", "", "needs-you"],
  ["question.asked", "", "needs-you"],
  ["question.v2.asked", "", "needs-you"],
  ["permission.replied", "", "working"],
  ["permission.v2.replied", "", "working"],
  ["question.replied", "", "working"],
  ["question.v2.replied", "", "working"],
  ["question.rejected", "", "idle"],
  ["question.v2.rejected", "", "idle"],
  ["tool.execute.before", "", ""],
  ["message.updated", "", ""],
]
for (const [type, status, want] of rows) {
  const got = mapEvent(type, status)
  if (got !== want) {
    console.error(type + " " + status + " = " + JSON.stringify(got) + " want " + JSON.stringify(want))
    process.exit(1)
  }
}
`
	if err := os.WriteFile(script, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	out, err := exec.Command("node", script).CombinedOutput()
	if err != nil {
		t.Fatalf("map table: %v: %s", err, out)
	}
}

func TestClaudeSetWiringRefusesEnable(t *testing.T) {
	_, err := claudeSetWiring("/nope", "", true)
	if err == nil || !strings.Contains(err.Error(), "refusing") {
		t.Fatalf("enable must refuse writing user settings: %v", err)
	}
}

func TestHermesProfileKeepsNativeRuntimeLease(t *testing.T) {
	root := t.TempDir()
	t.Setenv("HOME", root)
	wrapper, hookLog, envLog := writeHermesInnerProbe(t, filepath.Join(root, "data"), root)
	runHermesWrapper(t, wrapper, root, hookLog, envLog, "-p", "selected", "chat")
	raw, err := os.ReadFile(hookLog)
	if err != nil || !strings.Contains(string(raw), "runtime-start|hermes") {
		t.Fatalf("profile suppressed TUI lease: %s %v", raw, err)
	}
}

func TestCodexSubcommandsKeepHookOverrides(t *testing.T) {
	root := t.TempDir()
	real := filepath.Join(root, "codex-probe")
	if err := writeExecutable(real, "#!/bin/sh\nprintf '%s\\000' \"$@\"\n"); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"hello"}, {"resume", "session id"}, {"fork", "session id"}} {
		body := "real=" + shellQuote(real) + "\n" + codexInvoke([]string{"-c", "hooks.Stop=[]", "-c", "hooks.state={}"})
		cmd := exec.Command("sh", append([]string{"-c", body, "probe"}, args...)...)
		raw, err := cmd.Output()
		if err != nil {
			t.Fatal(err)
		}
		got := strings.Split(strings.TrimSuffix(string(raw), "\x00"), "\x00")
		want := []string{"-c", "hooks.Stop=[]", "-c", "hooks.state={}"}
		if args[0] == "resume" || args[0] == "fork" {
			want = append(append([]string{}, args...), want...)
		} else {
			want = append(want, args...)
		}
		if strings.Join(got, "\x00") != strings.Join(want, "\x00") {
			t.Fatalf("%q => %q want %q", args, got, want)
		}
	}
}

func TestCodexHookContextAndLegacySelection(t *testing.T) {
	if _, err := exec.LookPath("python3"); err != nil {
		t.Skip("python3 unavailable")
	}
	root := t.TempDir()
	hook, err := ensureHookScript(root)
	if err != nil {
		t.Fatal(err)
	}
	writeExecutable(filepath.Join(root, "curl"), "#!/bin/sh\nexit 0\n")
	for _, tc := range []struct {
		event   string
		context bool
	}{
		{`{"hook_event_name":"SessionStart","session_id":"root"}`, true},
		{`{"hook_event_name":"Stop","session_id":"root"}`, false},
		{`{"hook_event_name":"SessionStart","session_id":"child","parent_session_id":"root"}`, false},
	} {
		cmd := exec.Command(hook, "auto", "codex", tc.event)
		cmd.Env = append(os.Environ(), "PATH="+root+string(os.PathListSeparator)+os.Getenv("PATH"), "PICODE_TERM_ID=fixture", "PICODE_TERM_URL=http://localhost:1", "PICODE_CODEX_HOOKS=1")
		out, err := cmd.Output()
		if err != nil {
			t.Fatal(err)
		}
		if !tc.context {
			if len(out) != 0 {
				t.Fatalf("unexpected hook context: %s", out)
			}
			continue
		}
		var got struct {
			Output struct {
				Event   string `json:"hookEventName"`
				Context string `json:"additionalContext"`
			} `json:"hookSpecificOutput"`
		}
		if json.Unmarshal(out, &got) != nil || got.Output.Event != "SessionStart" || !strings.Contains(got.Output.Context, "picode messages --help") {
			t.Fatalf("missing native discovery: %s", out)
		}
	}
	for _, modern := range []bool{false, true} {
		probe := "#!/bin/sh\nif [ \"$1\" = --help ]; then\n"
		if modern {
			probe += "printf '%s\\n' --dangerously-bypass-hook-trust\n"
		}
		probe += "exit 0\nfi\nprintf '%s' \"${PICODE_CODEX_HOOKS:-legacy}\"\n"
		writeExecutable(filepath.Join(root, "codex"), probe)
		writeExecutable(hook, "#!/bin/sh\nexit 0\n")
		if err := writeCodexIntercept(root, hook); err != nil {
			t.Fatal(err)
		}
		cmd := exec.Command(wrapperPath(root, "codex"))
		cmd.Env = append(os.Environ(), "PATH="+interceptBinDir(root)+string(os.PathListSeparator)+root+string(os.PathListSeparator)+"/usr/bin:/bin", "PICODE_TERM_ID=fixture", "PICODE_CODEX_HOOKS=1")
		out, err := cmd.Output()
		want := "legacy"
		if modern {
			want = "1"
		}
		if err != nil || string(out) != want {
			t.Fatalf("modern=%v: %q %v", modern, out, err)
		}
	}
}
