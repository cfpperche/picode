package remind

import (
	"context"
	"log"
	"sync"
	"time"
)

// Store is what the engine needs from internal/store, in plain types so
// the store can import this package's rules without a cycle.
type Store interface {
	// DueReminderIDs lists enabled reminders whose next_at <= now.
	DueReminderIDs(now time.Time) ([]string, error)
	// FireReminder files (or re-raises) the Inbox item for one reminder,
	// announces pin.reminded and advances next_at — one transaction.
	FireReminder(id string, now time.Time) error
	// WakeSnoozedReminders re-raises reminder items whose snooze ended.
	WakeSnoozedReminders(now time.Time) (int, error)
}

// Engine ticks every minute, like internal/automate: the schedule lives in
// SQLite and survives restarts; the first tick after boot is the catch-up.
type Engine struct {
	Store Store
	Now   func() time.Time

	mu sync.Mutex
}

func (e *Engine) now() time.Time {
	if e.Now != nil {
		return e.Now()
	}
	return time.Now()
}

// Loop ticks once at start, then every minute until ctx ends.
func (e *Engine) Loop(ctx context.Context) {
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

// Tick fires every due reminder once and wakes ended snoozes.
func (e *Engine) Tick() {
	if e.Store == nil {
		return
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	now := e.now()
	ids, err := e.Store.DueReminderIDs(now)
	if err != nil {
		log.Printf("remind: due: %v", err)
		return
	}
	for _, id := range ids {
		if err := e.Store.FireReminder(id, now); err != nil {
			log.Printf("remind: fire %s: %v", id, err)
		}
	}
	if _, err := e.Store.WakeSnoozedReminders(now); err != nil {
		log.Printf("remind: wake: %v", err)
	}
}
