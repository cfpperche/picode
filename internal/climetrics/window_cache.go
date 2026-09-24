package climetrics

import (
	"sync"
	"time"
)

// WindowCache keeps each meter's last Window per window key, so a poll after
// one CLI wrote a line re-meters that CLI alone instead of all nine.
// Measured 2026-09-24 on the owner's machine: a warm 7-day poll spent
// 300–500 ms re-metering every CLI (OpenCode's database ~270 ms of it, on
// every poll, unchanged), while one dirty CLI costs ~20 ms on its own.
//
// An entry is reused only when the meter's Fingerprint and the whole
// request (bounds, bucketing, location, price table) still match. An empty
// fingerprint never hits, the same rule the fleet cache keeps.
type WindowCache struct {
	mu      sync.Mutex
	entries map[string]windowEntry
}

type windowEntry struct {
	fp  string
	req requestID
	w   Window
}

// requestID is what a Window depends on besides the meter's own files.
type requestID struct {
	from, to, prior time.Time
	loc             string
	hourly          bool
	prices          string
}

func idOf(req Request) requestID {
	id := requestID{from: req.From, to: req.To, prior: req.PriorFrom, loc: req.LocOf().String(), hourly: req.Hourly}
	if req.Prices != nil {
		id.prices = req.Prices.Version()
	}
	return id
}

// AggregateCached is Aggregate with each meter's Window reused from c while
// its fingerprint holds. key names the window (the server's root|range).
// A request that filters by folder (KeepCwd) or sets per-CLI billing is
// not cacheable here and goes straight to Aggregate.
func AggregateCached(req Request, meters []Meter, c *WindowCache, key string) FleetStats {
	if c == nil || req.KeepCwd != nil || len(req.Billing) > 0 {
		return Aggregate(req, meters)
	}
	id := idOf(req)
	windows := make([]Window, len(meters))
	var wg sync.WaitGroup
	for i, m := range meters {
		fp := m.Fingerprint()
		if w, ok := c.get(key+"|"+m.CLI(), fp, id); ok {
			windows[i] = w
			continue
		}
		wg.Add(1)
		go func(i int, m Meter, fp string) {
			defer wg.Done()
			w, err := m.Meter(req)
			if err != nil {
				// A failure is not cached: the next poll tries again.
				windows[i] = Window{CLI: m.CLI(), Coverage: unavailableRow(m, req.BillingFor(m.CLI()), err.Error())}
				return
			}
			w.CLI = m.CLI()
			windows[i] = w
			c.put(key+"|"+m.CLI(), fp, id, w)
		}(i, m, fp)
	}
	wg.Wait()
	return merge(req, windows)
}

func (c *WindowCache) get(key, fp string, id requestID) (Window, bool) {
	if fp == "" {
		return Window{}, false
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	e, ok := c.entries[key]
	if !ok || e.fp != fp || e.req != id {
		return Window{}, false
	}
	return e.w, true
}

func (c *WindowCache) put(key, fp string, id requestID, w Window) {
	if fp == "" {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.entries == nil {
		c.entries = map[string]windowEntry{}
	}
	c.entries[key] = windowEntry{fp: fp, req: id, w: w}
}

// Reset drops every entry (tests; a changed session root).
func (c *WindowCache) Reset() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries = nil
}
