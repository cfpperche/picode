package server

import (
	"sync"
	"time"
)

// TerminalsCache is the shared, short-lived snapshot behind GET /api/terminals
// (ADR-0048: one computation for the whole fleet, not one per client). The
// Agent CLIs page refetches the list on every terminal.* feed event; one list
// costs up to a dozen tmux and git subprocesses per terminal, so concurrent
// refetches share a single computation (singleflight) and a snapshot younger
// than the TTL answers without spawning anything. Mutating terminal handlers
// call Invalidate so a fresh write is never hidden by the snapshot.
//
// The zero value is ready to use. A nil *TerminalsCache in Deps means "no
// cache": every request computes directly (tests, minimal embeddings).
type TerminalsCache struct {
	mu    sync.Mutex
	term  []map[string]any
	at    time.Time
	fly   *terminalFlight
	clock func() time.Time // test seam
}

// terminalFlight is one in-flight computation. Late joiners block on wg and
// read the same result.
type terminalFlight struct {
	wg   sync.WaitGroup
	term []map[string]any
	err  error
}

// DefaultTTL is how long a computed snapshot answers without recompute. The
// CLIs page refreshes on feed events; state flip-flops arriving up to a
// second late are invisible next to the subprocess cost of computing again.
const DefaultTTL = time.Second

func (c *TerminalsCache) now() time.Time {
	if c.clock != nil {
		return c.clock()
	}
	return time.Now()
}

// View serves the cached snapshot when fresh enough, joins an in-flight
// computation, or leads a new one. compute runs at most once per round and
// its error is shared by everyone waiting on that round; a failed round is
// not cached, so the next request computes again.
func (c *TerminalsCache) View(ttl time.Duration, compute func() ([]map[string]any, error)) ([]map[string]any, error) {
	c.mu.Lock()
	if c.term != nil && c.now().Sub(c.at) < ttl {
		term := c.term
		c.mu.Unlock()
		return term, nil
	}
	if c.fly != nil {
		f := c.fly
		c.mu.Unlock()
		f.wg.Wait()
		return f.term, f.err
	}
	f := &terminalFlight{}
	f.wg.Add(1)
	c.fly = f
	c.mu.Unlock()

	term, err := compute()

	c.mu.Lock()
	c.fly = nil
	if err == nil {
		c.term, c.at = term, c.now()
	}
	c.mu.Unlock()

	f.term, f.err = term, err
	f.wg.Done()
	return term, err
}

// Invalidate drops the snapshot so the next View recomputes. Mutating
// terminal handlers call it after the store write succeeds.
func (c *TerminalsCache) Invalidate() {
	c.mu.Lock()
	c.term = nil
	c.mu.Unlock()
}
