// Package pricing estimates what a turn would cost at list price, for the
// turns an agent CLI left unpriced (ADR-0185). Rates come from LiteLLM's
// model_prices_and_context_window.json — the table ccusage and t3code price
// the same stores against — fetched off the request path and kept in the
// data dir, so a dashboard poll never waits on the network.
//
// An estimate is never a CLI's own figure: callers price only what the CLI
// did not, and carry the result as "estimated" beside the reported cost.
package pricing

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
	"sync/atomic"
)

// Rate is USD per token for each kind of token a transcript separates.
type Rate struct {
	Input      float64
	Output     float64
	CacheRead  float64
	CacheWrite float64
	// CacheWrite1h is the rate for a cache entry written to live an hour,
	// which Anthropic bills at 1.6x the 5-minute one. Claude Code wrote 72%
	// of its cache tokens that way over 30 days here (2026-09-22).
	CacheWrite1h float64
}

// Table is a parsed price table. The zero value and nil both price nothing.
type Table struct {
	rates   map[string]Rate
	version string
}

// entry is the subset of a LiteLLM row this package reads.
type entry struct {
	Input      *float64 `json:"input_cost_per_token"`
	Output     *float64 `json:"output_cost_per_token"`
	CacheRead  *float64 `json:"cache_read_input_token_cost"`
	CacheWrite *float64 `json:"cache_creation_input_token_cost"`
	Write1h    *float64 `json:"cache_creation_input_token_cost_above_1hr"`
}

// Parse reads LiteLLM's document. A row without both an input and an
// output rate is dropped: a half-priced model would under-report silently,
// which is worse than reporting it unpriced. A row that omits its cache
// rates prices cached tokens as plain input rather than as free.
//
// Keys are kept whole ("azure_ai/grok-4.6"); a bare name ("grok-4.6") is
// added as an alias only when no row claims it directly and every row that
// ends in it agrees on the rate — two providers selling the same model at
// two prices leave the bare name unpriced rather than picking one. Rows
// nested one level deeper ("azure/eu/gpt-5.2-codex", a +10% regional
// surcharge; "openrouter/openai/…", a reseller) do not vote: counting them
// left every Codex model without a bare row unpriced, 9,005 turns here.
func Parse(raw []byte) (*Table, error) {
	var doc map[string]json.RawMessage
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil, err
	}
	t := &Table{rates: make(map[string]Rate, len(doc))}
	for name, blob := range doc {
		var e entry
		if json.Unmarshal(blob, &e) != nil || e.Input == nil || e.Output == nil {
			continue
		}
		key := normalize(name)
		if key == "" {
			continue
		}
		r := Rate{Input: *e.Input, Output: *e.Output, CacheRead: *e.Input, CacheWrite: *e.Input}
		if e.CacheRead != nil {
			r.CacheRead = *e.CacheRead
		}
		if e.CacheWrite != nil {
			r.CacheWrite = *e.CacheWrite
		}
		r.CacheWrite1h = r.CacheWrite
		if e.Write1h != nil {
			r.CacheWrite1h = *e.Write1h
		}
		t.rates[key] = r
	}
	alias := map[string]*Rate{} // nil marks a bare name claimed at two rates
	conflict := map[string]bool{}
	for key, r := range t.rates {
		bare := bareName(key)
		if bare == key || strings.Count(key, "/") > 1 {
			continue
		}
		if _, direct := t.rates[bare]; direct || conflict[bare] {
			continue
		}
		if held, ok := alias[bare]; ok {
			if *held != r {
				conflict[bare] = true
				delete(alias, bare)
			}
			continue
		}
		rr := r
		alias[bare] = &rr
	}
	for bare, r := range alias {
		t.rates[bare] = *r
	}
	sum := sha256.Sum256(raw)
	t.version = hex.EncodeToString(sum[:8])
	return t, nil
}

// Version identifies the table's content. It joins the stats cache key, so
// a new table recomputes every window instead of serving old estimates.
func (t *Table) Version() string {
	if t == nil {
		return ""
	}
	return t.version
}

// Len is how many models the table prices, aliases included.
func (t *Table) Len() int {
	if t == nil {
		return 0
	}
	return len(t.rates)
}

// unpriceable are names never priced whatever the table says:
// "<synthetic>" marks a message the CLI generated itself and never billed,
// and a bare family name is ambiguous across generations — guessing one
// would be a price nobody could check.
var unpriceable = map[string]bool{
	"": true, "unknown": true, "<synthetic>": true, "synthetic": true,
	"opus": true, "sonnet": true, "haiku": true, "fable": true,
}

// Lookup is the rate for a model id as a CLI wrote it. The bracketed
// variant Claude Code appends for the 1M-context tier ("claude-opus-5[1m]")
// is dropped: the table prices the base tier, and so does every estimate.
func (t *Table) Lookup(model string) (Rate, bool) {
	if t == nil {
		return Rate{}, false
	}
	key := normalize(model)
	if i := strings.IndexByte(key, '['); i >= 0 {
		key = key[:i]
	}
	if unpriceable[bareName(key)] {
		return Rate{}, false
	}
	r, ok := t.rates[key]
	return r, ok
}

// Cost prices one turn's tokens. input is the uncached input only; cached
// tokens are priced at their own rates, and reasoning is already inside
// output, so it is never charged twice. cacheWrite1h is the part of
// cacheWrite written for an hour, priced at its own rate.
func (t *Table) Cost(model string, input, output, cacheRead, cacheWrite, cacheWrite1h int64) (float64, bool) {
	r, ok := t.Lookup(model)
	if !ok {
		return 0, false
	}
	cacheWrite1h = min(cacheWrite1h, cacheWrite)
	return float64(input)*r.Input + float64(output)*r.Output +
		float64(cacheRead)*r.CacheRead +
		float64(cacheWrite-cacheWrite1h)*r.CacheWrite + float64(cacheWrite1h)*r.CacheWrite1h, true
}

func normalize(s string) string { return strings.ToLower(strings.TrimSpace(s)) }

func bareName(key string) string {
	if i := strings.LastIndexByte(key, '/'); i >= 0 {
		return key[i+1:]
	}
	return key
}

var current atomic.Pointer[Table]

// Current is the table in use, or nil before any was loaded — which prices
// nothing, the honest answer when PiCode has never seen a table.
func Current() *Table { return current.Load() }

// Set installs a table (the loader, and tests).
func Set(t *Table) { current.Store(t) }
