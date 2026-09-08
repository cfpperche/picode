// Package remind holds the pin reminder rules (ADR-0100): what a cadence
// means, when it fires next, and what a missed slot does. Pure functions
// over time; the store persists rows and the engine ticks. Nothing here
// imports the store, so the store can call in.
package remind

import (
	"errors"
	"fmt"
	"strings"
	"time"
	_ "time/tzdata" // named zones on hosts without a tz database (Windows)

	"github.com/cfpperche/picode/internal/cron"
)

const (
	KindOnce     = "once"     // one instant, then finished
	KindInterval = "interval" // every N minutes, from the schedule or from the close
	KindCron     = "cron"     // a calendar rule in a named zone (internal/cron)

	AnchorSchedule   = "schedule"   // next = slot + interval, whatever the person did
	AnchorCompletion = "completion" // next = close time + interval; nothing owed until then

	MinInterval = 5 * time.Minute
	MaxInterval = 366 * 24 * time.Hour
	// A slot older than this at fire time was missed (daemon down, backlog):
	// it fires once and says so. A normal tick lands within a minute.
	catchUpAfter = 90 * time.Second
)

var ErrInvalid = errors.New("invalid reminder")

func invalid(format string, args ...any) error {
	return fmt.Errorf("%w: %s", ErrInvalid, fmt.Sprintf(format, args...))
}

// Rule is one reminder's cadence.
type Rule struct {
	Kind     string
	At       time.Time     // once: the instant; interval: the first fire (optional)
	Interval time.Duration // interval
	Cron     string        // cron
	Anchor   string        // interval only
	TZ       string        // IANA zone; wall-clock rules are evaluated in it
}

// Location resolves the zone; an unknown zone is UTC, but Validate refuses it.
func (r Rule) Location() *time.Location {
	if loc, err := time.LoadLocation(r.TZ); err == nil && r.TZ != "" {
		return loc
	}
	return time.UTC
}

// Validate refuses what the picker must never produce.
func (r Rule) Validate(now time.Time) error {
	if r.TZ == "" {
		return invalid("time zone is required")
	}
	if _, err := time.LoadLocation(r.TZ); err != nil {
		return invalid("unknown time zone %q", r.TZ)
	}
	switch r.Kind {
	case KindOnce:
		if r.At.IsZero() {
			return invalid("a date and time are required")
		}
		if !r.At.After(now.Add(-time.Minute)) {
			return invalid("that time has already passed")
		}
	case KindInterval:
		if r.Interval < MinInterval {
			return invalid("the interval is at least %d minutes", int(MinInterval/time.Minute))
		}
		if !r.At.IsZero() && !r.At.After(now.Add(-time.Minute)) {
			return invalid("the first time has already passed")
		}
		if r.Interval > MaxInterval {
			return invalid("the interval is at most a year")
		}
		if r.Anchor != AnchorSchedule && r.Anchor != AnchorCompletion {
			return invalid("anchor must be schedule or completion")
		}
	case KindCron:
		if _, err := cron.Parse(r.Cron); err != nil {
			return invalid("%v", err)
		}
	default:
		return invalid("kind must be once, interval or cron")
	}
	return nil
}

// First is the first instant owed after the rule is set.
func (r Rule) First(now time.Time) (time.Time, bool) {
	switch r.Kind {
	case KindOnce:
		return r.At.UTC(), true
	case KindInterval:
		// "Every 3 days at 09:00": the person names the first fire and
		// the interval counts from there (drifting across DST, as a
		// duration does — the picker says so).
		if !r.At.IsZero() {
			return r.At.UTC(), true
		}
		return now.Add(r.Interval).UTC(), true
	case KindCron:
		return r.nextCron(now)
	}
	return time.Time{}, false
}

// AfterFire is what the row owes after the slot fired at now. A backlog
// collapses: an interval whose next step is already past restarts from
// now, a cron takes the next match after now, a once is finished, and a
// completion-anchored interval owes nothing until the item is closed.
func (r Rule) AfterFire(slot, now time.Time) (time.Time, bool) {
	switch r.Kind {
	case KindInterval:
		if r.Anchor == AnchorCompletion {
			return time.Time{}, false
		}
		next := slot.Add(r.Interval)
		if !next.After(now) {
			next = now.Add(r.Interval)
		}
		return next.UTC(), true
	case KindCron:
		return r.nextCron(now)
	}
	return time.Time{}, false
}

// AfterClose is the completion anchor: the person closed the item at
// closedAt, the next fire is one interval later. Every other rule is
// unaffected by closing.
func (r Rule) AfterClose(closedAt time.Time) (time.Time, bool) {
	if r.Kind == KindInterval && r.Anchor == AnchorCompletion {
		return closedAt.Add(r.Interval).UTC(), true
	}
	return time.Time{}, false
}

// CatchUp reports whether a slot fired late enough to have been missed.
func CatchUp(slot, now time.Time) bool {
	return now.Sub(slot) >= catchUpAfter
}

func (r Rule) nextCron(from time.Time) (time.Time, bool) {
	sched, err := cron.Parse(r.Cron)
	if err != nil {
		return time.Time{}, false
	}
	t, ok := sched.Next(from.In(r.Location()))
	if !ok {
		return time.Time{}, false
	}
	return t.UTC(), true
}

// Label is the cadence in words, for the Inbox row's reason and the
// sidebar line: "at Tue 09:00", "every day at 09:00", "every 3 h after
// you close it".
func (r Rule) Label() string {
	loc := r.Location()
	switch r.Kind {
	case KindOnce:
		return "at " + r.At.In(loc).Format("Mon 2 Jan 15:04")
	case KindInterval:
		s := "every " + humanDuration(r.Interval)
		if r.Interval%(24*time.Hour) == 0 && !r.At.IsZero() && r.Anchor != AnchorCompletion {
			s += " at " + r.At.In(loc).Format("15:04")
		}
		if r.Anchor == AnchorCompletion {
			s += " after you close it"
		}
		return s
	case KindCron:
		return cronLabel(r.Cron)
	}
	return ""
}

func humanDuration(d time.Duration) string {
	switch {
	case d%(24*time.Hour) == 0:
		n := int(d / (24 * time.Hour))
		if n == 1 {
			return "day"
		}
		return fmt.Sprintf("%d days", n)
	case d%time.Hour == 0:
		n := int(d / time.Hour)
		if n == 1 {
			return "hour"
		}
		return fmt.Sprintf("%d h", n)
	}
	return fmt.Sprintf("%d min", int(d/time.Minute))
}

// cronLabel names the presets the picker writes; anything else is shown
// as the expression it is.
func cronLabel(expr string) string {
	f := strings.Fields(expr)
	if len(f) != 5 {
		return expr
	}
	hhmm := func() string {
		var mm, hh int
		if _, err := fmt.Sscanf(f[0]+" "+f[1], "%d %d", &mm, &hh); err != nil {
			return ""
		}
		return fmt.Sprintf("%02d:%02d", hh, mm)
	}
	t := hhmm()
	switch {
	case t != "" && f[2] == "*" && f[3] == "*" && f[4] == "*":
		return "every day at " + t
	case t != "" && f[2] == "*" && f[3] == "*" && f[4] == "1-5":
		return "weekdays at " + t
	case t != "" && f[2] == "*" && f[3] == "*":
		if d, ok := dayName(f[4]); ok {
			return "every " + d + " at " + t
		}
	case f[0] != "*" && f[1] == "*" && f[2] == "*" && f[3] == "*" && f[4] == "*":
		if !strings.ContainsAny(f[0], ",-/") {
			return "every hour at :" + fmt.Sprintf("%02s", f[0])
		}
	}
	return "cron " + expr
}

func dayName(s string) (string, bool) {
	names := []string{"Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday", "Sunday"}
	var n int
	if _, err := fmt.Sscanf(s, "%d", &n); err != nil || n < 0 || n > 7 || strings.ContainsAny(s, ",-/") {
		return "", false
	}
	return names[n], true
}
