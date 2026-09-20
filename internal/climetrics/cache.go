package climetrics

import (
	"os"
	"strings"
	"sync"
	"time"
)

// parsed is everything one session store yields that a window can be cut
// from: the per-message entries, plus the file-lifetime figures that have
// to be prorated against whichever window asks.
//
// It is deliberately window-independent. Caching a *windowed* result would
// have to be thrown away whenever the range changed; caching the parse
// survives every range, and the windowing that remains is arithmetic over
// a slice.
type parsed struct {
	key         string // the session identity every entry in this file carries
	ents        []cliEntry
	units       int64 // sum of ents' weights, the denominator for proration
	byPresence  bool  // weigh each entry as 1 rather than its tokens (Grok)
	impact      Impact
	timing      Timing
	limits      []LimitWindow
	limitAt     time.Time
	compactions []compaction // pi's markers; counted when they fall in window
	priced      bool         // the file carried a cost record of its own
	underpriced bool         // …and that record admits a model it could not price
}

// parseCache memoises parses by (path, size, mtime).
//
// This is what makes the dashboard's 60-second poll affordable. Measured on
// this machine before it existed: a cold 7-day window over all nine CLIs took
// 5.8s and `all` took 11.4s, and because agents write constantly the
// fingerprint changed on almost every poll — so almost every poll paid it.
// With the cache a poll re-parses only the handful of files that actually
// moved.
//
// Bounded by total cached entries rather than file count, because session
// files differ by four orders of magnitude in size; eviction drops the
// least recently used file whole, never half a file.
type parseCache struct {
	mu      sync.Mutex
	m       map[string]cacheEntry
	entries int
	tick    int64
	max     int
}

type cacheEntry struct {
	size, mtime int64
	used        int64
	val         *parsed
}

// maxCachedEntries caps the cache at roughly 60–80 MB of parsed messages —
// a cliEntry carries strings and a tools slice, so the first estimate of
// 45 MB was optimistic. The
// common windows (today / 7d / 30d) hold their working set well inside it;
// `all` may evict, and pays a full parse when it does — a deliberate trade,
// since it is a rare deliberate click and the fingerprint cache holds its
// answer afterwards.
const maxCachedEntries = 400_000

var files = &parseCache{max: maxCachedEntries}

// get returns the parse for path if the file has not changed since.
func (c *parseCache) get(path string, size, mtime int64) (*parsed, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e, ok := c.m[path]
	if !ok || e.size != size || e.mtime != mtime {
		return nil, false
	}
	c.tick++
	e.used = c.tick
	c.m[path] = e
	return e.val, true
}

func (c *parseCache) put(path string, size, mtime int64, p *parsed) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.m == nil {
		c.m = map[string]cacheEntry{}
	}
	if old, ok := c.m[path]; ok {
		c.entries -= len(old.val.ents)
	}
	c.tick++
	c.m[path] = cacheEntry{size: size, mtime: mtime, used: c.tick, val: p}
	c.entries += len(p.ents)
	c.evict()
}

// evict drops least-recently-used files until the entry budget is met. It
// runs under the caller's lock.
func (c *parseCache) evict() {
	for c.entries > c.max && len(c.m) > 1 {
		var oldestKey string
		var oldest int64 = 1<<63 - 1
		for k, e := range c.m {
			if e.used < oldest {
				oldest, oldestKey = e.used, k
			}
		}
		if oldestKey == "" {
			return
		}
		c.entries -= len(c.m[oldestKey].val.ents)
		delete(c.m, oldestKey)
	}
}

// reset empties the cache. Tests call it so one fixture cannot answer for
// another; nothing in production does.
func (c *parseCache) reset() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.m = nil
	c.entries = 0
}

// cachedParse returns path's parse, calling parse only when the file has
// changed. stat failures fall through to a direct parse rather than an
// error: an unreadable file is one CLI's problem, not the dashboard's.
func cachedParse(path string, parse func(string) *parsed) *parsed {
	info, err := os.Stat(path)
	if err != nil {
		return parse(path)
	}
	size, mtime := info.Size(), info.ModTime().UnixNano()
	if p, ok := files.get(path, size, mtime); ok {
		return p
	}
	p := parse(path)
	if p == nil {
		p = &parsed{}
	}
	files.put(path, size, mtime, p)
	return p
}

// replay files a parse into a window. Proration of the file-lifetime
// figures (lines changed, durations) happens here, against this window's
// share of the file's tokens, so one parse serves every range.
func replay(p *parsed, acc *cliAcc, req Request) (contributed bool) {
	if p == nil {
		return false
	}
	// Impact proration counts only entries inside the window. Counting by
	// time alone was not always enough — the retired scope mode also had to
	// gate by folder (ADR-0127) — and the window check alone is what remains.
	var inWindow int64
	for i := range p.ents {
		e := &p.ents[i]
		if inCurrentWindow(req, e.at) {
			inWindow += p.weight(e)
		}
		before := acc.current.Messages
		acc.add(*e)
		if acc.current.Messages != before {
			contributed = true
		}
	}
	for _, c := range p.compactions {
		if inCurrentWindow(req, c.at) {
			acc.turns.Compactions++
		}
	}
	if p.units > 0 && inWindow > 0 {
		share := float64(inWindow) / float64(p.units)
		acc.impact.LinesAdded += scale(p.impact.LinesAdded, share)
		acc.impact.LinesRemoved += scale(p.impact.LinesRemoved, share)
		acc.timing.APIMs += scale(p.timing.APIMs, share)
		acc.timing.ToolMs += scale(p.timing.ToolMs, share)
		acc.timing.SessionMs += scale(p.timing.SessionMs, share)
	}
	return contributed
}

// compaction is one pi compaction marker and the folder current when it
// was written, so a scoped window counts only its own.
type compaction struct {
	at  time.Time
	cwd string
}

// weight is what proration counts for one entry. Token units are the
// default: cost, lines and durations scale with how much a file said. Grok's
// events timeline is the exception — it records turns, tools and durations
// for sessions whose usage.json (the only place tokens live) may not exist
// yet, so its entries weigh one each and a window's share is its share of
// turns. Weighing them by tokens would drop the durations of every session
// Grok has not written a usage record for, which on this machine is all but
// three of them.
func (p *parsed) weight(e *cliEntry) int64 {
	if p.byPresence {
		return 1
	}
	return e.toks.Input + e.toks.Output + e.toks.CacheRead + e.toks.CacheWrite
}

// statKey is a cache key that changes whenever any of paths does, absence
// included: a store that appears later must not be answered by the empty
// parse cached while it was missing.
func statKey(paths ...string) string {
	var b strings.Builder
	for _, p := range paths {
		b.WriteString(p)
		info, err := os.Stat(p)
		if err != nil {
			b.WriteString("=absent;")
			continue
		}
		b.WriteByte('=')
		b.WriteString(itoa64(info.Size()))
		b.WriteByte(':')
		b.WriteString(itoa64(info.ModTime().UnixNano()))
		b.WriteByte(';')
	}
	return b.String()
}

// cachedParseKeyed is cachedParse for a parse that spans more than one file.
// The caller builds key with statKey, so the whole multi-file state is the
// cache identity and nothing here stats anything.
func cachedParseKeyed(key string, parse func() *parsed) *parsed {
	if key == "" {
		return &parsed{}
	}
	if p, ok := files.get(key, 0, 0); ok {
		return p
	}
	p := parse()
	if p == nil {
		p = &parsed{}
	}
	files.put(key, 0, 0, p)
	return p
}

func inCurrentWindow(req Request, t time.Time) bool {
	if !t.Before(req.To) {
		return false
	}
	return req.From.IsZero() || !t.Before(req.From)
}

func scale(v int64, share float64) int64 { return int64(float64(v) * share) }
