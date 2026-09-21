package pkgs

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/cfpperche/picode/internal/clipkgs"
)

// grokScript is one stub answering every route the pane reads for this CLI:
// the roster (with one enabled and one disabled plugin), the CLI's own catalog
// behind `--available`, and its marketplace sources.
const grokScript = `case "$*" in
  *"marketplace list"*) printf '%s' '{"marketplaces":[{"name":"team","source":{"url":"https://example.test/mp"}}]}' ;;
  *--available*) printf '%s' '[{"status":"available","name":"probe","version":"0.4.0","marketplace":"mp","description":"Probe things"},{"status":"available","name":"steady","version":"1.0.0","marketplace":"mp"}]' ;;
  *) printf '%s' '[{"status":"installed","name":"probe","version":"0.2.0","path":"/p","enabled":true,"description":"Probe things"},{"status":"disabled","name":"off","version":"1.0.0","path":"/o"}]' ;;
esac`

// stubGuest puts a fake vendor binary first on PATH: a driver resolves its CLI
// by name (ADR-0167), so this is the seam these tests read through, and the
// script sees the vendor's own argv exactly as the real binary would.
func stubGuest(t *testing.T, name, script string) {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"+script), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	clipkgs.Invalidate(name)
}

// sameBytes compares two payloads as the pane receives them.
func sameBytes(t *testing.T, got, want any) {
	t.Helper()
	g, err := json.Marshal(got)
	if err != nil {
		t.Fatal(err)
	}
	w, err := json.Marshal(want)
	if err != nil {
		t.Fatal(err)
	}
	if string(g) != string(w) {
		t.Fatalf("payload drifted:\n got %s\nwant %s", g, w)
	}
}

// The guest pane's bytes are derived, not re-invented: each read runs the same
// fixture through the engine and through the driver, and the mapped payload
// marshals to the bytes the handler answered while it called the engine
// directly — including the empty roster, the disabled row and the catalog row
// (ADR-0176 slice 2).
func TestGuestViewMatchesTheEngineByteForByte(t *testing.T) {
	ws := t.TempDir()
	ctx := context.Background()

	t.Run("empty roster", func(t *testing.T) {
		stubGuest(t, "grok", `printf '%s' '[]'`)
		engine, err := clipkgs.List(ctx, "grok", clipkgs.Paths{Cwd: ws}, "user", true)
		if err != nil {
			t.Fatal(err)
		}
		rep, err := DriverFor("grok").List(ctx, Query{Vendor: "user", WorkspacePath: ws, Fresh: true})
		if err != nil {
			t.Fatal(err)
		}
		// The two reads are microseconds apart and the engine stamps each one,
		// so the stamp is the only field that may differ: it is held equal, and
		// its own mapping is pinned by TestGuestViewIsThePanesPayload.
		engine.ReadAt = rep.ReadAt
		sameBytes(t, Guest("grok", rep), GuestViewOf("grok", engine))
	})

	t.Run("installed, disabled and a catalog", func(t *testing.T) {
		stubGuest(t, "grok", grokScript)
		engine, err := clipkgs.List(ctx, "grok", clipkgs.Paths{Cwd: ws}, "user", true)
		if err != nil {
			t.Fatal(err)
		}
		rep, err := DriverFor("grok").List(ctx, Query{Vendor: "user", WorkspacePath: ws, Fresh: true})
		if err != nil {
			t.Fatal(err)
		}
		engine.ReadAt = rep.ReadAt
		sameBytes(t, Guest("grok", rep), GuestViewOf("grok", engine))
	})

	// The catalog and the marketplace list answered maps before this slice, so
	// their key order is the map's — the pane's bytes must not move.
	t.Run("the catalog", func(t *testing.T) {
		stubGuest(t, "grok", grokScript)
		engine, err := clipkgs.Available(ctx, "grok", clipkgs.Paths{Cwd: ws}, "user")
		if err != nil {
			t.Fatal(err)
		}
		rep, err := DriverFor("grok").Available(ctx, Query{Vendor: "user", WorkspacePath: ws})
		if err != nil {
			t.Fatal(err)
		}
		sameBytes(t, GuestAvailable("grok", rep), map[string]any{"cli": "grok", "rows": engine.Rows, "note": engine.Note})
	})

	t.Run("the marketplace sources", func(t *testing.T) {
		stubGuest(t, "grok", grokScript)
		engine, err := clipkgs.Marketplaces(ctx, "grok", clipkgs.Paths{Cwd: ws})
		if err != nil {
			t.Fatal(err)
		}
		rows, err := DriverFor("grok").Marketplaces(ctx, Query{WorkspacePath: ws})
		if err != nil {
			t.Fatal(err)
		}
		sameBytes(t, GuestMarketplaces("grok", rows), map[string]any{"cli": "grok", "marketplaces": engine})
	})

	t.Run("the update check", func(t *testing.T) {
		stubGuest(t, "grok", grokScript)
		engine, err := clipkgs.CheckUpdates(ctx, "grok", clipkgs.Paths{Cwd: ws}, "user", true)
		if err != nil {
			t.Fatal(err)
		}
		rep, err := DriverFor("grok").CheckUpdates(ctx, Query{Vendor: "user", WorkspacePath: ws, Fresh: true})
		if err != nil {
			t.Fatal(err)
		}
		engine.ReadAt, engine.CheckedAt = rep.ReadAt, rep.CheckedAt
		sameBytes(t, Guest("grok", rep), GuestViewOf("grok", engine))
	})

	// A CLI with no source-management verb keeps the engine's own refusal: the
	// pane shows the vendor's reason, never an empty list.
	t.Run("a CLI with no sources", func(t *testing.T) {
		_, err := DriverFor("hermes").Marketplaces(ctx, Query{})
		if _, engineErr := clipkgs.Marketplaces(ctx, "hermes", clipkgs.Paths{}); err == nil || engineErr == nil || err.Error() != engineErr.Error() {
			t.Fatalf("driver err = %v, engine err = %v", err, engineErr)
		}
	})
}

// The pane parses these keys, so the derived payload is pinned as bytes: the
// declaration, the notes the pane indexes by verb, every row field it draws,
// and the two stamps that tell a fresh compare from one that never ran.
func TestGuestViewIsThePanesPayload(t *testing.T) {
	rep := Report{
		CLI: "omp",
		Scopes: []ScopeRow{
			{ID: Machine, Vendor: "user", Label: "This machine"},
			{ID: Workspace, Vendor: "project", Label: "This workspace", Note: "Omp applies a project scope to marketplace installs."},
		},
		Caps:  Caps{List: true, Available: true, Install: true, Toggle: true, Update: true, Inspect: true, Marketplace: true},
		Notes: map[string]string{"inspect": "Inspect prints the vendor's own report."},
		Rows: []Row{
			{
				CLI: "omp", ID: "ext", Name: "ext", Version: "1.2.3", Source: "ext", Kind: "user",
				Scope: Machine, Vendor: "user", Enabled: true, Installed: true, InstalledPath: "/home/u/.omp/ext",
				Status: "installed", Marketplace: "mp", Description: "Does things", Note: "Installed by PiCode.",
				ManagedByPiCode: true,
			},
			{CLI: "omp", ID: "off", Name: "off", Scope: Machine, Vendor: "user"},
			{CLI: "omp", ID: "probe", Name: "probe", Version: "0.2.0", Scope: Workspace, Vendor: "project", Installed: true, Behind: "0.4.0"},
		},
		Note:      "Read from the CLI's own list.",
		ReadAt:    "2026-09-21T10:00:00Z",
		CheckedAt: "2026-09-21T10:01:00Z",
	}
	got, err := json.Marshal(Guest("omp", rep))
	if err != nil {
		t.Fatal(err)
	}
	want := `{"cli":"omp",` +
		`"scopes":[{"id":"user","label":"This machine"},` +
		`{"id":"project","label":"This workspace","note":"Omp applies a project scope to marketplace installs."}],` +
		`"caps":{"install":true,"remove":false,"toggle":true,"update":true,"inspect":true,"marketplace":true,"available":true,"catalogInstall":false},` +
		`"notes":{"inspect":"Inspect prints the vendor's own report."},` +
		`"rows":[` +
		`{"id":"ext","name":"ext","version":"1.2.3","description":"Does things","scope":"user","enabled":true,"installed":true,` +
		`"source":"ext","sourceKind":"user","marketplace":"mp","installPath":"/home/u/.omp/ext","status":"installed",` +
		`"note":"Installed by PiCode.","managedByPiCode":true},` +
		`{"id":"off","name":"off","scope":"user","enabled":false,"installed":false},` +
		`{"id":"probe","name":"probe","version":"0.2.0","scope":"project","enabled":false,"installed":true,"latest":"0.4.0","updateAvailable":true}` +
		`],` +
		`"note":"Read from the CLI's own list.","readAt":"2026-09-21T10:00:00Z","checkedAt":"2026-09-21T10:01:00Z"}`
	if string(got) != want {
		t.Fatalf("payload = %s\n      want %s", got, want)
	}

	// An empty roster is an empty list, never null — the pane's empty state —
	// and a read that compared nothing carries no checkedAt.
	got, err = json.Marshal(Guest("omp", Report{CLI: "omp", Rows: []Row{}, Notes: map[string]string{}, Caps: Caps{List: true, Available: true}}))
	if err != nil {
		t.Fatal(err)
	}
	want = `{"cli":"omp","scopes":[],` +
		`"caps":{"install":false,"remove":false,"toggle":false,"update":false,"inspect":false,"marketplace":false,"available":true,"catalogInstall":false},` +
		`"notes":{},"rows":[]}`
	if string(got) != want {
		t.Fatalf("empty payload = %s\n           want %s", got, want)
	}

	// The two map-shaped payloads keep the key order their previous
	// implementation marshalled.
	got, err = json.Marshal(GuestAvailable("grok", Report{Note: "The CLI's own catalog."}))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != `{"cli":"grok","note":"The CLI's own catalog.","rows":[]}` {
		t.Fatalf("available payload = %s", got)
	}
	got, err = json.Marshal(GuestMarketplaces("grok", nil))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != `{"cli":"grok","marketplaces":[]}` {
		t.Fatalf("marketplaces payload = %s", got)
	}
}

// A scope class cannot say everything a vendor's own command word says: Claude
// Code's `local` is a workspace-class layer with its own word, and the read
// asks the CLI with the word the request carried (Query.Vendor), not with the
// class's spelling.
func TestGuestReadsKeepTheVendorScopeWord(t *testing.T) {
	stubGuest(t, "claude", `printf '%s' '[
  {"id":"here@mp","name":"here","scope":"local","enabled":true},
  {"id":"other@mp","name":"other","scope":"project","enabled":true}]'`)
	rep, err := DriverFor("claude-code").List(context.Background(), Query{
		Scope: Workspace, Vendor: "local", WorkspacePath: t.TempDir(), Fresh: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Rows) != 1 || rep.Rows[0].Name != "here" {
		t.Fatalf("rows = %+v, want the local layer's own row", rep.Rows)
	}
	// The class alone would have asked for `project` — and answered a
	// different list.
	rep, err = DriverFor("claude-code").List(context.Background(), Query{
		Scope: Workspace, WorkspacePath: t.TempDir(), Fresh: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Rows) != 1 || rep.Rows[0].Name != "other" {
		t.Fatalf("rows = %+v, want the class's spelling to be the fallback", rep.Rows)
	}
}
