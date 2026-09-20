// Package computer is the daemon's half of the computer tool (ADR-0148): one
// grant per principal, a closed catalog of actions, and nothing else. The
// shell is the actuator; the grant is the only gate — capability first, the
// refinements (tiers, bindings, Ask classes) are later amendments.
package computer

import (
	"encoding/json"
	"strings"

	"github.com/cfpperche/picode/internal/grant"
	"github.com/cfpperche/picode/internal/store"
)

// Grant is what a principal has: the switch, and nothing more in v1.
type Grant struct {
	Enabled bool `json:"enabled"`
}

// SettingPrefix names one setting per principal ("computer.policy.<key>",
// the key being grant.Key's: an agent id or "term:<terminal id>").
const SettingPrefix = "computer.policy."

// Resolve reads a principal's grant. Missing, unreadable or malformed is
// off: a broken setting must never become a working grant.
func Resolve(st *store.Store, key string) Grant {
	if st == nil || strings.TrimSpace(key) == "" {
		return Grant{}
	}
	raw, ok, err := st.GetSetting(SettingPrefix + key)
	if err != nil || !ok {
		return Grant{}
	}
	var g Grant
	if json.Unmarshal([]byte(raw), &g) != nil {
		return Grant{}
	}
	return g
}

// ResolveCaller applies the house identity rule (ADR-0143 through
// grant.Key): a managed agent id wins, a terminal id is the second branch,
// and a caller with neither has no grant at all.
func ResolveCaller(st *store.Store, agentID, termID string) Grant {
	return Resolve(st, grant.Key(agentID, termID))
}

// Save writes one principal's grant.
func Save(st *store.Store, key string, enabled bool) error {
	body, err := json.Marshal(Grant{Enabled: enabled})
	if err != nil {
		return err
	}
	return st.SetSetting(SettingPrefix+key, string(body))
}
