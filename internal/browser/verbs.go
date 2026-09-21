package browser

import (
	"errors"
	"strings"
)

// Verb is the product's vocabulary for the browser tool: one friendly action,
// the CDP method it runs, and the tier that method needs. The shell's catalog
// stays the authority (it re-checks the method); this table is what the daemon
// is willing to ask for on an agent's behalf. A session drive (ADR-0172) reaches
// the act verbs without a stored grant; raw CDP still needs the full tier.
// A tool never names a CDP method itself — it names a verb, so the surface
// cannot grow behind the catalog's back.
type Verb struct {
	Method string
	Tier   string
	// NeedsURL marks the one shape that has a destination: the route resolves
	// params.url and checks it against the grant before the command leaves.
	NeedsURL bool
	// Raw marks the one verb whose CDP method comes from the caller instead
	// of this table (ADR-0144). It needs the machine's developer mode *and*
	// the full tier, and the daemon records every call.
	Raw bool
}

var verbs = map[string]Verb{
	// The page as an accessibility tree: roles and names, no script runs.
	"snapshot": {Method: "Accessibility.getFullAXTree", Tier: "read"},
	// The page as the human sees it. Big (base64 PNG); the tool writes it out.
	"screenshot": {Method: "Page.captureScreenshot", Tier: "read"},
	// What the tab has recorded since the last poll (never reaches the page).
	"events": {Method: "shell.events", Tier: "read"},
	// history reads the human's own browsing record — not a page, so the
	// daemon answers it from the store and no shell method is named
	// (ADR-0146). Its own gate is the setting `browser.historyAccess`,
	// never by default; the tier is read because nothing is driven.
	"history": {Tier: "read"},

	// --- act: a session split (ADR-0172) reaches these without a stored grant.
	// An unidentified caller does not. ---

	// Run an expression in the page. The page's own trust, which is why an
	// unidentified read has no evaluate.
	"evaluate": {Method: "Runtime.evaluate", Tier: "act"},
	// Go to a URL. http and https only; the session split is not domain-gated.
	"navigate": {Method: "Page.navigate", Tier: "act", NeedsURL: true},
	// Open the split bound to this session. Not CDP: the page creates the pane.
	"open": {Method: "shell.open", Tier: "act"},
	// Click, type and press are one Runtime.evaluate. Prepare builds it.
	"click": {Method: "Runtime.evaluate", Tier: "act"},
	"type":  {Method: "Runtime.evaluate", Tier: "act"},
	"press": {Method: "Runtime.evaluate", Tier: "act"},

	// --- full (ADR-0144): raw CDP, machine opt-in, audited ---

	// A CDP method named by the caller, outside the catalog above. Off unless
	// the owner turned Developer mode on for this machine, and never for a
	// tier below full: the catalog is what keeps an agent's browsing narrow,
	// so the way past it must be deliberate and recorded.
	"cdp": {Tier: "full", Raw: true},
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

// AllowsRaw is the second half of ADR-0144's decision: a raw verb needs the
// full tier *and* the machine's developer mode. Both are inputs here so the
// table is one testable place, and neither alone is enough:
//
// | developer mode | tier  | raw CDP |
// | -------------- | ----- | ------- |
// | off            | any   | refused |
// | on             | read  | refused |
// | on             | act   | refused |
// | on             | full  | allowed |
func (p Policy) AllowsRaw(developerMode bool) bool {
	return developerMode && p.Rank() >= rankOf("full")
}

// RawMethod reads the method name a raw verb carries. Empty or non-string is
// an error: an unnamed method must never travel as "whatever the page does".
func RawMethod(params map[string]any) (string, error) {
	raw, ok := params["method"]
	if !ok {
		return "", errors.New("cdp needs a method, e.g. {\"method\": \"Network.getAllCookies\"}")
	}
	method, ok := raw.(string)
	if !ok {
		return "", errors.New("cdp method must be a string")
	}
	method = strings.TrimSpace(method)
	if method == "" {
		return "", errors.New("cdp needs a method, e.g. {\"method\": \"Network.getAllCookies\"}")
	}
	return method, nil
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
