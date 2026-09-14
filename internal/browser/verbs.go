package browser

import "strings"

// Verb is the product's vocabulary for the browser tool: one friendly action,
// the CDP method it runs, and the tier that method needs. The shell's catalog
// stays the authority (it re-checks the method); this table is what the daemon
// is willing to ask for on an agent's behalf, and it stays read-tier until the
// grants exist (ADR-0134). A tool never names a CDP method itself — it names a
// verb, so the surface cannot grow behind the catalog's back.
type Verb struct {
	Method string
	Tier   string
	// NeedsURL marks the one shape that has a destination: the route resolves
	// params.url and checks it against the grant before the command leaves.
	NeedsURL bool
}

var verbs = map[string]Verb{
	// The page as an accessibility tree: roles and names, no script runs.
	"snapshot": {Method: "Accessibility.getFullAXTree", Tier: "read"},
	// The page as the human sees it. Big (base64 PNG); the tool writes it out.
	"screenshot": {Method: "Page.captureScreenshot", Tier: "read"},
	// What the tab has recorded since the last poll (never reaches the page).
	"events": {Method: "shell.events", Tier: "read"},

	// --- act (ADR-0128): only an explicit grant reaches these ---

	// Run an expression in the page. The page's own trust, which is why read
	// has no evaluate and is genuinely read-only.
	"evaluate": {Method: "Runtime.evaluate", Tier: "act"},
	// Go to a URL. The one verb with a destination: the grant's domains are
	// checked here and again at the navigation gate.
	"navigate": {Method: "Page.navigate", Tier: "act", NeedsURL: true},
}

// VerbFor resolves a tool's action, case-insensitively. Unknown verbs are
// refused: the vocabulary is closed.
func VerbFor(name string) (Verb, bool) {
	v, ok := verbs[strings.ToLower(strings.TrimSpace(name))]
	return v, ok
}

// Verbs lists the vocabulary, for the tool's help text and the settings
// surface that will publish it.
func Verbs() map[string]Verb {
	out := make(map[string]Verb, len(verbs))
	for k, v := range verbs {
		out[k] = v
	}
	return out
}

// Allows reports whether a policy's tier reaches what the verb needs. It is
// the daemon's own gate — the shell checks the method again, and neither alone
// grants what the other refuses.
func (p Policy) Allows(v Verb) bool {
	return p.Rank() >= rankOf(v.Tier)
}

// Rank orders the tiers for comparison; 0 is "no tier" (refused).
func (p Policy) Rank() int {
	r, _ := tierOrder(p.Tier)
	return r
}

func rankOf(tier string) int {
	r, _ := tierOrder(tier)
	return r
}
