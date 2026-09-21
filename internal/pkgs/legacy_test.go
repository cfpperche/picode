package pkgs

import (
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
	"testing"

	"github.com/cfpperche/picode/internal/pipkg"
)

// The mapping is exact, not approximate: the same settings read through the
// driver and mapped back marshal to the bytes the engine alone answered, so
// the pane cannot tell which path served it (ADR-0176 slice 1c).
func TestLegacyReportMatchesTheEngineByteForByte(t *testing.T) {
	userDir := t.TempDir()
	old := pipkg.UserDir
	pipkg.UserDir = func() string { return userDir }
	t.Cleanup(func() { pipkg.UserDir = old })

	// Machine scope: a plain source, a filtered one (stored, not loaded), and
	// one that sets the web-search capability.
	mustWrite(t, filepath.Join(userDir, "settings.json"),
		`{"packages":["npm:pi-web-search",{"source":"npm:pi-roles","extensions":["inbox"]},"git:github.com/x/y"]}`)
	// Workspace scope, so the project layer's own scope word is in the payload.
	ws := t.TempDir()
	mustWrite(t, filepath.Join(ws, ".pi", "settings.json"), `{"packages":["npm:pi-checklist"]}`)

	agentSources := []string{"npm:pi-inbox"}

	want, err := pipkg.List(pipkg.UserDir(), ws)
	if err != nil {
		t.Fatal(err)
	}
	want = pipkg.WithAgent(want, agentSources)
	want.Isolated = true

	rep, err := DriverFor("pi").List(context.Background(), Query{
		WorkspacePath: ws,
		AgentSources:  agentSources,
		AgentIsolated: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	got, err := json.Marshal(rep.Legacy())
	if err != nil {
		t.Fatal(err)
	}
	engine, err := json.Marshal(want)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(engine) {
		t.Fatalf("legacy JSON drifted:\n got %s\nwant %s", got, engine)
	}
}

// The pane parses these keys, so the derived payload is pinned as bytes: one
// row per scope class, the filtered flag's polarity, both optional fields and
// the report's own gallery, capability and switch.
func TestLegacyReportIsThePanesPayload(t *testing.T) {
	rep := Report{
		CLI:          "pi",
		Catalog:      CatalogGallery,
		Gallery:      pipkg.Gallery,
		Capabilities: map[string]bool{"webSearch": true},
		Isolated:     true,
		Rows: []Row{
			{CLI: "pi", Name: "pi-web-search", Source: "npm:pi-web-search", Kind: "npm", Scope: Machine, Vendor: "user", Enabled: true, Installed: true, ConfigKind: "web-search"},
			{CLI: "pi", Name: "pi-roles", Source: "npm:pi-roles", Kind: "npm", Scope: Workspace, Vendor: "project", Enabled: false, Installed: true, InstalledPath: "/ws/.pi/npm/node_modules/pi-roles", ConfigKind: "roles"},
			{CLI: "pi", Name: "pi-inbox", Source: "npm:pi-inbox", Kind: "npm", Scope: Agent, Vendor: "agent", Enabled: true, Installed: true},
		},
	}
	b, err := json.Marshal(rep.Legacy())
	if err != nil {
		t.Fatal(err)
	}
	want := `{"packages":[` +
		`{"source":"npm:pi-web-search","scope":"user","kind":"npm","configKind":"web-search"},` +
		`{"source":"npm:pi-roles","scope":"project","kind":"npm","filtered":true,` +
		`"installedPath":"/ws/.pi/npm/node_modules/pi-roles","configKind":"roles"},` +
		`{"source":"npm:pi-inbox","scope":"agent","kind":"npm"}` +
		`],"capabilities":{"webSearch":true},"gallery":"https://pi.dev/packages","isolated":true}`
	if string(b) != want {
		t.Fatalf("payload = %s\n      want %s", b, want)
	}

	// An empty roster is an empty list, never null — the pane's empty state.
	b, err = json.Marshal(Report{Catalog: CatalogGallery, Gallery: pipkg.Gallery}.Legacy())
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != `{"packages":[],"capabilities":{"webSearch":false},"gallery":"https://pi.dev/packages"}` {
		t.Fatalf("empty payload = %s", b)
	}
}

// The update badge: the rows the catalog moved ahead of, each carrying the
// source to update, the scope it lives in and the installed/latest pair. A
// row that is not behind is dropped, and nothing behind is an empty list.
func TestLegacyUpdatesKeepsOnlyRowsThatAreBehind(t *testing.T) {
	rows := []Row{
		{CLI: "pi", Source: "npm:pi-roles", Vendor: "user", Version: "1.0.0", Behind: "1.1.0"},
		{CLI: "pi", Source: "npm:pi-current", Vendor: "project", Version: "2.0.0"},
		{CLI: "pi", Source: "npm:pi-checklist", Vendor: "agent", Version: "0.2.0", Behind: "0.3.0"},
	}
	b, err := json.Marshal(LegacyUpdates(rows))
	if err != nil {
		t.Fatal(err)
	}
	want := `{"updates":[` +
		`{"source":"npm:pi-roles","scope":"user","current":"1.0.0","latest":"1.1.0"},` +
		`{"source":"npm:pi-checklist","scope":"agent","current":"0.2.0","latest":"0.3.0"}]}`
	if string(b) != want {
		t.Fatalf("payload = %s\n      want %s", b, want)
	}
	b, err = json.Marshal(LegacyUpdates(nil))
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != `{"updates":[]}` {
		t.Fatalf("empty payload = %s", b)
	}
}

// The badge read is the driver's, and a machine with no npm package installed
// asks no registry: the engine's rules (installed path, pinned versions) are
// what keep this offline.
func TestPiDriverCheckUpdatesAnswersWithNothingInstalled(t *testing.T) {
	old := pipkg.UserDir
	pipkg.UserDir = func() string { return t.TempDir() }
	t.Cleanup(func() { pipkg.UserDir = old })

	rows, err := DriverFor("pi").CheckUpdates(context.Background(), Query{})
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 0 {
		t.Fatalf("rows = %+v, want none", rows)
	}
}

// A guest has no catalog check of its own here (its route is
// /api/cli-packages/updates): the read refuses instead of inventing an empty
// list, which is also what its Caps says.
func TestGuestCheckUpdatesRefuses(t *testing.T) {
	d := DriverFor("omp")
	if d.Caps().Update {
		t.Fatal("omp must not declare an update check yet")
	}
	rows, err := d.CheckUpdates(context.Background(), Query{})
	if !errors.Is(err, ErrNoUpdateCheck) {
		t.Fatalf("err = %v, want ErrNoUpdateCheck", err)
	}
	if rows != nil {
		t.Fatalf("rows = %+v, want none", rows)
	}
}
