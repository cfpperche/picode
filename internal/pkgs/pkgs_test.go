package pkgs

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/cfpperche/picode/internal/pipkg"
)

// The registry is the decision table: "" is Pi, the eight guests resolve by
// name, and anything else is unknown rather than a default driver.
func TestDriverForResolvesPiAndTheGuests(t *testing.T) {
	rows := []struct {
		cli   string
		id    string
		known bool
	}{
		{"", "pi", true},
		{"pi", "pi", true},
		{" omp ", "omp", true},
		{"claude-code", "claude-code", true},
		{"nope", "nope", false},
		{"pipi", "pipi", false},
	}
	for _, row := range rows {
		if got := DriverFor(row.cli).ID(); got != row.id {
			t.Fatalf("DriverFor(%q).ID() = %q, want %q", row.cli, got, row.id)
		}
		if got := Known(row.cli); got != row.known {
			t.Fatalf("Known(%q) = %v, want %v", row.cli, got, row.known)
		}
	}
}

// Only Pi declares the agent scope: it is PiCode's own list on the agent row.
// A guest that grows the mechanism is a decision, not an accident (ADR-0167).
func TestOnlyPiDeclaresTheAgentScope(t *testing.T) {
	for _, cli := range CLIs() {
		has := slices.ContainsFunc(DriverFor(cli).Scopes(), func(s ScopeRow) bool { return s.ID == Agent })
		if cli == "pi" && !has {
			t.Fatal("pi must declare the agent scope")
		}
		if cli != "pi" && has {
			t.Fatalf("%s declares the agent scope", cli)
		}
	}
}

// Pi's settings become unified rows: the class is ours, the vendor word is
// Pi's, and the agent's own list arrives as data from the caller.
func TestPiDriverMapsSettingsToUnifiedRows(t *testing.T) {
	userDir := t.TempDir()
	old := pipkg.UserDir
	pipkg.UserDir = func() string { return userDir }
	t.Cleanup(func() { pipkg.UserDir = old })

	mustWrite(t, filepath.Join(userDir, "settings.json"), `{"packages":["npm:pi-web-search"]}`)
	ws := t.TempDir()
	mustWrite(t, filepath.Join(ws, ".pi", "settings.json"), `{"packages":["/abs/pi-roles"]}`)

	rep, err := DriverFor("pi").List(context.Background(), Query{
		WorkspacePath: ws,
		AgentSources:  []string{"npm:pi-inbox"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if rep.CLI != "pi" || rep.Catalog != CatalogGallery || rep.Gallery != pipkg.Gallery {
		t.Fatalf("report = %+v", rep)
	}
	if !rep.Capabilities["webSearch"] {
		t.Fatal("a configured search package must set webSearch")
	}
	if rep.Caps.Install != true || rep.Caps.Config != true {
		t.Fatalf("caps = %+v", rep.Caps)
	}
	want := []Row{
		{Name: "pi-web-search", Scope: Machine, Vendor: "user", Kind: "npm"},
		{Name: "pi-roles", Scope: Workspace, Vendor: "project", Kind: "path"},
		{Name: "pi-inbox", Scope: Agent, Vendor: "agent", Kind: "npm"},
	}
	if len(rep.Rows) != len(want) {
		t.Fatalf("rows = %+v", rep.Rows)
	}
	for i, w := range want {
		got := rep.Rows[i]
		if got.CLI != "pi" || got.Name != w.Name || got.Scope != w.Scope || got.Vendor != w.Vendor || got.Kind != w.Kind {
			t.Fatalf("row %d = %+v, want %+v", i, got, w)
		}
		if !got.Enabled {
			t.Fatalf("row %d is not enabled: %+v", i, got)
		}
	}
}

// A guest's word for a scope is kept beside the class, and its caps come from
// the declaration — never restated by hand.
func TestGuestDriverKeepsTheVendorScopeWords(t *testing.T) {
	omp := DriverFor("omp")
	if omp.Caps().Install != true || omp.Caps().Toggle != true {
		t.Fatalf("omp caps = %+v", omp.Caps())
	}
	scopes := omp.Scopes()
	if len(scopes) == 0 || scopes[0].ID != Machine || scopes[0].Vendor != "user" {
		t.Fatalf("omp scopes = %+v", scopes)
	}
	classes := map[Scope]string{}
	for _, s := range scopes {
		classes[s.ID] = s.Vendor
	}
	if classes[Workspace] != "project" {
		t.Fatalf("omp workspace vendor word = %q, want project", classes[Workspace])
	}
	// Claude Code's third layer is `local` (uncommitted project scope): the
	// class is the workspace, the word stays the CLI's.
	claude := map[string]Scope{}
	for _, s := range DriverFor("claude-code").Scopes() {
		claude[s.Vendor] = s.ID
	}
	if claude["local"] != Workspace || claude["user"] != Machine {
		t.Fatalf("claude scopes = %+v", claude)
	}
}

func TestSourceNameReadsEveryKind(t *testing.T) {
	rows := map[string]string{
		"npm:pi-web-search":      "pi-web-search",
		"npm:@scope/pkg@1.2.3":   "@scope/pkg",
		"/home/u/picode/pkg":     "pkg",
		"git:github.com/x/y":     "y",
		"https://github.com/a/b": "b",
		"":                       "",
	}
	for source, want := range rows {
		if got := SourceName(source); got != want {
			t.Fatalf("SourceName(%q) = %q, want %q", source, got, want)
		}
	}
}

func mustWrite(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}
