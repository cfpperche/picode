package server

import (
	"net/http"
	"strings"
	"time"

	"github.com/cfpperche/picode/internal/session"
)

type sessionStatsView struct {
	session.WindowStats
	Range string `json:"range"` // the clamped value actually used, not the raw query
}

func handleSessionStats(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rng := normalizeRange(r.URL.Query().Get("range"))
		from, to, priorFrom := statsWindow(rng, time.Now(), time.Local)
		st, err := session.StatsForRange(from, to, priorFrom, time.Local)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, sessionStatsView{WindowStats: st, Range: rng})
	}
}

// normalizeRange clamps to the 4 supported values, defaulting rather than
// 400ing on drift — same spirit as gitgraph.go's graphLimit/graphRemotes.
func normalizeRange(raw string) string {
	switch strings.TrimSpace(raw) {
	case "today", "7d", "30d", "all":
		return strings.TrimSpace(raw)
	default:
		return "7d"
	}
}

// statsWindow computes [from,to) for the current period and priorFrom for
// the equal-length preceding period, in loc. to is always the start of
// tomorrow (today's activity-in-progress counts). priorFrom is the zero
// time for "all" — the caller's signal to skip the prior comparison
// entirely, since there is no well-defined "period before all time".
//
// Day boundaries use AddDate (calendar-day arithmetic), not a fixed
// Duration subtraction, so a DST transition never produces a 23h/25h window.
func statsWindow(rng string, now time.Time, loc *time.Location) (from, to, priorFrom time.Time) {
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
	tomorrow := today.AddDate(0, 0, 1)
	switch rng {
	case "today":
		return today, tomorrow, today.AddDate(0, 0, -1)
	case "30d":
		from = today.AddDate(0, 0, -29)
		return from, tomorrow, from.AddDate(0, 0, -30)
	case "all":
		return time.Time{}, tomorrow, time.Time{}
	default: // "7d" and the normalizeRange fallback
		from = today.AddDate(0, 0, -6)
		return from, tomorrow, from.AddDate(0, 0, -7)
	}
}
