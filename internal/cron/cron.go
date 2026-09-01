// Package cron parses and evaluates 5-field cron expressions
// (minute hour day-of-month month day-of-week) with vixie-cron semantics:
// `*`, single values, `a-b` ranges, `*/n` and `a-b/n` steps, comma lists;
// Sunday is 0 or 7; when both day fields are constrained a date matches
// if either does. Names (`MON`, `JAN`) and `L`/`W`/`?` are not supported.
// Stdlib only (AGENTS.md #3) — the grammar is small enough not to earn a
// dependency, and the web mirrors it in web/src/lib/cron.js.
package cron

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Schedule is a parsed expression. The zero value matches nothing.
type Schedule struct {
	minute, hour, dom, month, dow uint64 // bitsets
	domStar, dowStar              bool
	expr                          string
}

type field struct {
	name     string
	min, max int
}

var fields = [5]field{
	{"minute", 0, 59}, {"hour", 0, 23}, {"day-of-month", 1, 31}, {"month", 1, 12}, {"day-of-week", 0, 7},
}

// Parse validates expr and returns its Schedule.
func Parse(expr string) (Schedule, error) {
	parts := strings.Fields(expr)
	if len(parts) != 5 {
		return Schedule{}, fmt.Errorf("cron: expected 5 fields, got %d", len(parts))
	}
	var sets [5]uint64
	var stars [5]bool
	for i, p := range parts {
		set, star, err := parseField(p, fields[i])
		if err != nil {
			return Schedule{}, err
		}
		sets[i], stars[i] = set, star
	}
	// Sunday as 7 folds onto 0.
	if sets[4]&(1<<7) != 0 {
		sets[4] = (sets[4] &^ (1 << 7)) | 1
	}
	return Schedule{
		minute: sets[0], hour: sets[1], dom: sets[2], month: sets[3], dow: sets[4],
		domStar: stars[2], dowStar: stars[4], expr: strings.Join(parts, " "),
	}, nil
}

// String returns the normalized expression.
func (s Schedule) String() string { return s.expr }

func parseField(p string, f field) (set uint64, star bool, err error) {
	for _, item := range strings.Split(p, ",") {
		if item == "" {
			return 0, false, fmt.Errorf("cron: empty item in %s", f.name)
		}
		rng, stepStr, hasStep := strings.Cut(item, "/")
		step := 1
		if hasStep {
			step, err = strconv.Atoi(stepStr)
			if err != nil || step < 1 {
				return 0, false, fmt.Errorf("cron: bad step %q in %s", stepStr, f.name)
			}
		}
		lo, hi := f.min, f.max
		switch {
		case rng == "*":
			if !hasStep {
				star = true
			}
		case strings.Contains(rng, "-"):
			a, b, _ := strings.Cut(rng, "-")
			if lo, err = atoiIn(a, f); err != nil {
				return 0, false, err
			}
			if hi, err = atoiIn(b, f); err != nil {
				return 0, false, err
			}
			if lo > hi {
				return 0, false, fmt.Errorf("cron: range %q is reversed in %s", rng, f.name)
			}
		default:
			if lo, err = atoiIn(rng, f); err != nil {
				return 0, false, err
			}
			hi = lo
			if hasStep {
				hi = f.max // "5/15" = from 5 to max, vixie extension
			}
		}
		for v := lo; v <= hi; v += step {
			set |= 1 << uint(v)
		}
	}
	return set, star, nil
}

func atoiIn(s string, f field) (int, error) {
	v, err := strconv.Atoi(s)
	if err != nil {
		return 0, fmt.Errorf("cron: %q is not a number in %s", s, f.name)
	}
	if v < f.min || v > f.max {
		return 0, fmt.Errorf("cron: %d is out of range %d-%d in %s", v, f.min, f.max, f.name)
	}
	return v, nil
}

func has(set uint64, v int) bool { return set&(1<<uint(v)) != 0 }

// Matches reports whether t (truncated to the minute) satisfies the schedule.
func (s Schedule) Matches(t time.Time) bool {
	if s.expr == "" {
		return false
	}
	return has(s.minute, t.Minute()) && has(s.hour, t.Hour()) && s.dayMatches(t)
}

func (s Schedule) dayMatches(t time.Time) bool {
	if !has(s.month, int(t.Month())) {
		return false
	}
	dom := has(s.dom, t.Day())
	dow := has(s.dow, int(t.Weekday()))
	switch {
	case s.domStar && s.dowStar:
		return true
	case s.domStar:
		return dow
	case s.dowStar:
		return dom
	default:
		return dom || dow // vixie: either constrained field matches
	}
}

// Next returns the first matching minute strictly after t, or false when
// none exists within five years (e.g. `0 0 30 2 *`).
func (s Schedule) Next(t time.Time) (time.Time, bool) {
	if s.expr == "" {
		return time.Time{}, false
	}
	c := t.Truncate(time.Minute).Add(time.Minute)
	limit := t.AddDate(5, 0, 0)
	for c.Before(limit) {
		switch {
		case !has(s.month, int(c.Month())):
			c = time.Date(c.Year(), c.Month()+1, 1, 0, 0, 0, 0, c.Location())
		case !s.dayMatches(c):
			c = time.Date(c.Year(), c.Month(), c.Day()+1, 0, 0, 0, 0, c.Location())
		case !has(s.hour, c.Hour()):
			c = time.Date(c.Year(), c.Month(), c.Day(), c.Hour()+1, 0, 0, 0, c.Location())
		case !has(s.minute, c.Minute()):
			c = c.Add(time.Minute)
		default:
			return c, true
		}
	}
	return time.Time{}, false
}

// Interval estimates the gap between two consecutive fires after t. Used
// to bound jitter (never more than half the interval).
func (s Schedule) Interval(t time.Time) time.Duration {
	a, ok := s.Next(t)
	if !ok {
		return 0
	}
	b, ok := s.Next(a)
	if !ok {
		return 0
	}
	return b.Sub(a)
}
