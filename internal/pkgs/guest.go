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

// Caps is the vendor's own declaration, classed: every field clipkgs derives
// from a CLI's argv table has a home here, so a pane can gate a control on the
// model alone — including the catalog check, which this driver really answers.
func (g guestDriver) Caps() Caps {
	c := clipkgs.Capabilities(g.cli)
	return Caps{
		List:           true, // a CLI without a roster is read from its own files
		Available:      c.Available,
		Install:        c.Install,
		Remove:         c.Remove,
		Toggle:         c.Toggle,
		Update:         c.Update,
		Inspect:        c.Inspect,
		Marketplace:    c.Marketplace,
		CatalogInstall: c.CatalogInstall,
	}
}

// scopeWord is the word this CLI's own commands take for a read: the caller's
// when it named one (the pane sends `local` for Claude Code's uncommitted
// project layer, which the class alone cannot say), the class's spelling when
// it did not.
func (g guestDriver) scopeWord(q Query) string {
	if q.Vendor != "" {
		return q.Vendor
	}
	return vendorScope(q.Scope)
}

// paths is what a vendor call touches. The machine scope reads the CLI's own
// default directories; a project read runs in the workspace folder.
func (g guestDriver) paths(q Query) clipkgs.Paths { return clipkgs.Paths{Cwd: q.WorkspacePath} }

// report is the declaration half of every guest read — what the CLI is — so
// the four reads cannot describe it differently, and each one adds only what
// its own read found (the note and the stamps).
func (g guestDriver) report(rows []Row, note, readAt, checkedAt string) Report {
	rep := Report{
		CLI:       g.cli,
		Scopes:    g.Scopes(),
		Caps:      g.Caps(),
		Rows:      rows,
		Notes:     clipkgs.Notes(g.cli),
		Note:      note,
		ReadAt:    readAt,
		CheckedAt: checkedAt,
	}
	if rep.Caps.Available {
		rep.Catalog = CatalogVendor
	}
	return rep
}

// List reads the CLI's own roster (or its files, where it has no roster verb).
// Vendor output PiCode cannot parse is ErrRosterShape, never an empty list.
func (g guestDriver) List(ctx context.Context, q Query) (Report, error) {
	legacy, err := clipkgs.List(ctx, g.cli, g.paths(q), g.scopeWord(q), q.Fresh)
	if err != nil {
		return Report{}, err
	}
	return g.report(g.rows(legacy.Rows), legacy.Note, legacy.ReadAt, ""), nil
}

// Available reads the CLI's own catalog — the add surface, with the plugins it
// already has marked. A CLI whose declaration has no such list refuses with
// the vendor's own reason.
func (g guestDriver) Available(ctx context.Context, q Query) (Report, error) {
	legacy, err := clipkgs.Available(ctx, g.cli, g.paths(q), g.scopeWord(q))
	if err != nil {
		return Report{}, err
	}
	return g.report(g.rows(legacy.Rows), legacy.Note, legacy.ReadAt, ""), nil
}

// Marketplaces lists the sources the CLI is configured with. A CLI with no
// source-management verb answers ErrVerbAbsent — the same refusal its own
// route gave — and an empty list is the vendor saying it has none.
func (g guestDriver) Marketplaces(ctx context.Context, q Query) ([]Row, error) {
	legacy, err := clipkgs.Marketplaces(ctx, g.cli, g.paths(q))
	if err != nil {
		return nil, err
	}
	return g.rows(legacy), nil
}

// CheckUpdates asks the CLI's own catalog what moved ahead of its roster, and
// answers the pane load the badge comes from. Both halves are the vendor's: a
// catalog PiCode cannot read is reported with no badges and the reason in the
// note, never as "up to date", and the stamps say when each half was read.
func (g guestDriver) CheckUpdates(ctx context.Context, q Query) (Report, error) {
	legacy, err := clipkgs.CheckUpdates(ctx, g.cli, g.paths(q), g.scopeWord(q), q.Fresh)
	if err != nil {
		return Report{}, err
	}
	return g.report(g.rows(legacy.Rows), legacy.Note, legacy.ReadAt, legacy.CheckedAt), nil
}

// rows maps the vendor's own rows onto the unified model. Every field the pane
// reads has a home here: a field without one would be lost on the way back out,
// and a row is only "behind" when its catalog said so (Row.Behind).
func (g guestDriver) rows(legacy []clipkgs.Row) []Row {
	out := make([]Row, 0, len(legacy))
	for _, r := range legacy {
		behind := ""
		if r.UpdateAvailable {
			behind = r.Latest
		}
		out = append(out, Row{
			CLI:             g.cli,
			ID:              r.ID,
			Name:            r.Name,
			Source:          r.Source,
			Kind:            r.SourceKind,
			Scope:           scopeClass(r.Scope),
			Vendor:          r.Scope,
			Enabled:         r.Enabled,
			Installed:       r.Installed,
			Version:         r.Version,
			InstalledPath:   r.InstallPath,
			Status:          r.Status,
			Marketplace:     r.Marketplace,
			Behind:          behind,
			Description:     r.Description,
			Note:            r.Note,
			ManagedByPiCode: r.ManagedByPiCode,
		})
	}
	return out
}

// guestCLIs lists the CLIs with a guest driver, in the catalog's order.
func guestCLIs() []string { return clipkgs.CLIs() }
