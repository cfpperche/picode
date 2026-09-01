package session

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// DayBucket is one calendar day's totals in the current period's series.
type DayBucket struct {
	Date     string  `json:"date"` // YYYY-MM-DD, in the caller's Location
	Cost     float64 `json:"cost"`
	Messages int     `json:"messages"`
}

// ProviderBucket is one provider's totals in the current period.
type ProviderBucket struct {
	Provider string  `json:"provider"` // "unknown" when no model_change line was ever seen
	Cost     float64 `json:"cost"`
	Messages int     `json:"messages"`
}

// PeriodTotals is the aggregate over one window.
type PeriodTotals struct {
	Cost     float64 `json:"cost"`
	Messages int     `json:"messages"`
	Sessions int     `json:"sessions"` // distinct session files with >=1 in-window message
}

// WindowStats is the response shape for a spend/activity aggregation window.
type WindowStats struct {
	From       string           `json:"from"` // RFC3339, in loc
	To         string           `json:"to"`   // RFC3339, exclusive, in loc
	Current    PeriodTotals     `json:"current"`
	Prior      *PeriodTotals    `json:"prior,omitempty"` // nil when priorFrom is the zero time (range=all)
	ByProvider []ProviderBucket `json:"byProvider"`      // current period, sorted desc by Cost
	Series     []DayBucket      `json:"series"`          // current period, one entry per calendar day, zero-filled, ascending
}

type statsAcc struct {
	from, to, priorFrom time.Time
	loc                 *time.Location
	current             PeriodTotals
	prior               PeriodTotals
	curSessions         map[string]struct{}
	priorSessions       map[string]struct{}
	byProvider          map[string]*ProviderBucket
	byDay               map[string]*DayBucket
}

func newStatsAcc(from, to, priorFrom time.Time, loc *time.Location) *statsAcc {
	return &statsAcc{
		from: from, to: to, priorFrom: priorFrom, loc: loc,
		curSessions:   map[string]struct{}{},
		priorSessions: map[string]struct{}{},
		byProvider:    map[string]*ProviderBucket{},
		byDay:         map[string]*DayBucket{},
	}
}

// StatsForRange aggregates every session under the real sessions root.
func StatsForRange(from, to, priorFrom time.Time, loc *time.Location) (WindowStats, error) {
	return StatsRoot(Root(), from, to, priorFrom, loc)
}

// StatsRoot aggregates every session under root for [from,to), plus an
// equal-length prior window [priorFrom,from) for the delta comparison.
// priorFrom.IsZero() skips the prior computation (range=all).
func StatsRoot(root string, from, to, priorFrom time.Time, loc *time.Location) (WindowStats, error) {
	acc := newStatsAcc(from, to, priorFrom, loc)
	if root != "" {
		ents, err := os.ReadDir(root)
		if err != nil && !os.IsNotExist(err) {
			return WindowStats{}, err
		}
		for _, e := range ents {
			if !e.IsDir() {
				continue
			}
			acc.scanDir(filepath.Join(root, e.Name()))
		}
	}
	return acc.result(), nil
}

// scanDir walks one workspace's session directory, skipping files whose
// mtime cannot possibly hold an in-window message (mirrors the pre-filter
// sweepOrphanSessions already uses for the same reason: cheap to check,
// expensive to open and parse).
func (a *statsAcc) scanDir(dir string) {
	ents, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	for _, e := range ents {
		if e.IsDir() || filepath.Ext(e.Name()) != ".jsonl" {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		if !a.priorFrom.IsZero() && info.ModTime().Before(a.priorFrom) {
			continue // cannot contain any message in [priorFrom, to)
		}
		a.scanFile(filepath.Join(dir, e.Name()), info.ModTime())
	}
}

func (a *statsAcc) scanFile(path string, mtime time.Time) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()

	provider := ""
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for sc.Scan() {
		line := sc.Bytes()
		if len(line) == 0 {
			continue
		}
		var raw map[string]any
		if json.Unmarshal(line, &raw) != nil {
			continue
		}
		switch raw["type"] {
		case "model_change":
			if p, _ := raw["provider"].(string); p != "" {
				provider = p
			}
		case "message":
			ts := entryTS(raw)
			t := mtime
			if ts > 0 {
				t = time.Unix(ts, 0)
			}
			a.add(path, t, provider, costFrom(raw["message"]))
		}
	}
}

// add files one message's cost/count into whichever window t falls in.
func (a *statsAcc) add(path string, t time.Time, provider string, cost float64) {
	prov := provider
	if prov == "" {
		prov = "unknown"
	}
	switch {
	case !a.priorFrom.IsZero() && t.Before(a.priorFrom):
		return // older than the widened (current+prior) window
	case !t.Before(a.to):
		return // in the future relative to the window (defensive; should not happen)
	case !a.from.IsZero() && t.Before(a.from):
		a.prior.Cost += cost
		a.prior.Messages++
		a.priorSessions[path] = struct{}{}
	default:
		a.current.Cost += cost
		a.current.Messages++
		a.curSessions[path] = struct{}{}

		pb := a.byProvider[prov]
		if pb == nil {
			pb = &ProviderBucket{Provider: prov}
			a.byProvider[prov] = pb
		}
		pb.Cost += cost
		pb.Messages++

		day := t.In(a.loc).Format("2006-01-02")
		db := a.byDay[day]
		if db == nil {
			db = &DayBucket{Date: day}
			a.byDay[day] = db
		}
		db.Cost += cost
		db.Messages++
	}
}

func (a *statsAcc) result() WindowStats {
	a.current.Sessions = len(a.curSessions)
	a.prior.Sessions = len(a.priorSessions)

	providers := make([]ProviderBucket, 0, len(a.byProvider))
	for _, pb := range a.byProvider {
		providers = append(providers, *pb)
	}
	sort.Slice(providers, func(i, j int) bool { return providers[i].Cost > providers[j].Cost })

	st := WindowStats{
		From:       a.from.Format(time.RFC3339),
		To:         a.to.Format(time.RFC3339),
		Current:    a.current,
		ByProvider: providers,
		Series:     a.series(),
	}
	if !a.priorFrom.IsZero() {
		p := a.prior
		st.Prior = &p
	}
	return st
}

// series zero-fills every calendar day in the current window. For range=all
// (a.from is the zero time) the walk starts at the earliest day that
// actually has data instead of the zero time — iterating from year 1 would
// be unbounded and pointless. No data at all yields an empty series, which
// is the honest "nothing happened" signal rather than a fabricated range.
func (a *statsAcc) series() []DayBucket {
	from := a.from
	if from.IsZero() {
		if len(a.byDay) == 0 {
			return []DayBucket{}
		}
		days := make([]string, 0, len(a.byDay))
		for d := range a.byDay {
			days = append(days, d)
		}
		sort.Strings(days)
		earliest, err := time.ParseInLocation("2006-01-02", days[0], a.loc)
		if err != nil {
			return []DayBucket{}
		}
		from = earliest
	}
	out := []DayBucket{}
	for d := from; d.Before(a.to); d = d.AddDate(0, 0, 1) {
		key := d.In(a.loc).Format("2006-01-02")
		if db := a.byDay[key]; db != nil {
			out = append(out, *db)
		} else {
			out = append(out, DayBucket{Date: key})
		}
	}
	return out
}
