package pkgs

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/cfpperche/picode/internal/clipkgs"
)

// grokScript is one stub answering every read the pane makes for this CLI: the
// roster (with one enabled and one disabled plugin), the CLI's own catalog
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

// sameGuestRows compares the model's rows with the engine's own, one fact at a
// time: the driver may rename a field (the class stays in Scope while the CLI's
// own word travels in Vendor, Kind replaces SourceKind, InstalledPath stands
// for InstallPath, and Behind carries both the catalog's version and the flag
// the pane badges with), but every value a pane reads has to be the engine's.
func sameGuestRows(t *testing.T, got []Row, want []clipkgs.Row) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("rows = %d, want %d", len(got), len(want))
	}
	for i := range got {
		g, w := got[i], want[i]
		if g.ID != w.ID || g.Name != w.Name || g.Version != w.Version || g.Description != w.Description ||
			g.Vendor != w.Scope || g.Enabled != w.Enabled || g.Installed != w.Installed ||
			g.Source != w.Source || g.Kind != w.SourceKind || g.Marketplace != w.Marketplace ||
			g.InstalledPath != w.InstallPath || g.Status != w.Status || g.Behind != w.Latest ||
			g.Note != w.Note || g.ManagedByPiCode != w.ManagedByPiCode {
			t.Fatalf("row %d = %+v, want the engine's %+v", i, g, w)
		}
	}
}

// The driver derives its rows from the engine's, so each read has to carry the
// engine's own facts — including the empty roster, the disabled row and the
// catalog row (ADR-0176). The mappers that re-derived the bytes the guest pane
// used to parse are gone: what a pane reads now is this mapping, field by
// field.
func TestGuestRowsCarryTheEnginesFacts(t *testing.T) {
	ws := t.TempDir()
	ctx := context.Background()

	t.Run("an empty roster is an empty list", func(t *testing.T) {
		stubGuest(t, "grok", `printf '%s' '[]'`)
		rep, err := DriverFor("grok").List(ctx, Query{Vendor: "user", WorkspacePath: ws, Fresh: true})
		if err != nil {
			t.Fatal(err)
		}
		if rep.Rows == nil || len(rep.Rows) != 0 {
			t.Fatalf("rows = %+v, want an empty list", rep.Rows)
		}
	})

	t.Run("the roster", func(t *testing.T) {
		stubGuest(t, "grok", grokScript)
		engine, err := clipkgs.List(ctx, "grok", clipkgs.Paths{Cwd: ws}, "user", true)
		if err != nil {
			t.Fatal(err)
		}
		rep, err := DriverFor("grok").List(ctx, Query{Vendor: "user", WorkspacePath: ws, Fresh: true})
		if err != nil {
			t.Fatal(err)
		}
		if rep.CLI != "grok" {
			t.Fatalf("cli = %q, want the CLI the read asked for", rep.CLI)
		}
		sameGuestRows(t, rep.Rows, engine.Rows)
	})

	// The catalog and the source list are the vendor's own rows too, and a row
	// the catalog knows is marked installed only where the roster has it.
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
		sameGuestRows(t, rep.Rows, engine.Rows)
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
		sameGuestRows(t, rows, engine)
	})

	// The badge read is the same roster with the catalog's comparison on it, so
	// the row that is behind carries the version the catalog offers.
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
		if rep.ReadAt == "" || rep.CheckedAt == "" {
			t.Fatalf("a compared read carries both stamps: %+v", rep)
		}
		sameGuestRows(t, rep.Rows, engine.Rows)
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
