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

func seedCodex(t *testing.T, scope, text string) (Paths, string) {
	t.Helper()
	home := t.TempDir()
	var path string
	if scope == "user" {
		if text != "" {
			if err := os.MkdirAll(filepath.Join(home, ".codex"), 0o755); err != nil {
				t.Fatal(err)
			}
			path = filepath.Join(home, ".codex", "config.toml")
			if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
				t.Fatal(err)
			}
		}
		return Paths{Home: home}, path
	}
	cwd := ""
	if text != "" {
		cwd = t.TempDir()
		if err := os.MkdirAll(filepath.Join(cwd, ".codex"), 0o755); err != nil {
			t.Fatal(err)
		}
		path = filepath.Join(cwd, ".codex", "config.toml")
		if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return Paths{Home: home, Cwd: cwd}, path
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
	p, path := seedCodex(t, "project", codexGolden)

	if err := (Codex{}).Add(p, "project", "relay", mcp.Entry{
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
	p, path := seedCodex(t, "project", seed)

	if err := (Codex{}).Add(p, "project", "docs", mcp.Entry{Command: "npx", Args: []string{"-y", "new"}}); err != nil {
		t.Fatal(err)
	}
	got := fileText(t, path)
	want := "# mine\nkey = 1\n\n[mcp_servers.docs]\nargs = ['-y', 'new']\ncommand = 'npx'\n\n[profiles.fast]\nmodel = 'm'\n"
	if got != want {
		t.Fatalf("replace changed bytes outside the table:\n--- got:\n%q\n--- want:\n%q", got, want)
	}

	// Idempotent: writing the same definition again changes nothing.
	if err := (Codex{}).Add(p, "project", "docs", mcp.Entry{Command: "npx", Args: []string{"-y", "new"}}); err != nil {
		t.Fatal(err)
	}
	if got = fileText(t, path); got != want {
		t.Fatalf("second add not idempotent:\n%q", got)
	}
}

func TestCodexToggleAndRemoveGolden(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	seed := "# keep me\nx = 'y'\n\n[mcp_servers.docs]\nurl = 'https://d.example/mcp'\n\n[profiles.fast]\nmodel = 'm'\n"
	p, path := seedCodex(t, "project", seed)

	if err := (Codex{}).Toggle(p, "project", "docs", true); err != nil {
		t.Fatal(err)
	}
	got := fileText(t, path)
	want := "# keep me\nx = 'y'\n\n[mcp_servers.docs]\nenabled = false\nurl = 'https://d.example/mcp'\n\n[profiles.fast]\nmodel = 'm'\n"
	if got != want {
		t.Fatalf("toggle off:\n--- got:\n%q\n--- want:\n%q", got, want)
	}

	if err := (Codex{}).Toggle(p, "project", "docs", false); err != nil {
		t.Fatal(err)
	}
	got = fileText(t, path)
	want = "# keep me\nx = 'y'\n\n[mcp_servers.docs]\nenabled = true\nurl = 'https://d.example/mcp'\n\n[profiles.fast]\nmodel = 'm'\n"
	if got != want {
		t.Fatalf("toggle on:\n--- got:\n%q\n--- want:\n%q", got, want)
	}

	// Toggle is idempotent: enabling an enabled server rewrites the same bytes.
	if err := (Codex{}).Toggle(p, "project", "docs", false); err != nil {
		t.Fatal(err)
	}
	if got = fileText(t, path); got != want {
		t.Fatalf("second enable not idempotent:\n%q", got)
	}

	if err := (Codex{}).Remove(p, "project", "docs"); err != nil {
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

func TestCodexToggleFromOtherLayerDisabled(t *testing.T) {
	// enabled = false in the file reads as a disabled server.
	t.Setenv("PATH", t.TempDir())
	seed := "[mcp_servers.docs]\nurl = 'https://d.example/mcp'\nenabled = false\n"
	p, _ := seedCodex(t, "project", seed)
	rep, err := Codex{}.List(p)
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Servers) != 1 || !rep.Servers[0].Disabled {
		t.Fatalf("servers = %+v", rep.Servers)
	}
}

func TestCodexAddCreatesProjectFile(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	home := t.TempDir()
	cwd := t.TempDir()
	p := Paths{Home: home, Cwd: cwd}
	// Reading does not invent the folder…
	rep, err := Codex{}.List(p)
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Layers) != 1 {
		t.Fatalf("layers = %+v", rep.Layers)
	}
	// …but Add creates it on request.
	if err := (Codex{}).Add(p, "project", "docs", mcp.Entry{URL: "https://d.example/mcp"}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(cwd, ".codex", "config.toml")); err != nil {
		t.Fatalf("add did not create the project file: %v", err)
	}
}

func TestCodexQuotedHeaderRoundTrip(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	seed := "# c\n\n[mcp_servers.\"my-docs\"]\nurl = 'https://d.example/mcp'\n\n[profiles.fast]\nmodel = 'm'\n"
	p, path := seedCodex(t, "project", seed)
	if err := (Codex{}).Toggle(p, "project", "my-docs", true); err != nil {
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
// panic and never a silently rewritten file.
func TestCodexDecisionTable(t *testing.T) {
	cases := []struct {
		name    string
		scope   string
		op      string // add | toggle | remove
		seed    string // "" → missing file
		wantErr string
	}{
		{name: "add user no file", scope: "user", op: "add"},
		{name: "add project no cwd", scope: "project", op: "add", wantErr: "select a workspace"},
		{name: "toggle user no file", scope: "user", op: "toggle", wantErr: "is not in"},
		{name: "toggle project no folder", scope: "project", op: "toggle", wantErr: ".codex folder"},
		{name: "remove user no file", scope: "user", op: "remove", wantErr: "is not in"},
		{name: "remove project no folder", scope: "project", op: "remove", wantErr: ".codex folder"},
		{name: "toggle user missing server", scope: "user", op: "toggle", seed: "x = 1\n", wantErr: "is not in"},
		{name: "remove missing server", scope: "user", op: "remove", seed: "x = 1\n", wantErr: "is not in"},
		{name: "toggle malformed", scope: "user", op: "toggle", seed: "[mcp_servers.docs\nbroken", wantErr: "not valid TOML"},
		{name: "remove malformed", scope: "user", op: "remove", seed: "[mcp_servers.docs\nbroken", wantErr: "not valid TOML"},
		{name: "add on malformed", scope: "user", op: "add", seed: "not = [toml", wantErr: "not valid TOML"},
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

func TestCodexToggleWritePathKeepsOtherScope(t *testing.T) {
	// A docs server in the user file is untouched by a project toggle of a
	// same-named server.
	t.Setenv("PATH", t.TempDir())
	home := t.TempDir()
	if err := os.MkdirAll(filepath.Join(home, ".codex"), 0o755); err != nil {
		t.Fatal(err)
	}
	userSeed := "[mcp_servers.docs]\nurl = 'https://user.example/mcp'\n"
	if err := os.WriteFile(filepath.Join(home, ".codex", "config.toml"), []byte(userSeed), 0o644); err != nil {
		t.Fatal(err)
	}
	p, projectPath := seedCodex(t, "project", "[mcp_servers.docs]\nurl = 'https://project.example/mcp'\n")
	p.Home = home
	if err := (Codex{}).Toggle(p, "project", "docs", true); err != nil {
		t.Fatal(err)
	}
	if got := fileText(t, filepath.Join(home, ".codex", "config.toml")); got != userSeed {
		t.Fatalf("user file changed:\n%q", got)
	}
	if got := fileText(t, projectPath); !strings.Contains(got, "enabled = false") {
		t.Fatalf("project file not toggled:\n%s", got)
	}
}
