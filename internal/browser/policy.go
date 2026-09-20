package browser

import (
	"encoding/json"
	"strings"

	"github.com/cfpperche/picode/internal/grant"
	"github.com/cfpperche/picode/internal/store"
)

// Policy is what an agent may do in the work browser (ADR-0128): a tier that
// gates the method catalog, and the origins it may reach. It travels with
// every command so the shell can re-check both.
type Policy struct {
	Tier    string   `json:"tier"`
	Domains []string `json:"domains,omitempty"`
}

// Default is ADR-0134: an agent with no explicit grant reads the tab the human
// has on screen and nothing else. Empty domains means "no origin of its own" —
// it does not widen the on-screen target, because the target is the tab, not a
// domain.
func Default() Policy {
	return Policy{Tier: "read"}
}

// SettingPrefix names one setting per agent ("browser.policy.<agent id>").
const SettingPrefix = "browser.policy."

// Resolve reads an agent's grant and falls back to Default. An unreadable or
// invalid grant falls back too: a broken setting must not become a wider
// permission, and the default is the narrowest thing that is still useful.
func Resolve(st *store.Store, agentID string) Policy {
	if st == nil || strings.TrimSpace(agentID) == "" {
		return Default()
	}
	raw, ok, err := st.GetSetting(SettingPrefix + agentID)
	if err != nil || !ok {
		return Default()
	}
	var p Policy
	if json.Unmarshal([]byte(raw), &p) != nil {
		return Default()
	}
	if _, ok := tierOrder(p.Tier); !ok {
		return Default()
	}
	return p
}

// tierOrder ranks the tiers so "at least read" is one comparison. Unknown
// tiers are absent (and therefore refused).
func tierOrder(tier string) (int, bool) {
	switch strings.ToLower(strings.TrimSpace(tier)) {
	case "read":
		return 1, true
	case "act":
		return 2, true
	case "full":
		return 3, true
	}
	return 0, false
}

// TerminalPrefix names a terminal principal's grant (ADR-0143). The rule
// lives in package grant so the computer tool (ADR-0148) keys its grants the
// same way; this alias keeps the browser's callers unchanged.
const TerminalPrefix = grant.TerminalPrefix

// ResolveCaller applies the house identity rule (ADR-0143, the same one
// pi-inbox and pi-checklist use): a managed agent id wins, a terminal id is
// the second branch, and a caller with neither is unmanaged — Default(),
// always. A terminal that never got a grant therefore reads the tab on
// screen, exactly like an agent without one.
func ResolveCaller(st *store.Store, agentID, termID string) Policy {
	key := grant.Key(agentID, termID)
	if key == "" {
		return Default()
	}
	return Resolve(st, key)
}

// Save writes one agent's grant (slice 4's editor is the writer).
func Save(st *store.Store, agentID string, p Policy) error {
	if _, ok := tierOrder(p.Tier); !ok {
		return ErrBadTier
	}
	body, err := json.Marshal(p)
	if err != nil {
		return err
	}
	return st.SetSetting(SettingPrefix+agentID, string(body))
}

// ErrBadTier: the grant names a tier that does not exist.
var ErrBadTier = badTierError{}

type badTierError struct{}

func (badTierError) Error() string { return "the tier must be read, act or full" }
