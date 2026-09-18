package connectors

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cfpperche/picode/internal/mcp"
)

// The golden fixture: comments, a root key, an unrelated table and one
// mcp_server carrying a ${VAR} placeholder. Every write must keep
// everything outside the edited server's table byte-for-byte identical.
const grokGolden = `# Grok rules — must survive every write.
model = "grok-4"

[mcp_servers.docs]
url = "https://mcp.deepwiki.com/mcp"
startup_timeout_sec = 10

# A table Grok owns.
[theme]
name = "dark"
`

func seedGrok(t *testing.T, scope, text string) (Paths, string) {
	t.Helper()
	home := t.TempDir()
	var path string
	if scope == "user" {
		if text != "" {
			if err := os.MkdirAll(filepath.Join(home, ".grok"), 0o755); err != nil {
				t.Fatal(err)
			}
			path = filepath.Join(home, ".grok", "config.toml")
			if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
				t.Fatal(err)
			}
		}
		return Paths{Home: home}, path
	}
	cwd := ""
	if text != "" {
		cwd = t.TempDir()
		if err := os.MkdirAll(filepath.Join(cwd, ".grok"), 0o755); err != nil {
			t.Fatal(err)
		}
		path = filepath.Join(cwd, ".grok", "config.toml")
		if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return Paths{Home: home, Cwd: cwd}, path
}

func TestGrokAddAndRemovePreserveFileByteForByte(t *testing.T) {
	t.Setenv("PATH", t.TempDir()) // never the real grok
	p, path := seedGrok(t, "project", grokGolden)

	if err := (Grok{}).Add(p, "project", "relay", mcp.Entry{
		URL: "https://relay.example/mcp", Auth: "bearer", BearerToken: "tok",
	}); err != nil {
		t.Fatal(err)
	}
	got := fileText(t, path)
	for _, want := range []string{
		"# Grok rules — must survive every write.\n",
		"model = \"grok-4\"\n",
		"[mcp_servers.docs]\nurl = \"https://mcp.deepwiki.com/mcp\"\nstartup_timeout_sec = 10\n",
		"# A table Grok owns.\n[theme]\nname = \"dark\"\n",
		"[mcp_servers.relay]\nurl = 'https://relay.example/mcp'\n",
		"[mcp_servers.relay.headers]\nAuthorization = 'Bearer tok'\n",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("after add, missing:\n%s\n--- got:\n%s", want, got)
		}
	}

	// Remove deletes the table span and nothing else (the blank line that
	// separated the appended table stays — splice semantics, golden-tested
	// in codex_test.go).
	if err := (Grok{}).Remove(p, "project", "relay"); err != nil {
		t.Fatal(err)
	}
	if got = fileText(t, path); got != grokGolden+"\n" {
		t.Fatalf("remove did not restore the original bytes:\n--- got:\n%q", got)
	}

	rep, err := Grok{}.List(p)
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Servers) != 1 || rep.Servers[0].Name != "docs" || rep.Adapter.Source != "native:grok" {
		t.Fatalf("report = %+v", rep)
	}
}

func TestGrokReplaceKeepsNeighboursExact(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	seed := "# mine\nkey = 1\n\n[mcp_servers.docs]\ncommand = 'old'\n\n[theme]\nname = 'dark'\n"
	p, path := seedGrok(t, "project", seed)

	if err := (Grok{}).Add(p, "project", "docs", mcp.Entry{Command: "npx", Args: []string{"-y", "new"}}); err != nil {
		t.Fatal(err)
	}
	got := fileText(t, path)
	want := "# mine\nkey = 1\n\n[mcp_servers.docs]\nargs = ['-y', 'new']\ncommand = 'npx'\n\n[theme]\nname = 'dark'\n"
	if got != want {
		t.Fatalf("replace changed bytes outside the table:\n--- got:\n%q\n--- want:\n%q", got, want)
	}

	// Idempotent: writing the same definition again changes nothing.
	if err := (Grok{}).Add(p, "project", "docs", mcp.Entry{Command: "npx", Args: []string{"-y", "new"}}); err != nil {
		t.Fatal(err)
	}
	if got = fileText(t, path); got != want {
		t.Fatalf("second add not idempotent:\n%q", got)
	}
}

// Toggle is refused — Grok has no per-server switch — and the refusal must
// leave the file untouched.
func TestGrokToggleRefusesAndKeepsFile(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	p, path := seedGrok(t, "project", grokGolden)

	if err := (Grok{}).Toggle(p, "project", "docs", true); err == nil ||
		!strings.Contains(err.Error(), "remove and re-add instead") {
		t.Fatalf("toggle err = %v, want the remove-and-re-add refusal", err)
	}
	if got := fileText(t, path); got != grokGolden {
		t.Fatalf("refused toggle modified the file:\n%q", got)
	}
	if err := (Grok{}).Toggle(p, "project", "docs", false); err == nil {
		t.Fatal("toggle on must be refused too")
	}
	if err := (Grok{}).Toggle(p, "user", "docs", true); err == nil {
		t.Fatal("user-scope toggle must be refused too")
	}
}

func TestGrokPlaceholdersVerbatim(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	seed := "# c\n\n[mcp_servers.docs]\ncommand = 'node'\nargs = ['${DOCS_HOME}/run.js']\n\n[mcp_servers.docs.env]\nAPI_BASE = '${DOCS_BASE:-https://default.example}/api'\n\n# end\n"
	p, path := seedGrok(t, "user", seed)

	// Adding another server rewrites only its own table: the ${VAR} and
	// ${VAR:-default} placeholders stay byte-exact.
	if err := (Grok{}).Add(p, "user", "relay", mcp.Entry{URL: "https://r.example/mcp"}); err != nil {
		t.Fatal(err)
	}
	got := fileText(t, path)
	for _, want := range []string{
		"args = ['${DOCS_HOME}/run.js']",
		"API_BASE = '${DOCS_BASE:-https://default.example}/api'",
		"url = 'https://r.example/mcp'",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("placeholder or neighbour lost:\n%s", got)
		}
	}

	// Rewriting a table keeps the placeholders it is given, verbatim.
	if err := (Grok{}).Add(p, "user", "docs", mcp.Entry{Command: "node", Args: []string{"${DOCS_HOME}/run.js"}}); err != nil {
		t.Fatal(err)
	}
	if got = fileText(t, path); !strings.Contains(got, "'${DOCS_HOME}/run.js'") {
		t.Fatalf("placeholder not verbatim after rewrite:\n%s", got)
	}
}

func TestGrokUserScope(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	home := t.TempDir()
	p := Paths{Home: home}
	if err := (Grok{}).Add(p, "user", "docs", mcp.Entry{
		Command: "npx", Args: []string{"-y", "docs"}, Env: map[string]string{"TOKEN": "${DOCS_TOKEN}"},
	}); err != nil {
		t.Fatal(err)
	}
	got := fileText(t, filepath.Join(home, ".grok", "config.toml"))
	want := "[mcp_servers.docs]\nargs = ['-y', 'docs']\ncommand = 'npx'\n\n[mcp_servers.docs.env]\nTOKEN = '${DOCS_TOKEN}'\n"
	if got != want {
		t.Fatalf("user add:\n--- got:\n%q\n--- want:\n%q", got, want)
	}
}

func TestGrokAddCreatesProjectFile(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	home := t.TempDir()
	cwd := t.TempDir()
	p := Paths{Home: home, Cwd: cwd}
	// Reading reports the project layer (Grok's project path is
	// deterministic, unlike Codex's dot-folder gate) but does not invent it.
	rep, err := Grok{}.List(p)
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Layers) != 2 || rep.Layers[1].Exists {
		t.Fatalf("layers = %+v", rep.Layers)
	}
	// …but Add creates it on request.
	if err := (Grok{}).Add(p, "project", "docs", mcp.Entry{URL: "https://d.example/mcp"}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(cwd, ".grok", "config.toml")); err != nil {
		t.Fatalf("add did not create the project file: %v", err)
	}
}

func TestGrokWorksWithoutBinary(t *testing.T) {
	t.Setenv("PATH", t.TempDir()) // no grok anywhere
	home := t.TempDir()
	p := Paths{Home: home}
	if err := (Grok{}).Add(p, "user", "docs", mcp.Entry{Command: "npx", Args: []string{"-y", "docs"}}); err != nil {
		t.Fatal(err)
	}
	rep, err := Grok{}.List(p)
	if err != nil {
		t.Fatal(err)
	}
	if rep.Adapter.Installed {
		t.Fatal("installed must be false without the binary")
	}
	if len(rep.Servers) != 1 {
		t.Fatalf("servers = %+v", rep.Servers)
	}
}

func TestGrokMalformedFileRefuses(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	p, path := seedGrok(t, "project", "[mcp_servers.docs\nbroken")
	if err := (Grok{}).Add(p, "project", "docs", mcp.Entry{URL: "https://x.example/mcp"}); err == nil {
		t.Fatal("add on malformed file must fail")
	}
	if err := (Grok{}).Remove(p, "project", "docs"); err == nil {
		t.Fatal("remove on malformed file must fail")
	}
	if raw, _ := os.ReadFile(path); string(raw) != "[mcp_servers.docs\nbroken" {
		t.Fatalf("malformed file was modified: %q", raw)
	}
}

// The decision table (ADR-0150): add/toggle/remove × scope × binary
// missing × malformed file — every row ends in an observable result, never
// a panic and never a silently rewritten file. Toggle refuses in both
// scopes.
func TestGrokDecisionTable(t *testing.T) {
	cases := []struct {
		name    string
		scope   string
		op      string // add | toggle | remove
		seed    string // "" → missing file
		wantErr string
	}{
		{name: "add user no file", scope: "user", op: "add"},
		{name: "add project no cwd", scope: "project", op: "add", wantErr: "select a workspace"},
		{name: "toggle user no file", scope: "user", op: "toggle", wantErr: "remove and re-add"},
		{name: "toggle project no cwd", scope: "project", op: "toggle", wantErr: "remove and re-add"},
		{name: "remove user no file", scope: "user", op: "remove", wantErr: "is not in"},
		{name: "remove project no folder", scope: "project", op: "remove", wantErr: ".grok folder"},
		{name: "remove missing server", scope: "user", op: "remove", seed: "x = 1\n", wantErr: "is not in"},
		{name: "remove malformed", scope: "user", op: "remove", seed: "[mcp_servers.docs\nbroken", wantErr: "not valid TOML"},
		{name: "add on malformed", scope: "user", op: "add", seed: "not = [toml", wantErr: "not valid TOML"},
		{name: "add agent scope", scope: "agent", op: "add", wantErr: "not available yet"},
		{name: "toggle agent scope", scope: "agent", op: "toggle", wantErr: "not available yet"},
		{name: "remove agent scope", scope: "agent", op: "remove", wantErr: "not available yet"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("PATH", t.TempDir())
			p, path := seedGrok(t, tc.scope, tc.seed)
			var err error
			switch tc.op {
			case "add":
				err = Grok{}.Add(p, tc.scope, "docs", mcp.Entry{URL: "https://d.example/mcp"})
			case "toggle":
				err = Grok{}.Toggle(p, tc.scope, "docs", true)
			case "remove":
				err = Grok{}.Remove(p, tc.scope, "docs")
			}
			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("want success, got %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("err = %v, want it to contain %q", err, tc.wantErr)
			}
			if tc.seed != "" {
				if got := fileText(t, path); got != tc.seed {
					t.Fatalf("refused op still modified the file:\n%q", got)
				}
			}
		})
	}
}
