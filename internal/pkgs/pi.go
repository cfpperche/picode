package pkgs

import (
	"context"
	"path/filepath"
	"strings"

	"github.com/cfpperche/picode/internal/pipkg"
)

// piDriver is Pi's surface (ADR-0010) behind the unified interface. Pi keeps
// its engine: settings files for machine and workspace scope, the agent's list
// on its row (passed as `pi -e` on every start), `pi install/remove` for
// mutations, the npm gallery as its catalog, and the config descriptors.
type piDriver struct{}

func (piDriver) ID() string { return "pi" }

// Scopes are the three layers Pi has: its own user settings, the workspace's,
// and PiCode's list on the agent row (pi has no native per-agent scope).
func (piDriver) Scopes() []ScopeRow {
	return []ScopeRow{
		{ID: Machine, Vendor: "user", Label: "Global"},
		{ID: Workspace, Vendor: "project", Label: "This workspace", Note: "Installs into the workspace's .pi/settings.json."},
		{ID: Agent, Vendor: "agent", Label: "This agent", Note: "Attached to this agent only, loaded on every start."},
	}
}

// Caps is Pi's own surface: it lists, installs, removes and updates through
// pipkg, carries the config descriptors (ADR-0119), keeps PiCode's npm gallery
// as its catalog — whose rows do name the spec an install takes — and has the
// agent row's "only this agent's packages" switch. Its mutations are direct
// calls that answer the new list, so Async is false and a pane shows the
// transcript PiCode ran instead of the lane's job.
func (piDriver) Caps() Caps {
	return Caps{List: true, Available: true, Install: true, Remove: true, Update: true, Config: true, CatalogInstall: true, IsolatedSwitch: true}
}

// List reads Pi's settings for the machine (and the workspace, when one is
// named) plus whatever the caller resolved for the agent scope.
func (p piDriver) List(_ context.Context, q Query) (Report, error) {
	projectDir := q.WorkspacePath
	legacy, err := pipkg.List(pipkg.UserDir(), projectDir)
	if err != nil {
		return Report{}, err
	}
	if len(q.AgentSources) > 0 {
		legacy = pipkg.WithAgent(legacy, q.AgentSources)
	}
	rep := Report{
		CLI:           p.ID(),
		Scopes:        scopesForContext(p.Scopes(), q.WorkspaceName, q.AgentName),
		WorkspaceName: q.WorkspaceName,
		AgentName:     q.AgentName,
		Caps:          p.Caps(),
		Catalog:       CatalogGallery,
		Gallery:       legacy.Gallery,
		WorkspacePath: q.WorkspacePath,
		Capabilities:  map[string]bool{"webSearch": legacy.Capabilities.WebSearch},
		// The switch lives on PiCode's agent row, not in a settings file, so
		// it arrives with the agent's list (Query.AgentIsolated).
		Isolated: q.AgentIsolated,
		Rows:     make([]Row, 0, len(legacy.Packages)),
	}
	for _, pkg := range legacy.Packages {
		rep.Rows = append(rep.Rows, Row{
			CLI:    p.ID(),
			Name:   SourceName(pkg.Source),
			Source: pkg.Source,
			Kind:   pkg.Kind,
			Scope:  scopeClass(pkg.Scope),
			Vendor: pkg.Scope,
			// Filtered is the row the agent's "only this agent's packages"
			// switch keeps out: stored, not loaded.
			Enabled:       !pkg.Filtered,
			InstalledPath: pkg.InstalledPath,
			ConfigKind:    pkg.ConfigKind,
		})
	}
	return rep, nil
}

// Available refuses, and says why: Pi's installable list is PiCode's own npm
// gallery, read by search through its own route and page shape, never a roster
// of plugins a caller can be handed as rows.
func (piDriver) Available(context.Context, Query) (Report, error) {
	return Report{}, ErrNoCatalog
}

// Marketplaces refuses for the same reason: Pi keeps no source list of its own
// — its Marketplace tab is the gallery above.
func (piDriver) Marketplaces(context.Context, Query) ([]Row, error) {
	return nil, ErrNoMarketplaces
}

// CheckUpdates asks npm which of Pi's rows the registry has moved ahead of —
// the badge read. The engine keeps the rules: path, git, agent and pinned
// npm:@ver rows are skipped, a registry miss skips that row and the rest
// still answer. Pi's payload is the upgrades alone, so its rows carry the ones
// behind and its mapper drops the rest.
func (p piDriver) CheckUpdates(ctx context.Context, q Query) (Report, error) {
	legacy, err := pipkg.CheckUpdates(ctx, pipkg.UserDir(), q.WorkspacePath)
	if err != nil {
		return Report{}, err
	}
	rows := make([]Row, 0, len(legacy.Updates))
	for _, u := range legacy.Updates {
		rows = append(rows, Row{
			CLI:     p.ID(),
			Source:  u.Source,
			Vendor:  u.Scope,
			Version: u.Current,
			Behind:  u.Latest,
		})
	}
	return Report{CLI: p.ID(), Rows: rows}, nil
}

// Pi's mutations are not on this interface yet. They exist here so that Pi is a
// driver like any other, and every one of them refuses with ErrNoMutation: Pi
// installs, removes and updates packages today through pipkg on its own route
// (ADR-0176 moves that in a later slice), and no route of either family reaches
// these verbs for Pi — the guest routes refuse Pi by name before any verb. A
// caller that reaches one is told the truth rather than handed a command PiCode
// would not run.
func (piDriver) Install(context.Context, Query, Target) (Command, error) {
	return Command{}, ErrNoMutation
}
func (piDriver) Remove(context.Context, Query, Target) (Command, error) {
	return Command{}, ErrNoMutation
}
func (piDriver) Update(context.Context, Query, Target) (Command, error) {
	return Command{}, ErrNoMutation
}
func (piDriver) Toggle(context.Context, Query, Target) (Command, error) {
	return Command{}, ErrNoMutation
}
func (piDriver) Inspect(context.Context, Query, Target) (Command, string, error) {
	return Command{}, "", ErrNoMutation
}
func (piDriver) Marketplace(context.Context, Query, MarketRequest) (Command, error) {
	return Command{}, ErrNoMutation
}

// scopesForContext is the declaration as this read can honour it: Pi's workspace
// layer is the folder's settings file and its agent layer is that one agent's
// list, so a read that named neither declares neither — a radio that cannot
// work is not offered (the plan's decision table), and a radio that can is
// named the way the caller names the layer: the folder for the workspace, the
// agent in the line under it.
func scopesForContext(scopes []ScopeRow, workspace, agent string) []ScopeRow {
	out := make([]ScopeRow, 0, len(scopes))
	for _, s := range scopes {
		switch s.ID {
		case Workspace:
			if strings.TrimSpace(workspace) == "" {
				continue
			}
			s.Label = workspace
		case Agent:
			if strings.TrimSpace(agent) == "" {
				continue
			}
			s.Note = "Only " + agent + ", every session"
		}
		out = append(out, s)
	}
	return out
}

// SourceName is what a pane prints for a package source: the npm name without
// its prefix, the folder for a path, the last segment otherwise.
func SourceName(source string) string {
	if name := pipkg.NpmName(source); name != "" {
		return name
	}
	trimmed := strings.TrimSpace(source)
	if trimmed == "" {
		return ""
	}
	if pipkg.KindOf(trimmed) == "path" {
		return filepath.Base(strings.TrimRight(trimmed, "/"))
	}
	if i := strings.LastIndexAny(trimmed, "/:"); i >= 0 && i+1 < len(trimmed) {
		return trimmed[i+1:]
	}
	return trimmed
}

// scopeClass maps a vendor scope word onto the unified class. Pi's own three
// words are already the classes' spelling, Claude Code's `local` is its
// uncommitted project layer; anything else reads as the machine layer, so a
// pane never invents a fourth class.
func scopeClass(vendor string) Scope {
	switch strings.TrimSpace(vendor) {
	case "project", "local":
		return Workspace
	case "agent":
		return Agent
	default:
		return Machine
	}
}
