package connectors

// Live vendor parity (ADR-0150): every guest driver's add/toggle/remove is
// exercised against the real vendor CLI in a sandbox HOME, and the vendor's
// own commands are asked to confirm what PiCode wrote. This is a
// measurement, not a CI gate: it shells out to the actual binaries, touches
// nothing outside the sandbox, and skips unless it is explicitly pointed at
// one.
//
// Run it like this (the sandbox HOME is what the vendor CLIs themselves
// resolve, so their config files land there and the real user's files are
// never read or written):
//
//	SB=$(mktemp -d)
//	env HOME=$SB XDG_CONFIG_HOME=$SB/.config \
//	    PICODE_LIVE_SANDBOX=$SB PICODE_CONNECTORS_LIVE=1 \
//	    go test ./internal/connectors -run TestLiveVendorParity -v -count=1
//
// A divergent vendor breaks a subtest here and earns a driver fix with a
// fixture test next to it; what cannot be exercised headlessly (OAuth
// completion, TUI-only flows) is recorded in
// docs/handoff/open/connectors-parity.md instead of faked here.

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/cfpperche/picode/internal/mcp"
)

const (
	liveEnv     = "PICODE_CONNECTORS_LIVE"
	liveSandbox = "PICODE_LIVE_SANDBOX"
)

// liveSkip guards the whole harness: it must be opted in, and the process
// HOME must be the declared sandbox — otherwise a stray run would point the
// vendor CLIs at the real user store.
func liveSkip(t *testing.T) string {
	t.Helper()
	if os.Getenv(liveEnv) != "1" {
		t.Skip("set PICODE_CONNECTORS_LIVE=1 with HOME pointed at a sandbox to verify against the real vendor CLIs")
	}
	sb := os.Getenv(liveSandbox)
	if sb == "" || sb != os.Getenv("HOME") {
		t.Skip("PICODE_LIVE_SANDBOX must equal the process HOME (the vendor CLIs write there)")
	}
	return sb
}

func liveInstalled(t *testing.T, bin string) {
	t.Helper()
	if _, err := exec.LookPath(bin); err != nil {
		t.Skipf("%s is not installed on this machine", bin)
	}
}

// liveRun shells out to a real vendor CLI with a hard timeout; a hung CLI
// fails the test instead of wedging the run.
func liveRun(t *testing.T, timeout time.Duration, dir string, bin string, args ...string) string {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, bin, args...)
	cmd.Dir = dir
	out, _ := cmd.CombinedOutput()
	if ctx.Err() != nil {
		t.Fatalf("%s %s timed out after %s: %s", bin, strings.Join(args, " "), timeout, out)
	}
	return string(out)
}

func liveWS(t *testing.T) string {
	t.Helper()
	return t.TempDir()
}

// urlEntry is a harmless definition: example.invalid never resolves, so a
// health check fails loudly but nothing ever leaves the machine.
func urlEntry() mcp.Entry {
	return mcp.Entry{URL: "https://example.invalid/mcp", Auth: "oauth"}
}

// stdioEntry runs /bin/cat, which waits for EOF and exits when a vendor
// closes the pipe.
func stdioEntry() mcp.Entry {
	return mcp.Entry{Command: "/bin/cat", Args: []string{"-u"}}
}

func TestLiveVendorParity(t *testing.T) {
	sb := liveSkip(t)

	// Claude Code: the user store is the vendor's own; every user-scope
	// mutation is a `claude mcp …` round trip and List parses its output.
	t.Run("claude-code", func(t *testing.T) {
		liveInstalled(t, "claude")
		p := Paths{Home: sb, Cwd: liveWS(t)}
		d := Claude{}

		if err := d.Add(p, "user", "remoteserver", urlEntry()); err != nil {
			t.Fatalf("add url: %v", err)
		}
		if err := d.Add(p, "user", "localserver", stdioEntry()); err != nil {
			t.Fatalf("add stdio: %v", err)
		}
		store, err := os.ReadFile(filepath.Join(sb, ".claude.json"))
		if err != nil {
			t.Fatal(err)
		}
		for _, want := range []string{`"remoteserver"`, `"type": "http"`, `"localserver"`, `"/bin/cat"`} {
			if !strings.Contains(string(store), want) {
				t.Fatalf("vendor store missing %s:\n%s", want, store)
			}
		}
		rep, err := d.List(p)
		if err != nil {
			t.Fatalf("list: %v", err)
		}
		byName := map[string]mcp.Server{}
		for _, s := range rep.Servers {
			byName[s.Name] = s
		}
		if s := byName["remoteserver"]; s.URL != "https://example.invalid/mcp" || s.Scope != "user" {
			t.Fatalf("remoteserver row = %+v", s)
		}
		// The line format folds args into the display target.
		if s := byName["localserver"]; !strings.HasPrefix(s.Command, "/bin/cat") {
			t.Fatalf("localserver row = %+v", s)
		}
		if err := d.Remove(p, "user", "remoteserver"); err != nil {
			t.Fatalf("remove: %v", err)
		}
		if store, err := os.ReadFile(filepath.Join(sb, ".claude.json")); err != nil || strings.Contains(string(store), "remoteserver") {
			t.Fatalf("remove left the server in the store: %s (%v)", store, err)
		}
		// The sign-in hint is the vendor's own command; headless it refuses
		// honestly instead of hanging (measured 2.1.277).
		out := liveRun(t, 30*time.Second, p.home(), "claude", "mcp", "login", "localserver")
		if !strings.Contains(out, "localserver") {
			t.Fatalf("login output names nothing: %q", out)
		}
	})

	// Codex: file codec both ways — the vendor reads PiCode's tables and
	// PiCode reads the vendor's (mcp add writes command/args/url verbatim).
	t.Run("codex", func(t *testing.T) {
		liveInstalled(t, "codex")
		p := Paths{Home: sb, Cwd: liveWS(t)}
		d := Codex{}

		out := liveRun(t, 60*time.Second, p.home(), "codex", "mcp", "add", "vendored", "--url", "https://example.invalid/mcp")
		if !strings.Contains(out, "Added") {
			t.Fatalf("vendor add: %q", out)
		}
		rep, err := d.List(p)
		if err != nil {
			t.Fatalf("list: %v", err)
		}
		found := false
		for _, s := range rep.Servers {
			if s.Name == "vendored" {
				found = s.URL == "https://example.invalid/mcp"
			}
		}
		if !found {
			t.Fatalf("driver did not read the vendor's url entry: %+v", rep.Servers)
		}
		if err := d.Add(p, "user", "picode", stdioEntry()); err != nil {
			t.Fatalf("add: %v", err)
		}
		out = liveRun(t, 60*time.Second, p.home(), "codex", "mcp", "get", "picode")
		if !strings.Contains(out, "/bin/cat") {
			t.Fatalf("vendor does not see the PiCode entry: %q", out)
		}
		if err := d.Toggle(p, "user", "picode", true); err != nil {
			t.Fatalf("toggle: %v", err)
		}
		out = liveRun(t, 60*time.Second, p.home(), "codex", "mcp", "get", "picode")
		if !strings.Contains(out, "disabled") {
			t.Fatalf("vendor does not honor the enabled flag: %q", out)
		}
		if err := d.Remove(p, "user", "picode"); err != nil {
			t.Fatalf("remove: %v", err)
		}
		out = liveRun(t, 60*time.Second, p.home(), "codex", "mcp", "get", "picode")
		if !strings.Contains(strings.ToLower(out), "no mcp server named") {
			t.Fatalf("vendor still lists the removed server: %q", out)
		}
	})

	// Omp: no vendor CLI for MCP — the proof that it loads the driver's
	// file is a marker a spawned stdio server touches. Slow: two agent
	// startups. The command is /bin/touch itself (no shell metachars, so it
	// passes the add form's validation); it exits after the marker, which
	// still proves the spawn.
	t.Run("omp", func(t *testing.T) {
		liveInstalled(t, "omp")
		marker := filepath.Join(sb, "omp-marker")
		p := Paths{Home: sb, Cwd: liveWS(t)}
		d := Omp{}
		entry := mcp.Entry{Command: "/bin/touch", Args: []string{marker}}

		if err := d.Add(p, "user", "marker", entry); err != nil {
			t.Fatalf("add: %v", err)
		}
		_ = os.Remove(marker)
		liveRun(t, 90*time.Second, p.Cwd, "omp", "-p", "hi")
		if _, err := os.Stat(marker); err != nil {
			t.Fatalf("omp did not spawn the stdio server from %s: %v", p.home(), err)
		}
		if err := d.Toggle(p, "user", "marker", true); err != nil {
			t.Fatalf("toggle: %v", err)
		}
		_ = os.Remove(marker)
		liveRun(t, 90*time.Second, p.Cwd, "omp", "-p", "hi")
		if _, err := os.Stat(marker); err == nil {
			t.Fatal("omp spawned a disabled server")
		}
		// The workspace layer is a real file the vendor loads too.
		if err := d.Add(p, "project", "marker", entry); err != nil {
			t.Fatalf("project add: %v", err)
		}
		_ = os.Remove(marker)
		liveRun(t, 90*time.Second, p.Cwd, "omp", "-p", "hi")
		if _, err := os.Stat(marker); err != nil {
			t.Fatal("omp did not load the project .omp/mcp.json")
		}
	})

	// Antigravity: the vendor CLI is the oracle — it manages the same user
	// file and reports enabled/disabled state.
	t.Run("agy", func(t *testing.T) {
		liveInstalled(t, "agy")
		p := Paths{Home: sb, Cwd: liveWS(t)}
		d := AGY{}

		if err := d.Add(p, "user", "picode", urlEntry()); err != nil {
			t.Fatalf("add: %v", err)
		}
		out := liveRun(t, 30*time.Second, p.Cwd, "agy", "mcp", "list")
		if !strings.Contains(out, "picode") {
			t.Fatalf("vendor list misses the PiCode entry: %q", out)
		}
		if err := d.Toggle(p, "user", "picode", true); err != nil {
			t.Fatalf("toggle: %v", err)
		}
		out = liveRun(t, 30*time.Second, p.Cwd, "agy", "mcp", "list")
		if !strings.Contains(out, "disabled") {
			t.Fatalf("vendor does not report the disabled flag: %q", out)
		}
		if err := d.Remove(p, "user", "picode"); err != nil {
			t.Fatalf("remove: %v", err)
		}
		out = liveRun(t, 30*time.Second, p.Cwd, "agy", "mcp", "list")
		if strings.Contains(out, "picode") {
			t.Fatalf("vendor still lists the removed server: %q", out)
		}
	})

	// OpenCode: the .jsonc is the vendor's own store — a PiCode write must
	// land there when it exists, and `opencode mcp list` must see it.
	t.Run("opencode", func(t *testing.T) {
		liveInstalled(t, "opencode")
		p := Paths{Home: sb, Cwd: liveWS(t)}
		d := OpenCode{}

		jsoncDir := filepath.Join(sb, ".config", "opencode")
		if err := os.MkdirAll(jsoncDir, 0o755); err != nil {
			t.Fatal(err)
		}
		jsonc := `{
  "$schema": "https://opencode.ai/config.json",
  "mcp": {}
}
`
		if err := os.WriteFile(filepath.Join(jsoncDir, "opencode.jsonc"), []byte(jsonc), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := d.Add(p, "user", "picode", stdioEntry()); err != nil {
			t.Fatalf("add: %v", err)
		}
		wrote, err := os.ReadFile(filepath.Join(jsoncDir, "opencode.jsonc"))
		if err != nil || !strings.Contains(string(wrote), "picode") {
			t.Fatalf("add did not land in the vendor's jsonc (%v): %s", err, wrote)
		}
		out := liveRun(t, 90*time.Second, p.Cwd, "opencode", "mcp", "list")
		if !strings.Contains(out, "picode") {
			t.Fatalf("vendor list misses the PiCode entry: %q", out)
		}
		if err := d.Toggle(p, "user", "picode", true); err != nil {
			t.Fatalf("toggle: %v", err)
		}
		out = liveRun(t, 90*time.Second, p.Cwd, "opencode", "mcp", "list")
		if !strings.Contains(out, "disabled") {
			t.Fatalf("vendor does not report the enabled flag: %q", out)
		}
		if err := d.Remove(p, "user", "picode"); err != nil {
			t.Fatalf("remove: %v", err)
		}
	})

	// Grok: the vendor CLI is the oracle for the enabled flag and the
	// disabled_mcp_servers overlay.
	t.Run("grok", func(t *testing.T) {
		liveInstalled(t, "grok")
		p := Paths{Home: sb, Cwd: liveWS(t)}
		d := Grok{}

		if err := d.Add(p, "user", "picode", stdioEntry()); err != nil {
			t.Fatalf("add: %v", err)
		}
		out := liveRun(t, 60*time.Second, p.Cwd, "grok", "mcp", "list")
		if !strings.Contains(out, "picode") {
			t.Fatalf("vendor list misses the PiCode entry: %q", out)
		}
		if err := d.Toggle(p, "user", "picode", true); err != nil {
			t.Fatalf("toggle: %v", err)
		}
		out = liveRun(t, 60*time.Second, p.Cwd, "grok", "mcp", "list")
		if !strings.Contains(out, "disabled") {
			t.Fatalf("vendor does not report the enabled flag: %q", out)
		}
		if err := d.Toggle(p, "user", "picode", false); err != nil {
			t.Fatalf("enable: %v", err)
		}
		out = liveRun(t, 60*time.Second, p.Cwd, "grok", "mcp", "list")
		if strings.Contains(out, "(disabled)") {
			t.Fatalf("vendor still reports disabled: %q", out)
		}
		if err := d.Remove(p, "user", "picode"); err != nil {
			t.Fatalf("remove: %v", err)
		}
		out = liveRun(t, 60*time.Second, p.Cwd, "grok", "mcp", "list")
		if strings.Contains(out, "picode") {
			t.Fatalf("vendor still lists the removed server: %q", out)
		}
		userFile, err := os.ReadFile(filepath.Join(sb, ".grok", "config.toml"))
		if err == nil && strings.Contains(string(userFile), "disabled_mcp_servers") {
			t.Fatalf("stale overlay after remove: %s", userFile)
		}
	})

	// Muse: no add/list CLI — `muse mcp login` recognizing the entry (it
	// attempts OAuth instead of saying "no MCP server named") is the
	// observable that the driver's settings.json shape is the vendor's.
	t.Run("muse", func(t *testing.T) {
		liveInstalled(t, "muse")
		p := Paths{Home: sb, Cwd: liveWS(t)}
		d := Muse{}

		if err := d.Add(p, "user", "picode", urlEntry()); err != nil {
			t.Fatalf("add url: %v", err)
		}
		out := liveRun(t, 60*time.Second, p.Cwd, "muse", "mcp", "login", "picode", "--headless")
		if strings.Contains(out, "no MCP server named") {
			t.Fatalf("muse does not recognize the driver's entry: %q", out)
		}
		if err := d.Remove(p, "user", "picode"); err != nil {
			t.Fatalf("remove: %v", err)
		}
		if err := d.Add(p, "user", "picode", stdioEntry()); err != nil {
			t.Fatalf("add stdio: %v", err)
		}
		out = liveRun(t, 60*time.Second, p.Cwd, "muse", "mcp", "login", "picode", "--headless")
		if !strings.Contains(out, "stdio") {
			t.Fatalf("muse does not read the stdio transport: %q", out)
		}
	})

	// Hermes: the vendor's own list is the oracle for the YAML block.
	t.Run("hermes", func(t *testing.T) {
		liveInstalled(t, "hermes")
		p := Paths{Home: sb, Cwd: liveWS(t)}
		d := Hermes{}

		if err := d.Add(p, "user", "picode", urlEntry()); err != nil {
			t.Fatalf("add: %v", err)
		}
		out := liveRun(t, 90*time.Second, p.Cwd, "hermes", "mcp", "list")
		if !strings.Contains(out, "picode") {
			t.Fatalf("vendor list misses the PiCode entry: %q", out)
		}
		if err := d.Toggle(p, "user", "picode", true); err != nil {
			t.Fatalf("toggle: %v", err)
		}
		out = liveRun(t, 90*time.Second, p.Cwd, "hermes", "mcp", "list")
		if !strings.Contains(out, "disabled") {
			t.Fatalf("vendor does not report the enabled flag: %q", out)
		}
		if err := d.Remove(p, "user", "picode"); err != nil {
			t.Fatalf("remove: %v", err)
		}
	})
}
