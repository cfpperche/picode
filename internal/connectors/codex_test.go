package connectors

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cfpperche/picode/internal/mcp"
)

// The golden fixture: comments, a root key, an unrelated table and one
// mcp_server. Every write must keep everything outside the edited server's
// table byte-for-byte identical.
const codexGolden = `# Top comment — must survive every write.
model = "gpt-5.2"
disable_response_storage = true

[mcp_servers.docs]
url = "https://mcp.deepwiki.com/mcp"

[mcp_servers.docs.env]
API_BASE = "${DOCS_BASE}/api"

# A profile table Codex owns.
[profiles.fast]
model = "gpt-5-mini"
`

// seedCodex plants the one config file Codex loads (the user file, measured
// against codex-cli 0.155.0); scope "project" only opens an empty workspace
// directory — there is no project file for the driver to read or write.
func seedCodex(t *testing.T, scope, text string) (Paths, string) {
	t.Helper()
	home := t.TempDir()
	var path string
	if text != "" {
		if err := os.MkdirAll(filepath.Join(home, ".codex"), 0o755); err != nil {
			t.Fatal(err)
		}
		path = filepath.Join(home, ".codex", "config.toml")
		if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	p := Paths{Home: home}
	if scope == "project" {
		p.Cwd = t.TempDir()
	}
	return p, path
}

func fileText(t *testing.T, path string) string {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

func TestCodexAddPreservesFileByteForByte(t *testing.T) {
	t.Setenv("PATH", t.TempDir()) // never the real codex
	p, path := seedCodex(t, "user", codexGolden)

	if err := (Codex{}).Add(p, "user", "relay", mcp.Entry{
		URL: "https://relay.example/mcp", Auth: "bearer", BearerToken: "tok",
	}); err != nil {
		t.Fatal(err)
	}
	got := fileText(t, path)
	for _, want := range []string{
		"# Top comment — must survive every write.\n",
		"model = \"gpt-5.2\"\n",
		"[mcp_servers.docs]\nurl = \"https://mcp.deepwiki.com/mcp\"\n",
		"[mcp_servers.docs.env]\nAPI_BASE = \"${DOCS_BASE}/api\"\n",
		"# A profile table Codex owns.\n[profiles.fast]\nmodel = \"gpt-5-mini\"\n",
		"[mcp_servers.relay]\nurl = 'https://relay.example/mcp'\n",
		"[mcp_servers.relay.http_headers]\nAuthorization = 'Bearer tok'\n",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("after add, missing:\n%s\n--- got:\n%s", want, got)
		}
	}
}

func TestCodexReplaceKeepsNeighboursExact(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	seed := "# mine\nkey = 1\n\n[mcp_servers.docs]\ncommand = 'old'\n\n[profiles.fast]\nmodel = 'm'\n"
	p, path := seedCodex(t, "user", seed)

	if err := (Codex{}).Add(p, "user", "docs", mcp.Entry{Command: "npx", Args: []string{"-y", "new"}}); err != nil {
		t.Fatal(err)
	}
	got := fileText(t, path)
	want := "# mine\nkey = 1\n\n[mcp_servers.docs]\nargs = ['-y', 'new']\ncommand = 'npx'\n\n[profiles.fast]\nmodel = 'm'\n"
	if got != want {
		t.Fatalf("replace changed bytes outside the table:\n--- got:\n%q\n--- want:\n%q", got, want)
	}

	// Idempotent: writing the same definition again changes nothing.
	if err := (Codex{}).Add(p, "user", "docs", mcp.Entry{Command: "npx", Args: []string{"-y", "new"}}); err != nil {
		t.Fatal(err)
	}
	if got = fileText(t, path); got != want {
		t.Fatalf("second add not idempotent:\n%q", got)
	}
}

func TestCodexToggleAndRemoveGolden(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	seed := "# keep me\nx = 'y'\n\n[mcp_servers.docs]\nurl = 'https://d.example/mcp'\n\n[profiles.fast]\nmodel = 'm'\n"
	p, path := seedCodex(t, "user", seed)

	if err := (Codex{}).Toggle(p, "user", "docs", true); err != nil {
		t.Fatal(err)
	}
	got := fileText(t, path)
	want := "# keep me\nx = 'y'\n\n[mcp_servers.docs]\nenabled = false\nurl = 'https://d.example/mcp'\n\n[profiles.fast]\nmodel = 'm'\n"
	if got != want {
		t.Fatalf("toggle off:\n--- got:\n%q\n--- want:\n%q", got, want)
	}

	if err := (Codex{}).Toggle(p, "user", "docs", false); err != nil {
		t.Fatal(err)
	}
	got = fileText(t, path)
	want = "# keep me\nx = 'y'\n\n[mcp_servers.docs]\nenabled = true\nurl = 'https://d.example/mcp'\n\n[profiles.fast]\nmodel = 'm'\n"
	if got != want {
		t.Fatalf("toggle on:\n--- got:\n%q\n--- want:\n%q", got, want)
	}

	// Toggle is idempotent: enabling an enabled server rewrites the same bytes.
	if err := (Codex{}).Toggle(p, "user", "docs", false); err != nil {
		t.Fatal(err)
	}
	if got = fileText(t, path); got != want {
		t.Fatalf("second enable not idempotent:\n%q", got)
	}

	if err := (Codex{}).Remove(p, "user", "docs"); err != nil {
		t.Fatal(err)
	}
	got = fileText(t, path)
	want = "# keep me\nx = 'y'\n\n[profiles.fast]\nmodel = 'm'\n"
	if got != want {
		t.Fatalf("remove:\n--- got:\n%q\n--- want:\n%q", got, want)
	}

	rep, err := Codex{}.List(p)
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range rep.Servers {
		if s.Name == "docs" {
			t.Fatalf("docs survived remove: %+v", rep.Servers)
		}
	}
	if rep.Adapter.Source != "native:codex" {
		t.Fatalf("source = %q", rep.Adapter.Source)
	}
}

func TestCodexEnabledFalseReadsDisabled(t *testing.T) {
	// enabled = false in the file reads as a disabled server.
	t.Setenv("PATH", t.TempDir())
	seed := "[mcp_servers.docs]\nurl = 'https://d.example/mcp'\nenabled = false\n"
	p, _ := seedCodex(t, "user", seed)
	rep, err := Codex{}.List(p)
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Servers) != 1 || !rep.Servers[0].Disabled {
		t.Fatalf("servers = %+v", rep.Servers)
	}
}

// Live verification (codex-cli 0.155.0): only ~/.codex/config.toml is ever
// loaded, so project scope refuses all three writes and touches nothing —
// no workspace file appears and the user file keeps its bytes.
func TestCodexProjectScopeRefuses(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	p, path := seedCodex(t, "project", codexGolden)
	before := fileText(t, path)

	// One layer even with a workspace open: the user file is all there is.
	rep, err := Codex{}.List(p)
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Layers) != 1 || rep.Layers[0].Scope != "user" {
		t.Fatalf("layers = %+v", rep.Layers)
	}

	addErr := (Codex{}).Add(p, "project", "docs", mcp.Entry{URL: "https://d.example/mcp"})
	toggleErr := (Codex{}).Toggle(p, "project", "docs", true)
	removeErr := (Codex{}).Remove(p, "project", "docs")
	for _, err := range []error{addErr, toggleErr, removeErr} {
		if err == nil || !strings.Contains(err.Error(), "keeps one config file") {
			t.Fatalf("project-scope op = %v, want the one-config-file refusal", err)
		}
	}
	entries, err := os.ReadDir(p.Cwd)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("workspace not empty: %v", entries)
	}
	if got := fileText(t, path); got != before {
		t.Fatalf("user file changed:\n%q", got)
	}
}

func TestCodexQuotedHeaderRoundTrip(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	seed := "# c\n\n[mcp_servers.\"my-docs\"]\nurl = 'https://d.example/mcp'\n\n[profiles.fast]\nmodel = 'm'\n"
	p, path := seedCodex(t, "user", seed)
	if err := (Codex{}).Toggle(p, "user", "my-docs", true); err != nil {
		t.Fatal(err)
	}
	// The rewrite emits the bare key form — semantically identical TOML —
	// while everything outside the entry survives untouched.
	got := fileText(t, path)
	want := "# c\n\n[mcp_servers.my-docs]\nenabled = false\nurl = 'https://d.example/mcp'\n\n[profiles.fast]\nmodel = 'm'\n"
	if got != want {
		t.Fatalf("quoted header round trip:\n--- got:\n%q\n--- want:\n%q", got, want)
	}
}

func TestCodexUserScope(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	home := t.TempDir()
	p := Paths{Home: home}
	if err := (Codex{}).Add(p, "user", "docs", mcp.Entry{
		Command: "npx", Args: []string{"-y", "docs"}, Env: map[string]string{"TOKEN": "${DOCS_TOKEN}"},
	}); err != nil {
		t.Fatal(err)
	}
	got := fileText(t, filepath.Join(home, ".codex", "config.toml"))
	want := "[mcp_servers.docs]\nargs = ['-y', 'docs']\ncommand = 'npx'\n\n[mcp_servers.docs.env]\nTOKEN = '${DOCS_TOKEN}'\n"
	if got != want {
		t.Fatalf("user add:\n--- got:\n%q\n--- want:\n%q", got, want)
	}
	// ${VAR} stays verbatim through a full read/write cycle.
	if err := (Codex{}).Toggle(p, "user", "docs", true); err != nil {
		t.Fatal(err)
	}
	got = fileText(t, filepath.Join(home, ".codex", "config.toml"))
	if !strings.Contains(got, "TOKEN = '${DOCS_TOKEN}'") {
		t.Fatalf("placeholder not verbatim:\n%s", got)
	}
}

// The decision table (ADR-0150 phase 1): add/toggle/remove × scope × binary
// missing × malformed file — every row ends in an observable result, never a
// panic and never a silently rewritten file. Project scope refuses: Codex
// keeps one config file and there is no per-workspace file to write.
func TestCodexDecisionTable(t *testing.T) {
	cases := []struct {
		name    string
		scope   string
		op      string // add | toggle | remove
		seed    string // "" → missing file
		wantErr string
	}{
		{name: "add user no file", scope: "user", op: "add"},
		{name: "toggle user no file", scope: "user", op: "toggle", wantErr: "is not in"},
		{name: "remove user no file", scope: "user", op: "remove", wantErr: "is not in"},
		{name: "toggle user missing server", scope: "user", op: "toggle", seed: "x = 1\n", wantErr: "is not in"},
		{name: "remove missing server", scope: "user", op: "remove", seed: "x = 1\n", wantErr: "is not in"},
		{name: "toggle malformed", scope: "user", op: "toggle", seed: "[mcp_servers.docs\nbroken", wantErr: "not valid TOML"},
		{name: "remove malformed", scope: "user", op: "remove", seed: "[mcp_servers.docs\nbroken", wantErr: "not valid TOML"},
		{name: "add on malformed", scope: "user", op: "add", seed: "not = [toml", wantErr: "not valid TOML"},
		{name: "add project scope", scope: "project", op: "add", wantErr: "no per-workspace file"},
		{name: "toggle project scope", scope: "project", op: "toggle", wantErr: "no per-workspace file"},
		{name: "remove project scope", scope: "project", op: "remove", wantErr: "no per-workspace file"},
		{name: "add agent scope", scope: "agent", op: "add", wantErr: "not available yet"},
		{name: "toggle agent scope", scope: "agent", op: "toggle", wantErr: "not available yet"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("PATH", t.TempDir())
			p, path := seedCodex(t, tc.scope, tc.seed)
			var err error
			switch tc.op {
			case "add":
				err = Codex{}.Add(p, tc.scope, "docs", mcp.Entry{URL: "https://d.example/mcp"})
			case "toggle":
				err = Codex{}.Toggle(p, tc.scope, "docs", true)
			case "remove":
				err = Codex{}.Remove(p, tc.scope, "docs")
			}
			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("want success, got %v", err)
				}
				if tc.seed == "" && tc.op == "add" {
					if _, serr := os.Stat(filepath.Join(p.Home, ".codex", "config.toml")); serr != nil {
						t.Fatalf("add did not create the user file: %v", serr)
					}
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
