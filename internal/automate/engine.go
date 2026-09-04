// Package automate is the automations scheduler (ADR-0045): a one-minute
// ticker in the daemon that fires due automations through a Runner. It
// owns *when*; the Runner (internal/server) owns *how* a run happens.
// Modelled on internal/backup/loop.go — no machine crontab, no session-
// scoped timer: the schedule lives in SQLite and survives restarts.
package automate

import (
	"context"
	"hash/fnv"
	"log"
	"sync"
	"time"

	"github.com/cfpperche/picode/internal/cron"
	"github.com/cfpperche/picode/internal/session"
	"github.com/cfpperche/picode/internal/store"
)

// Runner starts (or refuses) one invocation. Skips and failures are the
// Runner's to record; the Engine only decides that a slot is due.
type Runner interface {
	Fire(a store.Automation, trigger, payload string) (store.Run, error)
}

// Engine ticks and fires. Now is injectable for tests.
type Engine struct {
	Store  *store.Store
	Runner Runner
	Now    func() time.Time

	mu sync.Mutex
}

func (e *Engine) now() time.Time {
	if e.Now != nil {
		return e.Now()
	}
	return time.Now()
}

// Loop reconciles once, then ticks every minute until ctx ends.
func (e *Engine) Loop(ctx context.Context) {
	e.Reconcile()
	t := time.NewTicker(time.Minute)
	defer t.Stop()
	e.Tick()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			e.Tick()
		}
	}
}

// Reconcile closes runs the previous process left running: nothing is
// watching them any more (a deploy restarts the daemon), so the honest
// status is failed with the reason on the row.
func (e *Engine) Reconcile() {
	if e.Store == nil {
		return
	}
	stale, err := e.Store.FailStaleRuns("daemon restarted", SessionCost)
	if err != nil {
		log.Printf("automate: reconcile: %v", err)
		return
	}
	for _, r := range stale {
		a, err := e.Store.GetAutomation(r.AutomationID)
		if err != nil {
			continue
		}
		_, _ = e.Store.CreateInboxItem(store.InboxItemParams{
			Kind: store.InboxFYI, SourceKind: store.InboxFromAutomation, SourceID: a.ID,
			WorkspaceID: a.WorkspaceID, Reason: "daemon restarted",
			Title: a.Name + " was interrupted",
			Body:  "PiCode restarted while this run was in progress. The session file keeps what the agent did; the next scheduled run starts fresh.",
		})
	}
}

// SessionCost prices a run from its pi session file; 0 when unreadable.
func SessionCost(path string) float64 {
	if path == "" {
		return 0
	}
	s, err := session.Summarize(path)
	if err != nil {
		return 0
	}
	return s.Cost
}

// Tick evaluates every enabled scheduled automation once.
func (e *Engine) Tick() {
	if e.Store == nil || e.Runner == nil {
		return
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	items, err := e.Store.ListAutomations()
	if err != nil {
		log.Printf("automate: list: %v", err)
		return
	}
	now := e.now().Truncate(time.Minute)
	for _, a := range items {
		if !a.Enabled || a.Cron == nil {
			continue
		}
		sched, err := cron.Parse(*a.Cron)
		if err != nil {
			continue
		}
		slot, trigger, ok := Due(sched, a, now)
		if !ok {
			continue
		}
		if err := e.Store.TouchAutomationFired(a.ID, slot); err != nil {
			log.Printf("automate: touch %s: %v", a.ID, err)
			continue
		}
		if _, err := e.Runner.Fire(a, trigger, ""); err != nil {
			log.Printf("automate: fire %s: %v", a.ID, err)
		}
	}
}

// Jitter is the deterministic per-automation delay added to every slot,
// so a fleet of daily-at-09:00 automations does not hit the provider in
// the same second. Bounded by half the interval and by 30 minutes; zero
// for schedules tighter than two minutes.
func Jitter(id string, interval time.Duration) time.Duration {
	limit := interval / 2
	if limit > 30*time.Minute {
		limit = 30 * time.Minute
	}
	limitMin := int64(limit / time.Minute)
	if limitMin < 1 {
		return 0
	}
	h := fnv.New32a()
	_, _ = h.Write([]byte(id))
	return time.Duration(int64(h.Sum32())%limitMin) * time.Minute
}

// NextFire is when automation id fires next after now: the first slot s
// with s+jitter still ahead, plus that jitter — the same arithmetic Due
// uses, so the list never promises a time the engine will not keep.
func NextFire(sched cron.Schedule, id string, now time.Time) (time.Time, bool) {
	jitter := Jitter(id, sched.Interval(now))
	s, ok := sched.Next(now.Truncate(time.Minute).Add(-jitter))
	if !ok {
		return time.Time{}, false
	}
	return s.Add(jitter), true
}

// Due answers whether automation a should fire at now (minute-truncated).
// It returns the slot being honoured and its trigger:
//   - schedule: now-jitter matches the cron and that slot has not fired;
//   - catch-up: the slot is not due now, but at least one slot between
//     last_fired_at and now was missed (daemon down) — fired once, and the
//     latest missed slot is what gets stamped, so the backlog collapses.
//
// A never-fired automation does not catch up: creating one at 10:00 with
// "daily at 09:00" waits for tomorrow, as every benchmark does.
func Due(sched cron.Schedule, a store.Automation, now time.Time) (slot time.Time, trigger string, ok bool) {
	jitter := Jitter(a.ID, sched.Interval(now))
	var last time.Time
	if a.LastFiredAt != nil {
		if t, err := time.Parse(time.RFC3339Nano, *a.LastFiredAt); err == nil {
			last = t.In(now.Location())
		}
	}
	candidate := now.Add(-jitter)
	if sched.Matches(candidate) && (last.IsZero() || candidate.After(last)) {
		return candidate, store.TriggerSchedule, true
	}
	if last.IsZero() {
		return time.Time{}, "", false
	}
	// Missed slots: every match strictly after last whose jittered fire
	// time already passed.
	var missed time.Time
	next, more := sched.Next(last)
	for i := 0; more && i < 100_000 && !next.Add(jitter).After(now); i++ {
		missed = next
		next, more = sched.Next(next)
	}
	if missed.IsZero() {
		return time.Time{}, "", false
	}
	return missed, store.TriggerCatchUp, true
}
