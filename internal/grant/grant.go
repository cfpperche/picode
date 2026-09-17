// Package grant holds the house identity rule for per-principal grants
// (ADR-0143): which setting key a caller's grant lives under. The work
// browser (browser.policy.*) and the computer tool (computer.policy.*,
// ADR-0148) both key their grants by it, so the rule is written once and a
// terminal principal is spelled the same way everywhere.
package grant

import "strings"

// TerminalPrefix names a terminal principal's grant: a CLI running in a
// PiCode terminal has no agent id of its own, and the prefix keeps the two
// namespaces apart in the same setting table.
const TerminalPrefix = "term:"

// Key returns the grant key for a caller: a managed agent id wins, a terminal
// id is the second branch (prefixed), and a caller with neither is unmanaged
// and gets "" — no key, no grant of its own, whatever the surface's default
// is. Identity is an assertion carried by the install token (ADR-0134); the
// key only says which grant to read.
func Key(agentID, termID string) string {
	if id := strings.TrimSpace(agentID); id != "" {
		return id
	}
	if term := strings.TrimSpace(termID); term != "" {
		return TerminalPrefix + term
	}
	return ""
}
