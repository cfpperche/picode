package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/cfpperche/picode/internal/feed"
	"github.com/cfpperche/picode/internal/store"
)

// eventsHeartbeat keeps proxies and browsers from timing the stream out.
var eventsHeartbeat = 25 * time.Second

// handleEvents is the change feed over server-sent events (ADR-0048).
//
//	hello  {bootId, latest}   first frame; a new bootId means a new binary
//	<type> {Event}            durable events carry id: for Last-Event-ID
//	<type> {Event}            ephemeral events carry no id
//	reset  {}                 the cursor is older than retention: refresh all
//
// The cursor comes from Last-Event-ID (browser reconnect) or ?after=.
func handleEvents(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps.Feed == nil {
			writeErr(w, http.StatusServiceUnavailable, "change feed is not running")
			return
		}
		fl, ok := w.(http.Flusher)
		if !ok {
			writeErr(w, http.StatusInternalServerError, "streaming unsupported")
			return
		}
		after := cursorFrom(r)
		replay, live, unsub, err := deps.Feed.Subscribe(after)
		reset := errors.Is(err, feed.ErrReset)
		if err != nil && !reset {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		if reset {
			replay, live, unsub, err = deps.Feed.Subscribe(0)
			if err != nil {
				writeErr(w, http.StatusInternalServerError, err.Error())
				return
			}
		}
		defer unsub()

		h := w.Header()
		h.Set("Content-Type", "text/event-stream; charset=utf-8")
		h.Set("Cache-Control", "no-cache")
		h.Set("Connection", "keep-alive")
		h.Set("X-Accel-Buffering", "no")
		w.WriteHeader(http.StatusOK)

		write := func(id int64, typ string, data any) bool {
			body, err := json.Marshal(data)
			if err != nil {
				return true
			}
			var b strings.Builder
			if id > 0 {
				fmt.Fprintf(&b, "id: %d\n", id)
			}
			fmt.Fprintf(&b, "event: %s\ndata: %s\n\n", typ, body)
			if _, err := w.Write([]byte(b.String())); err != nil {
				return false
			}
			fl.Flush()
			return true
		}

		if _, err := fmt.Fprint(w, "retry: 2000\n\n"); err != nil {
			return
		}
		if !write(0, "hello", map[string]any{"bootId": bootID, "latest": deps.Feed.Latest()}) {
			return
		}
		if reset && !write(0, "reset", map[string]any{}) {
			return
		}
		for _, ev := range replay {
			if !write(ev.ID, ev.Type, ev) {
				return
			}
		}
		beat := time.NewTicker(eventsHeartbeat)
		defer beat.Stop()
		for {
			select {
			case <-r.Context().Done():
				return
			case ev := <-live:
				if !write(ev.ID, ev.Type, ev) {
					return
				}
			case <-beat.C:
				if _, err := fmt.Fprint(w, ": ping\n\n"); err != nil {
					return
				}
				fl.Flush()
			}
		}
	}
}

func cursorFrom(r *http.Request) int64 {
	raw := strings.TrimSpace(r.Header.Get("Last-Event-ID"))
	if raw == "" {
		raw = strings.TrimSpace(r.URL.Query().Get("after"))
	}
	n, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || n < 0 {
		return 0
	}
	return n
}

// eventFrom decodes a store.Event payload for tests and in-process consumers.
func eventFrom(line string) (store.Event, error) {
	var ev store.Event
	err := json.Unmarshal([]byte(line), &ev)
	return ev, err
}
