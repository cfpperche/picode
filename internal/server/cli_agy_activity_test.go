package server

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cfpperche/picode/internal/clilaunch"
	"github.com/cfpperche/picode/internal/store"
	"github.com/cfpperche/picode/internal/tmux"
)

// The mapper in REPORT mode: the stale IDE transcript path must not ride
// along, the conversation id must.
func TestHookMapAgyReport(t *testing.T) {
	if _, err := exec.LookPath("python3"); err != nil {
		t.Skip("python3 not on PATH")
	}
	dir := t.TempDir()
	if _, err := ensureHookScript(dir); err != nil {
		t.Fatal(err)
	}
	in := `{"agent_state":"working","conversation_id":"c9","session_id":"c9","transcript_path":"/home/goat/.gemini/antigravity/brain/c9/x.jsonl","cwd":"/w"}`
	// Stdin comes from a file, not a pipe: twice under sharded load the
	// pipe-fed run exited 0 with empty output (unreproduced in isolation);
	// the pipe plumbing itself stays covered by TestAgyTitleReporterFlow.
	inFile := filepath.Join(dir, "in.json")
	if err := os.WriteFile(inFile, []byte(in), 0o600); err != nil {
		t.Fatal(err)
	}
	fh, err := os.Open(inFile)
	if err != nil {
		t.Fatal(err)
	}
	defer fh.Close()
	cmd := exec.Command("python3", filepath.Join(dir, "picode-hook-map.py"))
	cmd.Stdin = fh
	cmd.Env = append(os.Environ(), "PICODE_HOOK_CLI=agy", "PICODE_HOOK_REPORT=1")
	out, err := cmd.Output()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			t.Fatalf("map exit %d stderr: %s", ee.ExitCode(), ee.Stderr)
		}
		t.Fatalf("map: %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(out, &got); err != nil {
		// Twice under sharded load this came back empty with exit 0; never
		// in isolation. Capture everything before failing so the next
		// occurrence diagnoses itself instead of shrugging.
		again, _ := exec.Command("python3", filepath.Join(dir, "picode-hook-map.py")).Output()
		st, _ := os.Stat(filepath.Join(dir, "picode-hook-map.py"))
		t.Fatalf("report is not JSON: %q rerun=%q size=%d err=%v", out, again, st.Size(), err)
	}
	if got["state"] != "working" || got["cli"] != "agy" || got["sessionId"] != "c9" {
		t.Fatalf("report = %s", out)
	}
	if p, _ := got["sessionPath"].(string); p != "" {
		t.Fatalf("stale transcript path leaks: %s", out)
	}
}

func agyTestHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	return home
}

func TestAgyTitleSettingsMerge(t *testing.T) {
	home := agyTestHome(t)
	settings := filepath.Join(home, ".gemini", "antigravity-cli", "settings.json")
	dataDir := t.TempDir()
	reporter := agyTitleReporterPath(dataDir)
	read := func() map[string]any {
		t.Helper()
		raw, err := os.ReadFile(settings)
		if err != nil {
			t.Fatal(err)
		}
		var doc map[string]any
		if err := json.Unmarshal(raw, &doc); err != nil {
			t.Fatal(err)
		}
		return doc
	}
	// Absent file: created with our block plus nothing else.
	if err := installAgyTitleReporter(dataDir); err != nil {
		t.Fatal(err)
	}
	if doc := read(); doc["title"].(map[string]any)["command"] != reporter {
		t.Fatalf("block = %v", doc["title"])
	}
	// Idempotent: second install keeps the file byte-identical.
	before, _ := os.ReadFile(settings)
	if err := installAgyTitleReporter(dataDir); err != nil {
		t.Fatal(err)
	}
	if after, _ := os.ReadFile(settings); string(after) != string(before) {
		t.Fatal("reinstall rewrote the file")
	}
	// Foreign block: refused, file untouched.
	doc := read()
	doc["title"] = map[string]any{"type": "command", "command": "/user/own.sh"}
	raw, _ := json.Marshal(doc)
	if err := os.WriteFile(settings, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := installAgyTitleReporter(dataDir); err == nil {
		t.Fatal("foreign title block replaced without error")
	}
	if after, _ := os.ReadFile(settings); string(after) != string(raw) {
		t.Fatal("refused install modified the file")
	}
	// Malformed file: refused, never clobbered.
	if err := os.WriteFile(settings, []byte("{nope"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := installAgyTitleReporter(dataDir); err == nil {
		t.Fatal("malformed settings accepted")
	}
	// Uninstall removes only ours.
	if err := os.Remove(settings); err != nil {
		t.Fatal(err)
	}
	if err := installAgyTitleReporter(dataDir); err != nil {
		t.Fatal(err)
	}
	removeAgyTitleReporter(dataDir)
	if _, err := os.Stat(settings); !os.IsNotExist(err) {
		t.Fatal("file we created was not removed")
	}
	// Uninstall keeps a foreign block and other keys.
	doc = map[string]any{"model": "m", "title": map[string]any{"type": "command", "command": "/user/own.sh"}}
	raw, _ = json.Marshal(doc)
	if err := os.WriteFile(settings, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	removeAgyTitleReporter(dataDir)
	if after := read(); after["title"].(map[string]any)["command"] != "/user/own.sh" || after["model"] != "m" {
		t.Fatalf("foreign settings changed: %v", after)
	}
}

// The reporter end to end: canned agent-state JSON in, hook POST plus the
// short title out. The hook server is a stub; picode-hook itself is real.
func TestAgyTitleReporterFlow(t *testing.T) {
	if _, err := exec.LookPath("python3"); err != nil {
		t.Skip("python3 not on PATH")
	}
	agyTestHome(t)
	dataDir := t.TempDir()
	if _, err := ensureHookScript(dataDir); err != nil {
		t.Fatal(err)
	}
	if err := installAgyTitleReporter(dataDir); err != nil {
		t.Fatal(err)
	}
	var received []byte
	hook := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buf, _ := io.ReadAll(r.Body)
		received = buf
		w.WriteHeader(200)
	}))
	defer hook.Close()
	in := `{"agent_state":"working","conversation_id":"c9","session_id":"c9","cwd":"/tmp/agyprobe","model":{"id":"m"}}`
	cmd := exec.Command(agyTitleReporterPath(dataDir))
	cmd.Stdin = strings.NewReader(in)
	cmd.Env = append(os.Environ(),
		"PICODE_TERM_ID=t-agy-1",
		"PICODE_TERM_URL="+hook.URL,
		"PATH="+os.Getenv("PATH"),
	)
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("reporter: %v", err)
	}
	var posted map[string]any
	if err := json.Unmarshal(received, &posted); err != nil {
		t.Fatalf("hook got no JSON: %q", received)
	}
	if posted["state"] != "working" || posted["cli"] != "agy" || posted["sessionId"] != "c9" {
		t.Fatalf("posted = %s", received)
	}
	if title := strings.TrimSpace(string(out)); !strings.Contains(title, "working") || !strings.Contains(title, "agyprobe") {
		t.Fatalf("title = %q", out)
	}
}

// prepare with integration on for agy must succeed without a PATH wrapper:
// before the no-wrapper guard this failed reading the missing wrapper file.
func TestAgyPrepareNeedsNoWrapper(t *testing.T) {
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
	if err := st.SetCLIConfig("agy", clilaunch.Config{Executable: exe, Integration: true}); err != nil {
		t.Fatal(err)
	}
	deps := Deps{Store: st, Tmux: tmux.NewWithSocket(filepath.Join(dir, "unused-sock")), DataDir: dir}
	v := &store.TerminalLaunch{TerminalID: "t-agy-prep", CLI: "agy", Overrides: clilaunch.Overrides{}}
	p, err := prepareCLITerminal(deps, t.TempDir(), v)
	if err != nil {
		t.Fatalf("prepare: %v", err)
	}
	defer p.discard()
}

func TestAgyReporterPrepared(t *testing.T) {
	home := agyTestHome(t)
	dataDir := t.TempDir()
	if agyReporterPrepared(dataDir) {
		t.Fatal("nothing installed, yet prepared")
	}
	if err := installAgyTitleReporter(dataDir); err != nil {
		t.Fatal(err)
	}
	// ensureHookScript is what installIntercept runs first; without the
	// hook the reporter is installed but the chain is incomplete.
	if agyReporterPrepared(dataDir) {
		t.Fatal("prepared without the hook chain")
	}
	if _, err := ensureHookScript(dataDir); err != nil {
		t.Fatal(err)
	}
	if !agyReporterPrepared(dataDir) {
		t.Fatal("installed reporter + hook reads not prepared")
	}
	_ = home
}
