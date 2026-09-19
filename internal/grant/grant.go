// Package grant holds the house identity rule for per-principal grants
// (ADR-0143, ADR-0159): which setting key a caller's grant lives under. The
// work browser (browser.policy.*) and the computer tool (computer.policy.*,
// ADR-0148) both key their grants by it, so the rule is written once and a
// terminal principal is spelled the same way everywhere.
package grant

import "strings"

// TerminalPrefix names a terminal principal's grant: a CLI running in a
// PiCode terminal has no agent id of its own, and the prefix keeps the two
// namespaces apart in the same setting table.
const TerminalPrefix = "term:"

// Kind is which namespace a principal lives in (ADR-0159).
type Kind string

const (
	KindNone     Kind = ""
	KindAgent    Kind = "agent"
	KindTerminal Kind = "terminal"
)

// Principal is a managed actor: an agents row (ADR-0160, any catalog CLI),
// or a terminal that is not an agent (unbound CLI / shell). The zero value
// is unmanaged — no key, no grant of its own.
type Principal struct {
	Kind Kind   `json:"kind"`
	ID   string `json:"id"`
}

// Key returns the grant key for a caller: a managed agent id wins, a terminal
// id is the second branch (prefixed), and a caller with neither is unmanaged
// and gets "" — no key, no grant of its own, whatever the surface's default
// is. Identity is an assertion carried by the install token (ADR-0134); the
// key only says which grant to read.
func Key(agentID, termID string) string {
	return FromIDs(agentID, termID).Key()
}

// FromIDs is the house rule as a Principal: agent id wins, else terminal.
func FromIDs(agentID, termID string) Principal {
	if id := strings.TrimSpace(agentID); id != "" {
		return Principal{Kind: KindAgent, ID: id}
	}
	if term := strings.TrimSpace(termID); term != "" {
		return Principal{Kind: KindTerminal, ID: term}
	}
	return Principal{}
}

// ParseKey reverses Key. A key without the terminal prefix is an agent id.
func ParseKey(key string) Principal {
	key = strings.TrimSpace(key)
	if key == "" {
		return Principal{}
	}
	if strings.HasPrefix(key, TerminalPrefix) {
		id := strings.TrimSpace(strings.TrimPrefix(key, TerminalPrefix))
		if id == "" {
			return Principal{}
		}
		return Principal{Kind: KindTerminal, ID: id}
	}
	return Principal{Kind: KindAgent, ID: key}
}

// Key is the grant / setting spelling for this principal.
func (p Principal) Key() string {
	id := strings.TrimSpace(p.ID)
	if id == "" {
		return ""
	}
	switch p.Kind {
	case KindAgent:
		return id
	case KindTerminal:
		return TerminalPrefix + id
	default:
		return ""
	}
}
