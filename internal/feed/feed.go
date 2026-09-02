// Package feed is the change feed (ADR-0048): every committed store event
// fans out to subscribers, with replay from the events table for a
// subscriber that reconnects with a cursor. Ephemeral notices (presence,
// a waiting agent) ride the same channel with ID 0 and are never replayed.
package feed

import (
	"errors"
	"sync"
	"time"

	"github.com/cfpperche/picode/internal/store"
)

// ErrReset: the cursor is older than what the log still holds; the client
// must refresh everything and start from Latest.
var ErrReset = errors.New("feed: cursor older than retention")

// DefaultRetention bounds the events table; the sweep prunes past it.
const DefaultRetention = 7 * 24 * time.Hour

// Feed owns the subscriber set. Publish is safe from any goroutine and
// never blocks on a slow subscriber (its channel drops the event; the
// subscriber notices the gap on its next cursor and asks for a replay).
type Feed struct {
	Store *store.Store

	mu        sync.Mutex
	subs      map[int]chan store.Event
	next      int
	listeners []func(store.Event)
}

// Publish announces one event. It is the store's OnEvent.
func (f *Feed) Publish(ev store.Event) {
	f.mu.Lock()
	for _, ch := range f.subs {
		select {
		case ch <- ev:
		default: // slow subscriber: drop, replay will cover it
		}
	}
	ls := f.listeners
	f.mu.Unlock()
	for _, fn := range ls {
		fn(ev)
	}
}

// Ephemeral publishes a notice that is not in the log (ID 0).
func (f *Feed) Ephemeral(eventType string, data any) {
	body, err := marshal(data)
	if err != nil {
		return
	}
	f.Publish(store.Event{Type: eventType, Data: body, CreatedAt: time.Now().UTC().Format(time.RFC3339Nano)})
}

// Listen registers an in-process consumer called synchronously on every
// event (the push notifier). Listeners must return fast.
func (f *Feed) Listen(fn func(store.Event)) {
	f.mu.Lock()
	f.listeners = append(f.listeners, fn)
	f.mu.Unlock()
}

// Latest is the newest durable id (0 when the log is empty or unavailable).
func (f *Feed) Latest() int64 {
	if f.Store == nil {
		return 0
	}
	id, _ := f.Store.LatestEventID()
	return id
}

// Subscribe returns the events after afterID (replay), a live channel, and
// an unsubscribe. afterID <= 0 means live only. The replay is read under
// the same lock that registers the channel, so nothing falls between.
func (f *Feed) Subscribe(afterID int64) (replay []store.Event, live <-chan store.Event, unsub func(), err error) {
	ch := make(chan store.Event, 256)
	f.mu.Lock()
	defer f.mu.Unlock()
	if afterID > 0 && f.Store != nil {
		oldest, e := f.Store.OldestEventID()
		if e != nil {
			return nil, nil, nil, e
		}
		latest, _ := f.Store.LatestEventID()
		// A cursor below the oldest kept row minus one cannot be
		// replayed faithfully; equal to latest means nothing to replay.
		if oldest > 0 && afterID < oldest-1 {
			return nil, nil, nil, ErrReset
		}
		if afterID < latest {
			replay, e = f.Store.ListEventsSince(afterID, 5000)
			if e != nil {
				return nil, nil, nil, e
			}
		}
	}
	if f.subs == nil {
		f.subs = map[int]chan store.Event{}
	}
	id := f.next
	f.next++
	f.subs[id] = ch
	return replay, ch, func() {
		f.mu.Lock()
		delete(f.subs, id)
		f.mu.Unlock()
	}, nil
}

// Subscribers is the live subscriber count (tests, /api/system).
func (f *Feed) Subscribers() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.subs)
}
