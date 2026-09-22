package server

import (
	"strings"

	"github.com/cfpperche/picode/internal/grant"
)

// callerAgentID names the agent behind a caller (ADR-0184): the agent id it
// carries, else the agent bound to its terminal. A shell or sign-in
// terminal has none.
func callerAgentID(deps Deps, agentID, termID string) string {
	if id := strings.TrimSpace(agentID); id != "" {
		return id
	}
	if term := strings.TrimSpace(termID); term != "" && deps.Store != nil {
		if a, err := deps.Store.AgentByTerminal(term); err == nil {
			return a.ID
		}
	}
	return ""
}

// callerKey is the caller's principal key: its agent, else `term:<id>` —
// an identity that audits and a session drive can name, but that holds no
// stored grant (computer.Resolve, browser.Resolve).
func callerKey(deps Deps, agentID, termID string) string {
	if id := callerAgentID(deps, agentID, termID); id != "" {
		return id
	}
	return grant.Key("", termID)
}

// errShellGrant is the answer to a grant edit aimed at a terminal no agent
// binds.
const errShellGrant = "A shell has no permissions of its own. Make it an agent to give it one."
