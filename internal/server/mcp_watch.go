package server

import (
	"context"
	"time"

	"github.com/cfpperche/picode/internal/mcp"
)

// StartMcpWatch watches pi's live MCP status snapshots (one file per
// managed agent, written by the mcp-live extension bridge) and publishes
// changes as ephemeral mcp.updated events (ADR-0048). This replaces every
// open panel polling /api/mcp on its own: one file read per tick, for the
// whole fleet. The endpoint stays for the feed-down fallback and for
// reconciliation on feed.open.
func StartMcpWatch(ctx context.Context, deps Deps, every time.Duration) {
	if deps.Feed == nil || deps.Store == nil {
		return
	}
	prev := map[string]map[string]string{}
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
		cur := map[string]map[string]string{}
		for _, a := range agents {
			// applyMCPLive reads snapshots of running agents only; a
			// stopped agent's panel flips to idle on the agentRunning
			// change, not here.
			if deps.Runtime == nil || deps.Runtime.Get(a.ID) == nil {
				continue
			}
			live := mcp.ReadLive(mcp.LivePath(deps.DataDir, a.ID), 0)
			if len(live) == 0 {
				continue
			}
			cur[a.ID] = normalizeLive(live)
		}
		for _, id := range diffLive(prev, cur) {
			deps.Feed.Ephemeral("mcp.updated", map[string]any{"agentId": id, "servers": cur[id]})
		}
		prev = cur
	}
}

// normalizeLive folds pi's raw snapshot statuses onto the values the
// report shows (mirrors mcp.ApplyLive), so a raw flip with no visible
// effect does not publish an event.
func normalizeLive(live map[string]string) map[string]string {
	out := make(map[string]string, len(live))
	for name, raw := range live {
		switch raw {
		case "connected":
			out[name] = mcp.LiveOn
		case "failed":
			out[name] = mcp.LiveFailed
		case "needs-auth":
			out[name] = mcp.LiveAuth
		default:
			out[name] = mcp.LiveIdle
		}
	}
	return out
}

// diffLive lists the agents whose normalized snapshot changed between two
// ticks: a server flipped status, a snapshot appeared, or one went away.
func diffLive(prev, cur map[string]map[string]string) []string {
	var out []string
	for id, now := range cur {
		if was, ok := prev[id]; !ok || !liveEqual(was, now) {
			out = append(out, id)
		}
	}
	for id := range prev {
		if _, ok := cur[id]; !ok {
			out = append(out, id)
		}
	}
	return out
}

func liveEqual(a, b map[string]string) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if b[k] != v {
			return false
		}
	}
	return true
}
