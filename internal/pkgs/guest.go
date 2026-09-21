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

// The mutation half. The verbs split by transport, and that is the whole
// difference between them: the job lane reserves a durable job first and runs
// the argv afterwards — after a restart too — so Install, Remove, Update and a
// marketplace fetch hand back a command and stop there; Toggle, Inspect and a
// marketplace removal are calls whose answer depends on the result (the CLI's
// fresh list, the vendor's own text, the sources that are left), so they run
// here and hand back the command they ran. Every argv and every rendered line
// stays clipkgs': the driver maps the model onto the vendor's own words and
// never restates one.

// Install answers the vendor command a plugin install runs. Caps.Install is the
// pane's gate, and the argv table is the same declaration: a CLI that declares
// none refuses by name, in the engine's own words.
func (g guestDriver) Install(_ context.Context, q Query, t Target) (Command, error) {
	return g.command(q, clipkgs.VerbInstall, t)
}

// Remove answers the vendor command a plugin removal runs. It is the lane's
// verb: a CLI whose only removal is a write of its own config file (OpenCode's
// plugin array) has no command to hand the lane, and refuses exactly as its
// route always has.
func (g guestDriver) Remove(_ context.Context, q Query, t Target) (Command, error) {
	return g.command(q, clipkgs.VerbRemove, t)
}

// Update answers the vendor command one plugin's update runs, where the CLI
// declares an update verb at all (Caps.Update).
func (g guestDriver) Update(_ context.Context, q Query, t Target) (Command, error) {
	return g.command(q, clipkgs.VerbUpdate, t)
}

// Marketplace answers the command one marketplace-source action runs. An add
// and an update fetch, which is the job lane's work, so it builds and stops. A
// removal is a local change, so it runs the vendor's own verb here.
func (g guestDriver) Marketplace(ctx context.Context, q Query, r MarketRequest) (Command, error) {
	cmd, err := g.marketCommand(q, r)
	if err != nil {
		return cmd, err
	}
	if r.Action != "remove" {
		return cmd, nil
	}
	if _, err := clipkgs.Market(ctx, g.cli, r.Action, g.paths(q), g.market(r)); err != nil {
		return cmd, err
	}
	return cmd, nil
}

// Toggle turns one plugin on or off through the vendor's own verb, and runs it:
// a local state change, so the caller answers the CLI's fresh list read back
// from the CLI itself. The line is best effort — a toggle a CLI only offers as
// a write of its own file (OpenCode has no disable command) has none to hand a
// person, and the write happens all the same, with the same empty line its
// route has always answered.
func (g guestDriver) Toggle(ctx context.Context, q Query, t Target) (Command, error) {
	verb := clipkgs.VerbEnable
	if !t.On {
		verb = clipkgs.VerbDisable
	}
	cmd, _ := g.command(q, verb, t)
	if _, err := clipkgs.Run(ctx, g.cli, verb, g.paths(q), g.target(q, t)); err != nil {
		return cmd, err
	}
	return cmd, nil
}

// Inspect runs the CLI's own inspection verb and answers its output verbatim,
// with the command that produced it. A CLI that declares no inspection
// (Caps.Inspect) refuses in the engine's own words — which name the verb
// ("inspects nothing"), so the line is built best effort, exactly as the route
// built it before this verb moved here.
func (g guestDriver) Inspect(ctx context.Context, q Query, t Target) (Command, string, error) {
	cmd, _ := g.command(q, clipkgs.VerbInspect, t)
	out, err := clipkgs.Inspect(ctx, g.cli, g.paths(q), g.target(q, t))
	if err != nil {
		return cmd, "", err
	}
	return cmd, out, nil
}

// command is one vendor command as both halves its callers need: the argv the
// job lane runs and the exact line a pane copies. clipkgs builds both from one
// table, so the two cannot disagree (ADR-0167).
func (g guestDriver) command(q Query, verb clipkgs.Verb, t Target) (Command, error) {
	p := g.paths(q)
	dir, args, err := clipkgs.Argv(g.cli, verb, p, g.target(q, t))
	if err != nil {
		return Command{}, err
	}
	line, err := clipkgs.Command(g.cli, verb, p, g.target(q, t))
	if err != nil {
		return Command{}, err
	}
	return Command{Exe: clipkgs.Bin(g.cli), Args: args, Dir: dir, Line: line}, nil
}

// marketCommand is command for one marketplace-source action.
func (g guestDriver) marketCommand(q Query, r MarketRequest) (Command, error) {
	p := g.paths(q)
	dir, args, err := clipkgs.MarketArgv(g.cli, r.Action, p, g.market(r))
	if err != nil {
		return Command{}, err
	}
	line, err := clipkgs.MarketCommand(g.cli, r.Action, p, g.market(r))
	if err != nil {
		return Command{}, err
	}
	return Command{Exe: clipkgs.Bin(g.cli), Args: args, Dir: dir, Line: line}, nil
}

// target is one plugin as the vendor's own command names it: the scope word the
// request carried (a class cannot say OpenCode's or Claude Code's own words) and
// the state a toggle asks for.
func (g guestDriver) target(q Query, t Target) clipkgs.Target {
	return clipkgs.Target{Name: t.Name, Source: t.Source, Scope: g.mutationWord(q), On: t.On}
}

// market is one marketplace action as the vendor's own command takes it.
func (g guestDriver) market(r MarketRequest) clipkgs.MarketRequest {
	return clipkgs.MarketRequest{Action: r.Action, Source: r.Source, Name: r.Name, Ref: r.Ref}
}

// mutationWord is the scope word a mutation runs with: the caller's, verbatim,
// because the vendors' argv builders take the CLI's own word and default it
// themselves when none was named. Only a caller that named a class alone is
// spelled out — the class cannot say Claude Code's uncommitted `local` layer,
// and the read paths use the same rule.
func (g guestDriver) mutationWord(q Query) string {
	if q.Vendor == "" && q.Scope != "" {
		return vendorScope(q.Scope)
	}
	return q.Vendor
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
