package pkgs

import (
	"context"

	"github.com/cfpperche/picode/internal/clipkgs"
)

// guestDriver is one of the eight CLIs whose plugin store the vendor owns
// (ADR-0167) behind the unified interface. The engine stays clipkgs: the argv
// builders, the parsers and the fixtures are vendor facts and do not move.
type guestDriver struct{ cli string }

func (g guestDriver) ID() string { return g.cli }

// Scopes are the CLI's own layers, classed. Claude Code's third layer is
// `local` (uncommitted project scope): it is still the workspace class, and
// Vendor keeps the word the CLI uses.
func (g guestDriver) Scopes() []ScopeRow {
	out := []ScopeRow{}
	for _, s := range clipkgs.Scopes(g.cli) {
		out = append(out, ScopeRow{ID: scopeClass(s.ID), Vendor: s.ID, Label: s.Label, Note: s.Note})
	}
	return out
}

func (g guestDriver) Caps() Caps {
	c := clipkgs.Capabilities(g.cli)
	return Caps{
		List:        true, // a CLI without a roster is read from its own files
		Available:   c.Available,
		Install:     c.Install,
		Remove:      c.Remove,
		Toggle:      c.Toggle,
		Inspect:     c.Inspect,
		Marketplace: c.Marketplace,
	}
}

// List reads the CLI's own roster (or its files, where it has no roster verb).
// Vendor output PiCode cannot parse is ErrRosterShape, never an empty list.
func (g guestDriver) List(ctx context.Context, q Query) (Report, error) {
	legacy, err := clipkgs.List(ctx, g.cli, clipkgs.Paths{Cwd: q.WorkspacePath}, vendorScope(q.Scope), q.Fresh)
	if err != nil {
		return Report{}, err
	}
	rep := Report{
		CLI:    g.cli,
		Scopes: g.Scopes(),
		Caps:   g.Caps(),
		Rows:   make([]Row, 0, len(legacy.Rows)),
		Notes:  clipkgs.Notes(g.cli),
	}
	if g.Caps().Available {
		rep.Catalog = CatalogVendor
	}
	for _, r := range legacy.Rows {
		behind := ""
		if r.UpdateAvailable {
			behind = r.Latest
		}
		detail := r.Description
		if detail == "" {
			detail = r.Note
		}
		rep.Rows = append(rep.Rows, Row{
			CLI:           g.cli,
			Name:          r.Name,
			Source:        r.Source,
			Kind:          r.SourceKind,
			Scope:         scopeClass(r.Scope),
			Vendor:        r.Scope,
			Enabled:       r.Enabled,
			Installed:     r.Installed,
			Version:       r.Version,
			InstalledPath: r.InstallPath,
			Behind:        behind,
			Detail:        detail,
		})
	}
	if legacy.Note != "" {
		if rep.Notes == nil {
			rep.Notes = map[string]string{}
		}
		rep.Notes["roster"] = legacy.Note
	}
	return rep, nil
}

// guestCLIs lists the CLIs with a guest driver, in the catalog's order.
func guestCLIs() []string { return clipkgs.CLIs() }
