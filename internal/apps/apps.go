// Package apps is the PiCode apps host (ADR-0036): first-party apps
// declared by manifests, rendering through schema-driven primitives —
// never code loaded into the PiCode process or page.
package apps

import (
	"context"

	"github.com/cfpperche/picode/internal/store"
)

// APIVersion is the primitives contract this binary speaks. A client
// refuses to render a manifest or view whose apiVersion it doesn't know.
const APIVersion = 1

// Badge decorates an app tile: Count for actionable items (numeric
// pill), Dot for non-actionable activity. Zero value = no badge.
type Badge struct {
	Count int  `json:"count,omitempty"`
	Dot   bool `json:"dot,omitempty"`
}

// Manifest is everything the UI needs to draw an app before any of its
// code runs (grid tile, tab title, palette entry).
type Manifest struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Icon       string `json:"icon"` // key into the UI icon map; falls back to a letter tile
	APIVersion int    `json:"apiVersion"`
}

// Host is the deliberately minimal slice of server dependencies an app
// may touch. Defined here so apps never import internal/server; later
// apps (ADR-0037's Inbox) grow Host, not server.Deps.
type Host struct {
	Store   *store.Store
	DataDir string
	// AgentDeliverable answers whether a reply queued for this agent will
	// be drained automatically — false only when the agent is currently
	// running in a TUI/tmux session, which nothing watches for follow_up
	// tasks (ADR-0037's Inbox). Same type and same polarity as
	// store.AgentDeliverable on purpose: two names for one true/false
	// meaning is how this got inverted the first time. Optional — nil
	// means "assume yes" (tests, the demo app).
	AgentDeliverable store.AgentDeliverable
	// OpenAgentTerminal starts (or confirms) the agent's interactive pi
	// TUI — the escape hatch an inbox item offers when a reply cannot be
	// delivered to a TUI agent (ADR-0037). Optional — nil means the host
	// cannot open terminals (tests, the demo app) and the action is not
	// offered.
	OpenAgentTerminal func(agentID string) error
	// SwitchToManaged moves the agent from its interactive TUI to chat
	// mode (the TUI session ends; the thread continues) — the consented
	// switch behind an inbox reply (Degrau 2). Optional — nil means the
	// reply just parks in the queue and delivers on the next managed
	// start.
	SwitchToManaged func(agentID string) error
}

// App is one first-party app. Implementations must be safe for
// concurrent use; the server calls them per request.
type App interface {
	Manifest() Manifest
	Badge(ctx context.Context, h Host) (Badge, error)
	View(ctx context.Context, h Host, path string) (View, error)
	Action(ctx context.Context, h Host, req ActionRequest) (ActionResult, error)
}

// Registry holds the installed apps. Explicit assembly, no init() magic;
// a nil *Registry reads as empty so tests that skip Deps.Apps still work.
type Registry struct {
	apps []App
}

func NewRegistry(list ...App) *Registry {
	return &Registry{apps: list}
}

// All returns the apps in registration order.
func (r *Registry) All() []App {
	if r == nil {
		return nil
	}
	return r.apps
}

// Find returns the app with the given manifest id.
func (r *Registry) Find(id string) (App, bool) {
	if r == nil {
		return nil, false
	}
	for _, a := range r.apps {
		if a.Manifest().ID == id {
			return a, true
		}
	}
	return nil, false
}

// BuiltIns assembles the first-party apps. demo adds the hidden QA app
// (the caller reads PICODE_DEMO_APP; env never reaches this package).
func BuiltIns(demo bool) []App {
	list := []App{inboxApp{}} // ADR-0037: the Inbox is the first production app
	if demo {
		list = append(list, demoApp{})
	}
	return list
}
