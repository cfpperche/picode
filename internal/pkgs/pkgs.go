// Package pkgs is the one package subsystem (ADR-0176): one model, one
// interface, one vocabulary — a driver per CLI instead of an engine per vendor
// identity. Pi and the eight guest CLIs are drivers here; nothing in this
// package knows a vendor's argv or its roster shape, and each driver keeps its
// own engine (pipkg for Pi, clipkgs for the guests) behind the interface.
//
// This is the read half of the interface. The mutation verbs (install, remove,
// update, toggle, inspect) and the surfaces that call them arrive with the
// slices that rewire them; a verb is added here when something reaches it, so
// nothing in this package is speculative.
//
// Scope is the caller's vocabulary, not the vendor's: Machine, Workspace and
// Agent are classes, and each driver keeps the CLI's own word in Vendor
// ("user", "project", "local") so a pane can still say what the CLI says.
package pkgs

import (
	"context"
	"errors"
	"strings"
)

// Scope is the class of layer a row lives in. Every CLI's own list of scopes
// maps onto these three; a driver that does not offer one does not declare it.
type Scope string

const (
	Machine   Scope = "machine"
	Workspace Scope = "workspace"
	Agent     Scope = "agent"
)

// ScopeRow is one scope as a pane shows it: the class, the CLI's own word for
// it, and the line the pane prints under a radio.
type ScopeRow struct {
	ID     Scope  `json:"id"`
	Vendor string `json:"vendor"`
	Label  string `json:"label"`
	Note   string `json:"note,omitempty"`
}

// Catalog is where a CLI's installable list comes from, when it has one.
type Catalog string

const (
	CatalogNone Catalog = ""
	// CatalogGallery is PiCode's own npm search for pi packages (ADR-0010).
	CatalogGallery Catalog = "gallery"
	// CatalogVendor is the CLI's own available list (`--available` and
	// friends) — a vendor fact, read as the vendor wrote it.
	CatalogVendor Catalog = "vendor"
)

// Caps is what a driver's CLI exposes. A control a pane draws is gated here,
// so "this CLI cannot do that" is data instead of a branch.
type Caps struct {
	List        bool `json:"list"`
	Available   bool `json:"available"`
	Install     bool `json:"install"`
	Remove      bool `json:"remove"`
	Update      bool `json:"update"`
	Toggle      bool `json:"toggle"`
	Inspect     bool `json:"inspect"`
	Config      bool `json:"config"`
	Marketplace bool `json:"marketplace"`
	// CatalogInstall says the CLI's own catalog names the spec an install
	// takes, so a pane may offer Install on a catalog row. A catalog that
	// prints a name and a version and never says where the plugin comes from
	// (Omp's) is information: its rows carry no control.
	CatalogInstall bool `json:"catalogInstall"`
}

// Row is one package, from either engine. Source is what a removal takes back
// (a pi source string, a vendor plugin spec, or a path); Name is what the CLI's
// own list prints.
type Row struct {
	CLI string `json:"cli"`
	// ID is the CLI's own identifier for the row — what its commands take and
	// what a pane keys a row by where that is not the printed name
	// (Claude Code's `name@marketplace`).
	ID     string `json:"id,omitempty"`
	Name   string `json:"name"`
	Source string `json:"source,omitempty"`
	Kind   string `json:"kind,omitempty"`   // npm | git | path | plugin | extension
	Scope  Scope  `json:"scope"`            // the class
	Vendor string `json:"vendor,omitempty"` // the CLI's own word for the scope
	// Enabled is what the CLI loads the row as; Installed separates "the CLI
	// has it" from "it is turned on" (Claude and Codex disable without
	// uninstalling, and a catalog row is neither). Pi's rows are installed
	// when stored.
	Enabled   bool   `json:"enabled"`
	Installed bool   `json:"installed"`
	Version   string `json:"version,omitempty"`
	// InstalledPath is where the CLI keeps it, when that is known.
	InstalledPath string `json:"installedPath,omitempty"`
	// ConfigKind names a PiCode config editor for the source (ADR-0119).
	ConfigKind string `json:"configKind,omitempty"`
	// Status is the vendor's own word for the row (`installed`, `available`,
	// a trust level), shown verbatim where the CLI prints one.
	Status string `json:"status,omitempty"`
	// Marketplace is the marketplace a row came from, when the CLI names one:
	// it is what a pane groups and filters a mixed list by.
	Marketplace string `json:"marketplace,omitempty"`
	// Behind is the catalog's version when this row is out of date. It is only
	// ever set from a read of that catalog — never guessed (clipkgs' rule).
	Behind string `json:"behind,omitempty"`
	// Description is the vendor's own sentence about the package; Note is the
	// one line the row must carry (a PiCode-managed integration, a text-parsed
	// roster, a vendor caveat). Two facts, so a row may carry both.
	Description string `json:"description,omitempty"`
	Note        string `json:"note,omitempty"`
	// ManagedByPiCode marks a row PiCode wrote itself, so a reader finds it
	// without knowing which CLI installed it.
	ManagedByPiCode bool `json:"managedByPiCode,omitempty"`
}

// Report is one pane load for one CLI.
type Report struct {
	CLI     string     `json:"cli"`
	Scopes  []ScopeRow `json:"scopes"`
	Rows    []Row      `json:"rows"`
	Caps    Caps       `json:"caps"`
	Catalog Catalog    `json:"catalog,omitempty"`
	Gallery string     `json:"gallery,omitempty"`
	// Notes are the declaration's own sentences, one per verb or concept a
	// pane looks up by name; Note is about this read (a marketplace that
	// needed the network, a text roster), never about a package.
	Notes map[string]string `json:"notes,omitempty"`
	Note  string            `json:"note,omitempty"`
	// ReadAt is when the roster was read; CheckedAt is when its catalog was
	// compared, and stays empty until that check ran — a pane tells "up to
	// date" from "not compared yet" by it.
	ReadAt    string `json:"readAt,omitempty"`
	CheckedAt string `json:"checkedAt,omitempty"`
	// Capabilities are derived facts a surface may gate on (webSearch, …).
	Capabilities map[string]bool `json:"capabilities,omitempty"`
	// Isolated is Pi's "only this agent's packages" switch, carried so a pane
	// can explain why a stored row is not loaded.
	Isolated bool `json:"isolated,omitempty"`
}

// Query is one read. AgentSources and AgentIsolated are how the caller hands
// the agent scope in: the engine does not reach the store, so Pi's per-agent
// list — and its "only this agent's packages" switch — travel as data.
//
// Vendor is the CLI's own word for the scope, which the class cannot always
// say: Claude Code's `local` is a workspace-class layer with its own command
// word. A driver asks its CLI with Vendor when the caller named one, and with
// the class's spelling when it did not.
type Query struct {
	Scope         Scope
	Vendor        string
	WorkspacePath string
	AgentSources  []string
	AgentIsolated bool
	Fresh         bool
}

// ErrNoUpdateCheck is the refusal a caller that asks for an update check its
// CLI does not have gets. Caps.Update is the pane's gate; this is the same fact
// as a sentence, so a route that cannot draw the control has something to say
// instead of an invented empty list.
var ErrNoUpdateCheck = errors.New("this CLI has no update check")

// ErrNoCatalog is the refusal a caller that asks for a CLI's installable list
// gets where that CLI has no such list: Pi's catalog is PiCode's own npm
// gallery, read by search through its own route and page shape, never a roster
// of plugins.
var ErrNoCatalog = errors.New("this CLI's catalog is not a plugin list")

// ErrNoMarketplaces is the refusal for a CLI that keeps no marketplace sources
// of its own: Pi's Marketplace tab is the gallery above, so there is nothing
// the CLI holds to list.
var ErrNoMarketplaces = errors.New("this CLI keeps no marketplace sources")

// Driver is the read surface: what the CLI declares, and what it holds.
type Driver interface {
	ID() string
	Scopes() []ScopeRow
	Caps() Caps
	// List reads what the CLI holds in one scope, from the CLI's own roster
	// command or its own files.
	List(ctx context.Context, q Query) (Report, error)
	// Available reads the CLI's own catalog: the rows it can install from,
	// with the ones it already has marked. A driver whose Caps.Available is
	// false (or whose catalog is not a list, ErrNoCatalog) refuses rather than
	// answering an empty catalog.
	Available(ctx context.Context, q Query) (Report, error)
	// Marketplaces lists the marketplace sources the CLI is configured with —
	// what the pane's Marketplace tab header shows before anything is managed.
	// A CLI that keeps none answers ErrNoMarketplaces.
	Marketplaces(ctx context.Context, q Query) ([]Row, error)
	// CheckUpdates is the badge read: the pane load the update badge comes
	// from — every row the read could compare, each carrying Row.Behind when
	// the CLI's own catalog has moved ahead, plus the read's own note and
	// stamps. It is a read, never the Update mutation; a driver whose
	// Caps.Update is false declares no such read and refuses with
	// ErrNoUpdateCheck or the vendor's own reason.
	CheckUpdates(ctx context.Context, q Query) (Report, error)
}

// DriverFor resolves a CLI to its driver. "" is Pi, the default surface.
func DriverFor(cli string) Driver {
	switch strings.TrimSpace(cli) {
	case "", "pi":
		return piDriver{}
	default:
		return guestDriver{cli: strings.TrimSpace(cli)}
	}
}

// Known reports whether a CLI has a driver. An unknown CLI is not an error
// here: a pane asks before it knows what the machine holds.
func Known(cli string) bool {
	id := strings.TrimSpace(cli)
	if id == "" || id == "pi" {
		return true
	}
	for _, c := range guestCLIs() {
		if c == id {
			return true
		}
	}
	return false
}

// CLIs lists every CLI with a driver, Pi first.
func CLIs() []string {
	return append([]string{"pi"}, guestCLIs()...)
}

// vendorScope is the word a CLI uses for a scope class, for the drivers whose
// vendor vocabulary is user/project.
func vendorScope(s Scope) string {
	if s == Workspace {
		return "project"
	}
	return "user"
}
