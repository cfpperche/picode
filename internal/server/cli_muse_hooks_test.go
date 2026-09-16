package server

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cfpperche/picode/internal/clilaunch"
	"github.com/cfpperche/picode/internal/store"
	"github.com/cfpperche/picode/internal/tmux"
)

// The mapper needs no muse branch: hook_event_name payloads ride the
// generic path (measured live against R3233.1). This pins that contract —
// SessionStart reads idle the way Claude's does, work brackets working,
// approvals surface as needs-you.
func TestHookMapMuseReport(t *testing.T) {
	if _, err := exec.LookPath("python3"); err != nil {
		t.Skip("python3 not on PATH")
	}
	dir := t.TempDir()
	if _, err := ensureHookScript(dir); err != nil {
		t.Fatal(err)
	}
	cases := []struct{ in, want string }{
		{`{"hook_event_name":"SessionStart","session_id":"s1","cwd":"/w","source":"startup"}`, "idle"},
		{`{"hook_event_name":"UserPromptSubmit","session_id":"s1","turn_id":"t1","prompt":"hi"}`, "working"},
		{`{"hook_event_name":"PreToolUse","session_id":"s1","turn_id":"t1","tool_name":"read","tool_input":{},"tool_use_id":"c1"}`, "working"},
		{`{"hook_event_name":"PostToolUse","session_id":"s1","turn_id":"t1","tool_name":"read","tool_input":{},"tool_response":"ok","tool_use_id":"c1"}`, "working"},
		{`{"hook_event_name":"PermissionRequest","session_id":"s1","turn_id":"t1","tool_name":"write"}`, "needs-you"},
		{`{"hook_event_name":"Stop","session_id":"s1","turn_id":"t1","stop_hook_active":false}`, "idle"},
	}
	for _, c := range cases {
		// Hermetic against ambient terminal vars (see TestHookMapAgyReport).
		cmd := exec.Command("python3", filepath.Join(dir, "picode-hook-map.py"))
		cmd.Stdin = strings.NewReader(c.in)
		cmd.Env = append(os.Environ(), "PICODE_HOOK_CLI=muse", "PICODE_HOOK_REPORT=1", "PICODE_TUI_PID=", "PICODE_TUI_RUN_ID=")
		out, err := cmd.Output()
		if err != nil {
			t.Fatalf("map %s: %v", c.in, err)
		}
		var got map[string]any
		if err := json.Unmarshal(out, &got); err != nil {
			t.Fatalf("report is not JSON for %s: %q", c.in, out)
		}
		if got["state"] != c.want || got["cli"] != "muse" || got["sessionId"] != "s1" {
			t.Fatalf("report for %s = %s, want state %s", c.in, out, c.want)
		}
	}
}

func museTestHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", "")
	return home
}

func museTestSettings(t *testing.T) string {
	t.Helper()
	museTestHome(t)
	p, err := museSettingsPath()
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func readMuseDoc(t *testing.T, path string) map[string]any {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	return doc
}

func museEventCommands(t *testing.T, path, ev string) []string {
	t.Helper()
	doc := readMuseDoc(t, path)
	hooks := doc["hooks"].(map[string]any)
	var out []string
	for _, g := range hooks[ev].([]any) {
		for _, h := range g.(map[string]any)["hooks"].([]any) {
			out = append(out, h.(map[string]any)["command"].(string))
		}
	}
	return out
}

func TestMuseHooksSettingsMerge(t *testing.T) {
	path := museTestSettings(t)
	dataDir := t.TempDir()
	if _, err := ensureHookScript(dataDir); err != nil {
		t.Fatal(err)
	}
	// Absent file: created with schema_version and all six events.
	if err := installMuseHooks(dataDir); err != nil {
		t.Fatal(err)
	}
	doc := readMuseDoc(t, path)
	if doc["schema_version"] != float64(1) {
		t.Fatalf("schema_version = %v", doc["schema_version"])
	}
	hook := museHookPath(dataDir)
	for _, ev := range museHookEvents {
		cmds := museEventCommands(t, path, ev)
		if len(cmds) != 1 || cmds[0] != hook {
			t.Fatalf("%s commands = %v", ev, cmds)
		}
	}
	// Idempotent: a second install changes nothing.
	before, _ := os.ReadFile(path)
	if err := installMuseHooks(dataDir); err != nil {
		t.Fatal(err)
	}
	after, _ := os.ReadFile(path)
	if string(before) != string(after) {
		t.Fatal("second install rewrote the settings")
	}
	// Foreign entry on any event refuses, never replaces.
	doc["hooks"].(map[string]any)["Stop"] = []any{museHookGroup("/bin/false")}
	raw, _ := json.Marshal(doc)
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := installMuseHooks(dataDir); err == nil || !strings.Contains(err.Error(), "foreign") {
		t.Fatalf("foreign hook install = %v, want a refusal", err)
	}
	if cmds := museEventCommands(t, path, "Stop"); len(cmds) != 1 || cmds[0] != "/bin/false" {
		t.Fatalf("foreign Stop was touched: %v", cmds)
	}
	// Malformed file refuses, never clobbered.
	if err := os.WriteFile(path, []byte("{nope"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := installMuseHooks(dataDir); err == nil {
		t.Fatal("malformed settings merged without error")
	}
	if raw, _ := os.ReadFile(path); string(raw) != "{nope" {
		t.Fatal("malformed settings were clobbered")
	}
}

func TestMuseHooksUninstall(t *testing.T) {
	path := museTestSettings(t)
	dataDir := t.TempDir()
	if _, err := ensureHookScript(dataDir); err != nil {
		t.Fatal(err)
	}
	if err := installMuseHooks(dataDir); err != nil {
		t.Fatal(err)
	}
	// A foreign group on a shared event survives; ours is pruned.
	doc := readMuseDoc(t, path)
	ev := doc["hooks"].(map[string]any)["Stop"].([]any)
	doc["hooks"].(map[string]any)["Stop"] = append(ev, museHookGroup("/bin/false"))
	raw, _ := json.Marshal(doc)
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	removeMuseHooks(dataDir)
	doc = readMuseDoc(t, path)
	hooks := doc["hooks"].(map[string]any)
	if _, present := hooks["SessionStart"]; present {
		t.Fatal("SessionStart survived uninstall")
	}
	if cmds := museEventCommands(t, path, "Stop"); len(cmds) != 1 || cmds[0] != "/bin/false" {
		t.Fatalf("Stop after uninstall = %v", cmds)
	}
	if _, err := os.Stat(museHookPath(dataDir)); !os.IsNotExist(err) {
		t.Fatal("hook script survived uninstall")
	}
	// Clean removal of a file we created deletes it.
	if err := os.WriteFile(path, []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := installMuseHooks(dataDir); err != nil {
		t.Fatal(err)
	}
	removeMuseHooks(dataDir)
	if _, err := os.ReadFile(path); err == nil {
		// Only schema_version remains: valid file, kept.
		doc = map[string]any{}
		if raw, err := os.ReadFile(path); err == nil {
			_ = json.Unmarshal(raw, &doc)
		}
		if len(doc) != 1 || doc["schema_version"] != float64(1) {
			t.Fatalf("settings after full uninstall = %v", doc)
		}
	}
}

func TestMuseHooksPrepared(t *testing.T) {
	museTestHome(t)
	dataDir := t.TempDir()
	if museHooksPrepared(dataDir) {
		t.Fatal("nothing installed, yet prepared")
	}
	if _, err := ensureHookScript(dataDir); err != nil {
		t.Fatal(err)
	}
	if err := installMuseHooks(dataDir); err != nil {
		t.Fatal(err)
	}
	if !museHooksPrepared(dataDir) {
		t.Fatal("installed hooks read not prepared")
	}
	removeMuseHooks(dataDir)
	if museHooksPrepared(dataDir) {
		t.Fatal("prepared after uninstall")
	}
}

func TestMusePrepareNeedsNoWrapper(t *testing.T) {
	museTestHome(t)
	dir := t.TempDir()
	st, err := store.Open(filepath.Join(dir, "picode.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	exe := "/bin/true"
	if _, err := os.Stat(exe); err != nil {
		t.Skip("no /bin/true on this platform")
	}
	if err := st.SetCLIConfig("muse", clilaunch.Config{Executable: exe, Integration: true}); err != nil {
		t.Fatal(err)
	}
	deps := Deps{Store: st, Tmux: tmux.NewWithSocket(filepath.Join(dir, "unused-sock")), DataDir: dir}
	v := &store.TerminalLaunch{TerminalID: "t-muse-prep", CLI: "muse", Overrides: clilaunch.Overrides{}}
	p, err := prepareCLITerminal(deps, t.TempDir(), v)
	if err != nil {
		t.Fatalf("prepare: %v", err)
	}
	defer p.discard()
}
