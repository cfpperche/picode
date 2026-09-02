package gateway

import (
	"sync"
	"time"
)

// limiter is the brute-force gate for /pair and /-/auth/*: n events per
// window per key, then a lockout. Same shape as internal/auth's.
type limiter struct {
	mu      sync.Mutex
	n       int
	window  time.Duration
	lockout time.Duration
	seen    map[string]*bucket
	now     func() time.Time
}

type bucket struct {
	count int
	start time.Time
	until time.Time
}

func newLimiter(n int, window, lockout time.Duration) *limiter {
	return &limiter{n: n, window: window, lockout: lockout, seen: map[string]*bucket{}, now: time.Now}
}

// allow records one event for key and says whether it may proceed.
func (l *limiter) allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	b := l.seen[key]
	if b != nil && !b.until.IsZero() {
		if now.Before(b.until) {
			return false
		}
		delete(l.seen, key)
		b = nil
	}
	if b == nil || now.Sub(b.start) > l.window {
		b = &bucket{start: now}
		l.seen[key] = b
	}
	b.count++
	if b.count > l.n {
		b.until = now.Add(l.lockout)
		return false
	}
	return true
}
