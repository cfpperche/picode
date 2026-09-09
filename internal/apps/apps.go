// Package apps is the PiCode apps host (ADR-0036): first-party apps
// declared by manifests, rendering through schema-driven primitives —
// never code loaded into the PiCode process or page. A first-party app
// may instead declare a native surface (ADR-0109): its body is a
// component compiled into a shell, still never code loaded at runtime.
package apps

import (
	"context"

	"github.com/cfpperche/picode/internal/docker"
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

// SurfaceNative marks an app whose body is a component compiled into a
// shell (ADR-0109). The shell that has it registers the app id; every
// other client renders the app's one primitives view instead.
const SurfaceNative = "native"

// Manifest is everything the UI needs to draw an app before any of its
// code runs (grid tile, tab title, palette entry).
type Manifest struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Icon       string `json:"icon"` // key into the UI icon map; falls back to a letter tile
	APIVersion int    `json:"apiVersion"`
	// Surface names how the body renders: "" (omitted) is the primitives
	// view every shell has; SurfaceNative needs a shell that compiled the
	// app in. Optional and additive — APIVersion stays 1 (ADR-0109).
	Surface string `json:"surface,omitempty"`
}

// Host is the deliberately minimal slice of server dependencies an app
// may touch. Defined here so apps never import internal/server; later
// apps (ADR-0037's Inbox) grow Host, not server.Deps.
type Host struct {
	Store   *store.Store
	DataDir string
	Docker  *docker.Service
	Actor   string
	// AgentDeliverable answers whether a reply queued for this agent will
	// be drained automatically — false only when the agent is currently
	// running in a TUI/tmux session, which nothing watches for follow_up
	// tasks (ADR-0037's Inbox). Same type and same polarity as
	// store.AgentDeliverable on purpose: two names for one true/false
	// meaning is how this got inverted the first time. Optional — nil
	// means "assume yes" (tests, the demo app).
	AgentDeliverable store.AgentDeliverable
	// DeliverReply sends an Inbox reply directly into the agent's running
	// terminal TUI (ADR-0060): receiver extension, tmux paste fallback, and
	// durable JSONL proof with reopen-on-failure. It returns the source
	// agent. Optional means this host cannot deliver to a TUI agent.
	DeliverReply func(itemID, verb, text string) (agentID string, err error)
	// DeliverTerminalReply sends an Inbox reply into a pi running in an
	// Agent CLI terminal (sourceKind "terminal", ADR-0089's amendment)
	// through that terminal's receiver. It returns the source terminal.
	// Optional means this host cannot deliver to terminals.
	DeliverTerminalReply func(itemID, verb, text string) (termID string, err error)
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

// BuiltIns assembles the first-party apps. demo adds the hidden QA apps —
// the primitives demo and the native-surface demo (the caller reads
// PICODE_DEMO_APP; env never reaches this package).
func BuiltIns(demo bool) []App {
	list := []App{inboxApp{}, dockerApp{}}
	if demo {
		list = append(list, demoApp{}, nativeDemoApp{})
	}
	return list
}
