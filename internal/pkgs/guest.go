package pkgs

import (
	"context"
	"strings"

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
		List:      true, // a CLI without a roster is read from its own files
		Available: c.Available,
		Install:   c.Install,
		Remove:    c.Remove,
		Toggle:    c.Toggle,
		Update:    c.Update,
		Inspect:   c.Inspect,
		// A guest's mutations are the vendor's own commands in the durable job
		// lane ADR-0087 built, so the request answers the job and the pane
		// follows the lane's events — except where the CLI has no command at
		// all and PiCode writes its config file itself (OpenCode's removal),
		// which is one verb, not the whole driver: `Lane` is declared per verb
		// from the same engine table `Caps.Install`/`Remove`/`Update` come
		// from, so the two cannot disagree.
		Lane:           g.lane(),
		Marketplace:    c.Marketplace,
		CatalogInstall: c.CatalogInstall,
		// The agent row's "only this agent's packages" switch is PiCode's own
		// flag rather than a vendor verb, so it is offered only where the
		// CLI's launch can honour it. Omp's does, with the two flags that CLI
		// has for skipping what it would auto-load (ADR-0176 slice 4); no
		// other guest is handed PiCode entries at all.
		IsolatedSwitch: g.cli == "omp",
	}
}

// lane is the transport half of the declaration, derived from the same engine
// table `Caps.Install`/`Remove`/`Update` come from: a verb the CLI reaches with
// its own command is the job lane's, a verb whose only path is PiCode's edit of
// the CLI's config file is not, and the marketplace fetches are the lane's
// wherever the CLI manages sources at all. OpenCode is the mixed CLI this is
// for — its install is `opencode plugin <module>`, its removal a splice of its
// own opencode.json — so one bool could not describe it.
func (g guestDriver) lane() Transport {
	command := func(verb clipkgs.Verb) bool { return clipkgs.Commands(g.cli, verb) }
	return Transport{
		Install:     command(clipkgs.VerbInstall),
		Remove:      command(clipkgs.VerbRemove),
		Update:      command(clipkgs.VerbUpdate),
		Marketplace: clipkgs.MarketFetches(g.cli),
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
func (g guestDriver) report(q Query, rows []Row, note, readAt, checkedAt string) Report {
	rep := Report{
		CLI:       g.cli,
		Scopes:    scopesForAgentContext(g.Scopes(), q.AgentName),
		Caps:      g.Caps(),
		Rows:      rows,
		Notes:     clipkgs.Notes(g.cli),
		Note:      note,
		ReadAt:    readAt,
		CheckedAt: checkedAt,
		// The layer a row lives in is named the way the caller named it: the
		// agent scope is PiCode's own list, so its row is badged with the
		// agent's name rather than the class (ADR-0176 slice 4). The vendor's
		// own rows keep their own words.
		AgentName: q.AgentName,
	}
	if rep.Caps.Available {
		rep.Catalog = CatalogVendor
	}
	return rep
}

// List reads the CLI's own roster (or its files, where it has no roster verb).
// Vendor output PiCode cannot parse is ErrRosterShape, never an empty list.
func (g guestDriver) List(ctx context.Context, q Query) (Report, error) {
	// The agent layer is PiCode's own list on the agent row, not a vendor
	// layer: there is no vendor call to make, and the rows are the entries as
	// stored — the launch is what passes them on (ADR-0176 slice 4). This is
	// the same answer Pi's driver gives for the same scope.
	if asksAgentScope(q) {
		return g.report(q, agentSourceRows(g.cli, q.AgentSources), "", "", ""), nil
	}
	legacy, err := clipkgs.List(ctx, g.cli, g.paths(q), g.scopeWord(q), q.Fresh)
	if err != nil {
		return Report{}, err
	}
	return g.report(q, g.rows(legacy.Rows), legacy.Note, legacy.ReadAt, ""), nil
}

// asksAgentScope says the read asked for PiCode's own list on one agent. The
// pane names that layer two ways — by its class (`scope=agent`) and by the word
// the layer's own row carries (`vendor=agent`, which the unified reads send) —
// and both must answer the same, because the layer is PiCode's rather than the
// vendor's either way (ADR-0176 slice 4).
func asksAgentScope(q Query) bool {
	return q.Scope == Agent || strings.EqualFold(strings.TrimSpace(q.Vendor), "agent")
}

// Available reads the CLI's own catalog — the add surface, with the plugins it
// already has marked. A CLI whose declaration has no such list refuses with
// the vendor's own reason.
func (g guestDriver) Available(ctx context.Context, q Query) (Report, error) {
	legacy, err := clipkgs.Available(ctx, g.cli, g.paths(q), g.scopeWord(q))
	if err != nil {
		return Report{}, err
	}
	return g.report(q, g.rows(legacy.Rows), legacy.Note, legacy.ReadAt, ""), nil
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
	// The agent layer has no vendor catalog: the entries are PiCode's own, and
	// nothing about them is "behind" — the badge read answers empty rather than
	// asking the vendor for a layer it does not have (ADR-0176 slice 4).
	if asksAgentScope(q) {
		return Report{CLI: g.cli, Rows: []Row{}, Scopes: scopesForAgentContext(g.Scopes(), q.AgentName), AgentName: q.AgentName}, nil
	}
	legacy, err := clipkgs.CheckUpdates(ctx, g.cli, g.paths(q), g.scopeWord(q), q.Fresh)
	if err != nil {
		return Report{}, err
	}
	return g.report(q, g.rows(legacy.Rows), legacy.Note, legacy.ReadAt, legacy.CheckedAt), nil
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

// Remove answers the vendor command a plugin removal runs, or performs the
// removal itself where the CLI has no such command. It is the lane's verb
// wherever the vendor has an argv; a CLI whose only removal is a write of its
// own config file (OpenCode's `plugin` array) has none to hand the lane, so the
// engine performs the splice here and the empty command is what tells the route
// there is nothing to run — the answer is then the CLI's fresh list, which is
// also what its `Lane.Remove: false` declares. A module the config does not
// name is refused by name, from the engine.
//
// Omp's own `extensions` are the same kind of fact read from a different file —
// a row that came from the workspace settings is written back by PiCode, and the
// user layer goes through the CLI's config command — so an extension row is
// asked of the engine first, by the identity the pane sent.
func (g guestDriver) Remove(ctx context.Context, q Query, t Target) (Command, error) {
	if g.cli == "omp" {
		rm, ok, err := clipkgs.OmpExtensionRemove(ctx, g.paths(q), g.target(q, t))
		if ok || err != nil {
			if rm.InProcess {
				// The workspace layer has no vendor command — the engine wrote
				// the CLI's own settings file — so there is nothing for a lane
				// to run: the mutation has already happened.
				return Command{}, err
			}
			return Command{Exe: clipkgs.Bin(g.cli), Args: rm.Args, Dir: rm.Dir, Line: rm.Line}, err
		}
	}
	if clipkgs.Writes(g.cli, clipkgs.VerbRemove) {
		_, err := clipkgs.Run(ctx, g.cli, clipkgs.VerbRemove, g.paths(q), g.target(q, t))
		return Command{}, err
	}
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
//
// Omp's own `extensions` are the same kind of fact as its removal: the CLI's
// plugin verbs do not know a configured extension, so an extension row is asked
// of the engine first, by the identity the pane sent. The engine performs that
// write here — the workspace layer is a file PiCode splices, and the user layer
// the CLI's own `config set` — because a toggle is synchronous: there is no
// argv for a lane, and the answer is the fresh list either way.
func (g guestDriver) Toggle(ctx context.Context, q Query, t Target) (Command, error) {
	verb := clipkgs.VerbEnable
	if !t.On {
		verb = clipkgs.VerbDisable
	}
	if g.cli == "omp" {
		tog, ok, err := clipkgs.OmpExtensionToggle(ctx, g.paths(q), g.target(q, t))
		if ok || err != nil {
			if tog.InProcess {
				// The workspace settings file is PiCode's own write: there is no
				// vendor command to hand a person, and nothing left to run.
				return Command{}, err
			}
			return Command{Exe: clipkgs.Bin(g.cli), Args: tog.Args, Dir: tog.Dir, Line: tog.Line}, err
		}
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

// agentSourceRows maps PiCode's own entries for one agent onto rows — the same
// shape Pi's driver answers with (`pipkg.WithAgent`): the source as written,
// the agent as the vendor word, and no "installed" claim, because the CLI loads
// them at its next start and that is a fact about the launch, not the store.
func agentSourceRows(cli string, sources []string) []Row {
	out := make([]Row, 0, len(sources))
	for _, s := range sources {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		out = append(out, Row{CLI: cli, Name: SourceName(s), Source: s, Scope: Agent, Vendor: "agent", Enabled: true})
	}
	return out
}

// scopesForAgentContext drops the agent row when the read named no agent: that
// layer is PiCode's list on one agent, so a read that named none cannot answer
// it and must not offer the radio (the plan's decision table). The vendor's own
// rows keep their words — only the class PiCode owns is filtered.
func scopesForAgentContext(scopes []ScopeRow, agent string) []ScopeRow {
	if strings.TrimSpace(agent) != "" {
		return scopes
	}
	out := make([]ScopeRow, 0, len(scopes))
	for _, s := range scopes {
		if s.ID == Agent {
			continue
		}
		out = append(out, s)
	}
	return out
}

// guestCLIs lists the CLIs with a guest driver, in the catalog's order.
func guestCLIs() []string { return clipkgs.CLIs() }
