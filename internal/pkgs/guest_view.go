// The guest pane's JSON, derived from the unified model (ADR-0176 slice 2), as
// legacy.go derives Pi's. The bytes the pane parses stay exactly what the
// handlers answered while they called clipkgs directly: these mappers are pure,
// so the two can be compared over fixtures in a test instead of trusted.
package pkgs

import "github.com/cfpperche/picode/internal/clipkgs"

// GuestView is GET /api/cli-packages (and /updates) as the guest pane reads
// it: the declaration, the rows, and the stamps that tell a fresh compare from
// one that never ran.
type GuestView struct {
	CLI       string            `json:"cli"`
	Scopes    []clipkgs.Scope   `json:"scopes"`
	Caps      clipkgs.Caps      `json:"caps"`
	Notes     map[string]string `json:"notes"`
	Rows      []clipkgs.Row     `json:"rows"`
	Note      string            `json:"note,omitempty"`
	ReadAt    string            `json:"readAt,omitempty"`
	CheckedAt string            `json:"checkedAt,omitempty"`
}

// Guest maps one guest read onto that payload. cli is the identity the request
// carried: the pane reads back exactly what it asked with.
func Guest(cli string, r Report) GuestView {
	return GuestView{
		CLI:       cli,
		Scopes:    guestScopes(r.Scopes),
		Caps:      guestCaps(r.Caps),
		Notes:     guestNotes(r.Notes),
		Rows:      guestRowsLegacy(r.Rows),
		Note:      r.Note,
		ReadAt:    r.ReadAt,
		CheckedAt: r.CheckedAt,
	}
}

// AvailableView is GET /api/cli-packages/available as the pane reads it. The
// field order is the payload's: this route answered a map, whose keys marshal
// sorted, and the pane's bytes must not move (ADR-0176's acceptance).
type AvailableView struct {
	CLI  string        `json:"cli"`
	Note string        `json:"note"`
	Rows []clipkgs.Row `json:"rows"`
}

// GuestAvailable maps the CLI's catalog read onto that payload.
func GuestAvailable(cli string, r Report) AvailableView {
	return AvailableView{CLI: cli, Note: r.Note, Rows: guestRowsLegacy(r.Rows)}
}

// MarketplacesView is GET /api/cli-packages/marketplaces as the pane reads it:
// the sources the CLI is configured with.
type MarketplacesView struct {
	CLI          string        `json:"cli"`
	Marketplaces []clipkgs.Row `json:"marketplaces"`
}

// GuestMarketplaces maps the driver's source rows onto that payload.
func GuestMarketplaces(cli string, rows []Row) MarketplacesView {
	return MarketplacesView{CLI: cli, Marketplaces: guestRowsLegacy(rows)}
}

// GuestViewOf is the same payload built from the engine's own report: the
// mutation handlers read through clipkgs and answer with this until they move
// onto the driver (slice 2b).
func GuestViewOf(cli string, rep clipkgs.Report) GuestView {
	rows := rep.Rows
	if rows == nil {
		rows = []clipkgs.Row{}
	}
	return GuestView{
		CLI:       cli,
		Scopes:    clipkgs.Scopes(cli),
		Caps:      clipkgs.Capabilities(cli),
		Notes:     clipkgs.Notes(cli),
		Rows:      rows,
		Note:      rep.Note,
		ReadAt:    rep.ReadAt,
		CheckedAt: rep.CheckedAt,
	}
}

// guestScopes is the declaration's scopes as the pane shows them: the CLI's own
// word is the id it reads back, the class stays PiCode's.
func guestScopes(scopes []ScopeRow) []clipkgs.Scope {
	out := make([]clipkgs.Scope, 0, len(scopes))
	for _, s := range scopes {
		out = append(out, clipkgs.Scope{ID: s.Vendor, Label: s.Label, Note: s.Note})
	}
	return out
}

// guestCaps is the model's capability set as the pane parses it. The two sets
// carry the same facts under different names: List and Config are the model's
// own (a guest always lists, and never edits a vendor plugin's config), so the
// pane's payload has no field for them.
func guestCaps(c Caps) clipkgs.Caps {
	return clipkgs.Caps{
		Install:        c.Install,
		Remove:         c.Remove,
		Toggle:         c.Toggle,
		Update:         c.Update,
		Inspect:        c.Inspect,
		Marketplace:    c.Marketplace,
		Available:      c.Available,
		CatalogInstall: c.CatalogInstall,
	}
}

// guestNotes is the declaration's notes as the pane parses them: a map the pane
// indexes by verb, never null.
func guestNotes(notes map[string]string) map[string]string {
	if notes == nil {
		return map[string]string{}
	}
	return notes
}

// guestRowsLegacy is the model's rows as the pane parses them, in the order the
// read returned. A row that is behind carries both halves of the badge — the
// catalog's version and the flag — because the pane gates the Update control
// on the flag and prints the version beside it.
func guestRowsLegacy(rows []Row) []clipkgs.Row {
	out := make([]clipkgs.Row, 0, len(rows))
	for _, r := range rows {
		out = append(out, clipkgs.Row{
			ID:              r.ID,
			Name:            r.Name,
			Version:         r.Version,
			Description:     r.Description,
			Scope:           r.Vendor,
			Enabled:         r.Enabled,
			Installed:       r.Installed,
			Source:          r.Source,
			SourceKind:      r.Kind,
			Marketplace:     r.Marketplace,
			InstallPath:     r.InstalledPath,
			Status:          r.Status,
			Latest:          r.Behind,
			UpdateAvailable: r.Behind != "",
			Note:            r.Note,
			ManagedByPiCode: r.ManagedByPiCode,
		})
	}
	return out
}
