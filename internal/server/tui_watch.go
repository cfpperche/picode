package server

import (
	"context"
	"time"

	"github.com/cfpperche/picode/internal/tmux"
)

// StartTuiWatch scrapes pi's own "working" status line out of every
// interactive (tmux) agent's pane on a ticker and publishes changes as
// ephemeral agent.tui events (ADR-0048). This replaces every browser
// polling /api/tui-working on its own: one scrape per tick, for all.
// The endpoint stays for the feed-down fallback and for reconciliation
// on feed.open.
func StartTuiWatch(ctx context.Context, deps Deps, every time.Duration) {
	if deps.Feed == nil || deps.Store == nil || deps.Tmux == nil || !deps.Tmux.Available() {
		return
	}
	prev := map[string]bool{}
	t := time.NewTicker(every)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
		agents, err := deps.Store.ListAllAgents()
		if err != nil {
			continue
		}
		cur := map[string]bool{}
		for _, a := range agents {
			name := tmux.SessionName(a.ID)
			has, err := deps.Tmux.HasSession(ctx, name)
			if err != nil || !has {
				continue
			}
			tail, err := deps.Tmux.CaptureTail(ctx, name, 8)
			if err != nil {
				continue
			}
			cur[a.ID] = tmux.LooksWorking(tail)
		}
		for _, ch := range diffWorking(prev, cur) {
			deps.Feed.Ephemeral("agent.tui", map[string]any{"agentId": ch.id, "working": ch.working, "interactive": ch.interactive})
		}
		prev = cur
	}
}

type tuiChange struct {
	id          string
	working     bool
	interactive bool // false once the tmux session is gone
}

// diffWorking lists the agents whose scraped state changed between two
// ticks: a working flip, a session that appeared, or one that went away.
func diffWorking(prev, cur map[string]bool) []tuiChange {
	var out []tuiChange
	for id, working := range cur {
		if was, ok := prev[id]; !ok || was != working {
			out = append(out, tuiChange{id: id, working: working, interactive: true})
		}
	}
	for id := range prev {
		if _, ok := cur[id]; !ok {
			out = append(out, tuiChange{id: id, working: false, interactive: false})
		}
	}
	return out
}
