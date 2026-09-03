package usage

import (
	"context"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/cfpperche/picode/internal/catalog"
)

// StatusUnknown is a row we have never fetched. It is not an error and not a
// zero: the roster renders a dash and offers Refresh (ADR-0031 — a guessed
// bar is worse than no bar).
const StatusUnknown = "unknown"

// Entry is one cached report plus how old it is. The roster reads only this;
// nothing on a page load reaches a vendor.
type Entry struct {
	Provider  string   `json:"provider"`
	AccountID string   `json:"accountId"`
	Status    string   `json:"status"`
	Plan      string   `json:"plan,omitempty"`
	Error     string   `json:"error,omitempty"`
	Windows   []Window `json:"windows"`
	FetchedAt string   `json:"fetchedAt,omitempty"`
	AgeSec    int64    `json:"ageSec"`
	Resets    int      `json:"resets,omitempty"`
}

type cacheKey struct{ provider, account string }

type cacheVal struct {
	rep Report
	at  time.Time
}

var (
	cacheMu sync.RWMutex
	cache   = map[cacheKey]cacheVal{}
)

func key(provider, accountID string) cacheKey {
	return cacheKey{strings.TrimSpace(provider), strings.TrimSpace(accountID)}
}

// Remember stores a fetched report, including a failed one — "sign in again"
// is state the roster must show, not a hole to retry on every render.
func Remember(provider, accountID string, rep Report, at time.Time) {
	cacheMu.Lock()
	cache[key(provider, accountID)] = cacheVal{rep: rep, at: at}
	cacheMu.Unlock()
}

// Lookup is the cached report for one row.
func Lookup(provider, accountID string) (Report, time.Time, bool) {
	cacheMu.RLock()
	v, ok := cache[key(provider, accountID)]
	cacheMu.RUnlock()
	return v.rep, v.at, ok
}

// Forget drops a row (used when an account is signed out).
func Forget(provider, accountID string) {
	cacheMu.Lock()
	delete(cache, key(provider, accountID))
	cacheMu.Unlock()
}

// ForgetProvider drops every row of a provider.
func ForgetProvider(provider string) {
	p := strings.TrimSpace(provider)
	cacheMu.Lock()
	for k := range cache {
		if k.provider == p {
			delete(cache, k)
		}
	}
	cacheMu.Unlock()
}

func entryOf(provider, accountID string, now time.Time) Entry {
	e := Entry{Provider: provider, AccountID: accountID, Status: StatusUnknown, Windows: []Window{}}
	rep, at, ok := Lookup(provider, accountID)
	if !ok {
		return e
	}
	e.Status = rep.Status
	e.Plan = rep.Plan
	e.Error = rep.Error
	if rep.Windows != nil {
		e.Windows = rep.Windows
	}
	e.Resets = len(rep.Resets)
	e.FetchedAt = rep.FetchedAt
	if !at.IsZero() {
		if d := now.Sub(at); d > 0 {
			e.AgeSec = int64(d / time.Second)
		}
	}
	return e
}

// Summary is every meterable row of every signed-in provider, from cache
// only. Rows are sorted so the response is stable across polls.
func Summary(providers []catalog.Provider, now time.Time) []Entry {
	out := []Entry{}
	for _, p := range providers {
		if !p.SignedIn {
			continue
		}
		if len(p.Accounts) == 0 {
			if catalog.QuotaKind(p.ID, p.AuthType) == "" {
				continue
			}
			out = append(out, entryOf(p.ID, "", now))
			continue
		}
		for _, a := range p.Accounts {
			if a.QuotaKind == "" || a.QuotaKind != a.Type {
				continue
			}
			out = append(out, entryOf(p.ID, a.ID, now))
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Provider != out[j].Provider {
			return out[i].Provider < out[j].Provider
		}
		return out[i].AccountID < out[j].AccountID
	})
	return out
}

// RefreshTarget is one row the background loop may fetch.
type RefreshTarget struct{ Provider, AccountID string }

// ActiveTargets is the active slot of every meterable provider. Only the
// active row refreshes on a timer: the vendor endpoints are undocumented and
// rate-limited, and polling every account of every provider is how a vault
// gets throttled (cc-switch refreshes only the enabled provider).
func ActiveTargets(providers []catalog.Provider) []RefreshTarget {
	out := []RefreshTarget{}
	for _, p := range providers {
		if !p.SignedIn {
			continue
		}
		if len(p.Accounts) == 0 {
			if catalog.QuotaKind(p.ID, p.AuthType) != "" {
				out = append(out, RefreshTarget{Provider: p.ID})
			}
			continue
		}
		for _, a := range p.Accounts {
			if !a.Active || a.Paused || a.QuotaKind == "" || a.QuotaKind != a.Type {
				continue
			}
			out = append(out, RefreshTarget{Provider: p.ID, AccountID: a.ID})
		}
	}
	return out
}

// DefaultRefresh is how often the active slot of each provider refreshes.
// Zero disables the loop entirely.
const DefaultRefresh = 5 * time.Minute

// Refresh fetches the given rows in order, one at a time. Sequential on
// purpose: eight parallel calls to eight vendors on a laptop waking from
// sleep is a burst none of these undocumented endpoints expects.
func (c *Client) Refresh(ctx context.Context, targets []RefreshTarget) int {
	n := 0
	for _, t := range targets {
		if ctx.Err() != nil {
			return n
		}
		c.FetchAccount(ctx, t.Provider, t.AccountID)
		n++
	}
	return n
}
