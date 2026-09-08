package climetrics

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/cfpperche/picode/internal/clisession"
	"github.com/cfpperche/picode/internal/session"
)

// CodexMeter reports Codex from the rollouts under ~/.codex/sessions, the
// same tree clisession.CodexSource lists.
//
// Codex is the CLI that never prices a token. It records tokens in detail —
// `token_count` carries both a cumulative total and `last_token_usage` for
// the turn just finished — and it records something no other CLI here does:
// the quota windows its plan is actually constrained by (`used_percent`,
// `window_minutes`, `resets_at`, `plan_type`). PiCode reports those instead
// of a dollar figure, because the alternative is a price table that ages
// silently and is wrong without ever looking wrong (ADR-0097).
//
// The tree is the largest on this machine — 2.24 GB across 915 files — and
// it is partitioned YYYY/MM/DD, so a window skips whole directories by name
// before stat-ing anything inside them.
type CodexMeter struct{}

func (CodexMeter) CLI() string   { return "codex" }
func (CodexMeter) Label() string { return "Codex" }

// Fingerprint sweeps only the day directories a window could touch, which
// is what keeps the biggest tree cheap to poll.
func (CodexMeter) Fingerprint() string {
	root := clisession.CodexSessionsRoot()
	if root == "" {
		return ""
	}
	if _, err := os.Stat(root); os.IsNotExist(err) {
		return "0:0:0"
	}
	var n, size, newest int64
	_ = filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(d.Name(), ".jsonl") {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return nil
		}
		n++
		size += info.Size()
		if m := info.ModTime().UnixNano(); m > newest {
			newest = m
		}
		return nil
	})
	return itoa64(n) + ":" + itoa64(size) + ":" + itoa64(newest)
}

func (m CodexMeter) Meter(req Request) (Window, error) {
	root := clisession.CodexSessionsRoot()
	if root == "" {
		return absentWindow(m, req), nil
	}
	if _, err := os.Stat(root); err != nil {
		return absentWindow(m, req), nil
	}
	acc := newGuestAcc(req, m.CLI())
	var limits []LimitWindow
	var newestLimit time.Time

	for _, path := range codexFiles(root, req) {
		p := cachedParse(path, codexParse)
		replay(p, acc, req)
		if len(p.limits) > 0 && p.limitAt.After(newestLimit) {
			limits, newestLimit = p.limits, p.limitAt
		}
	}

	w := Window{CLI: m.CLI(), Stats: acc.result(), Coverage: codexCoverage(m, req.BillingFor(m.CLI()), len(limits) > 0)}
	w.Limits = limits
	return w, nil
}

func codexCoverage(m CodexMeter, b Billing, sawLimits bool) CoverageRow {
	limits := StateNotReported
	if sawLimits {
		limits = StateReported
	}
	return CoverageRow{
		CLI: m.CLI(), Label: m.Label(), Billing: b,
		Signals: map[Signal]State{
			SigCost:     StateNotReported,
			SigTokens:   StateReported,
			SigModel:    StateReported,
			SigMessages: StateReported,
			SigTurns:    StateReported,
			SigTools:    StateReported,
			SigErrors:   StateNotReported,
			SigImpact:   StateNotReported,
			SigTiming:   StateNotReported,
			SigLimits:   limits,
		},
		Note: "Never prices a token, so it is not counted in spend — its quota windows stand in for cost instead. It records no per-turn error state or edit counts.",
	}
}

// codexFiles lists the rollouts a window could contain, skipping whole day
// directories by name. Codex writes ~/.codex/sessions/YYYY/MM/DD/, so a
// 7-day window opens seven directories out of hundreds.
func codexFiles(root string, req Request) []string {
	var out []string
	from := req.PriorFrom
	if from.IsZero() {
		from = req.From
	}
	_ = filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if !codexDirInRange(root, p, from, req.To) {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(d.Name(), ".jsonl") {
			out = append(out, p)
		}
		return nil
	})
	return out
}

// codexDirInRange decides whether a YYYY, YYYY/MM or YYYY/MM/DD directory
// can hold anything in [from, to). A path shallower than a full date always
// passes: it is only a prefix, and pruning it on partial information would
// drop real days.
func codexDirInRange(root, dir string, from, to time.Time) bool {
	if from.IsZero() {
		return true
	}
	rel, err := filepath.Rel(root, dir)
	if err != nil || rel == "." {
		return true
	}
	parts := strings.Split(filepath.ToSlash(rel), "/")
	nums := make([]int, 0, 3)
	for _, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil {
			return true // not a date segment: do not prune what we cannot read
		}
		nums = append(nums, n)
	}
	switch len(nums) {
	case 1:
		return nums[0] >= from.Year() && nums[0] <= to.Year()
	case 2:
		start := time.Date(nums[0], time.Month(nums[1]), 1, 0, 0, 0, 0, time.UTC)
		return !start.AddDate(0, 1, 0).Before(from.UTC()) && start.Before(to.UTC())
	case 3:
		day := time.Date(nums[0], time.Month(nums[1]), nums[2], 0, 0, 0, 0, time.UTC)
		// One day of slack each way: rollout paths are UTC-dated and the
		// window is local, so a boundary day must not be pruned.
		return !day.AddDate(0, 0, 2).Before(from.UTC()) && day.AddDate(0, 0, -1).Before(to.UTC())
	default:
		return true
	}
}

type codexLine struct {
	Timestamp string `json:"timestamp"`
	Type      string `json:"type"`
	Payload   struct {
		Type string `json:"type"`
		// session_meta
		ID       string `json:"id"`
		Cwd      string `json:"cwd"`
		Provider string `json:"model_provider"`
		// turn_context
		Model string `json:"model"`
		// response_item
		Role string `json:"role"`
		Name string `json:"name"`
		// event_msg: token_count
		Info *struct {
			Last struct {
				Input     int64 `json:"input_tokens"`
				Cached    int64 `json:"cached_input_tokens"`
				CacheWr   int64 `json:"cache_write_input_tokens"`
				Output    int64 `json:"output_tokens"`
				Reasoning int64 `json:"reasoning_output_tokens"`
			} `json:"last_token_usage"`
		} `json:"info"`
		RateLimits *struct {
			Primary   *codexLimit `json:"primary"`
			Secondary *codexLimit `json:"secondary"`
			PlanType  string      `json:"plan_type"`
		} `json:"rate_limits"`
	} `json:"payload"`
}

type codexLimit struct {
	UsedPercent float64 `json:"used_percent"`
	WindowMin   int     `json:"window_minutes"`
	ResetsAt    int64   `json:"resets_at"`
}

// codexParse reads one rollout into a window-independent parse.
func codexParse(path string) *parsed {
	out := &parsed{}
	f, err := os.Open(path)
	if err != nil {
		return out
	}
	defer f.Close()

	cwd, provider, model, id := "", "", "", ""
	var pendingTools []string
	add := func(e guestEntry) {
		out.ents = append(out.ents, e)
		out.units += e.toks.Input + e.toks.Output + e.toks.CacheRead + e.toks.CacheWrite
	}

	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 8*1024*1024)
	for sc.Scan() {
		raw := sc.Bytes()
		if len(raw) == 0 {
			continue
		}
		var l codexLine
		if json.Unmarshal(raw, &l) != nil {
			continue
		}
		at, err := time.Parse(time.RFC3339, l.Timestamp)
		if err != nil {
			continue
		}
		p := l.Payload
		switch l.Type {
		case "session_meta":
			if p.Cwd != "" {
				cwd = p.Cwd
			}
			if p.Provider != "" {
				provider = p.Provider
			}
			if p.ID != "" {
				id = p.ID
			}
		case "turn_context":
			if p.Cwd != "" {
				cwd = p.Cwd
			}
			if p.Model != "" {
				model = p.Model
			}
		case "response_item":
			// A function call is the tool; its output is the same call's
			// other half and is not counted twice.
			if p.Type == "function_call" && p.Name != "" {
				pendingTools = append(pendingTools, p.Name)
			}
			if p.Type == "message" && p.Role == "user" {
				add(guestEntry{at: at, key: codexKey(path, id), cwd: cwd, role: "user"})
			}
		case "event_msg":
			switch p.Type {
			case "user_message":
				add(guestEntry{at: at, key: codexKey(path, id), cwd: cwd, role: "user"})
			case "token_count":
				if p.RateLimits != nil && at.After(out.limitAt) {
					out.limits, out.limitAt = codexLimits(*p.RateLimits, at), at
				}
				if p.Info == nil {
					continue
				}
				u := p.Info.Last
				// last_token_usage is the turn that just finished — the one
				// windowable token figure Codex writes. The cumulative
				// total beside it would double-count on every line.
				e := guestEntry{
					at: at, key: codexKey(path, id), cwd: cwd,
					role: "assistant", model: model, prov: provider,
					toks: session.TokenTotals{
						Input:      u.Input,
						Output:     u.Output,
						CacheRead:  u.Cached,
						CacheWrite: u.CacheWr,
						Reasoning:  u.Reasoning,
					},
					tools: pendingTools,
				}
				pendingTools = nil
				add(e)
			case "turn_aborted":
				add(guestEntry{at: at, key: codexKey(path, id), cwd: cwd, role: "assistant", model: model, prov: provider, abort: true})
			}
		}
	}
	// The cwd only becomes known at session_meta, which is the first line;
	// entries built before it would carry "" and fall out of every scope.
	for i := range out.ents {
		if out.ents[i].cwd == "" {
			out.ents[i].cwd = cwd
		}
	}
	return out
}

func codexKey(path, id string) string {
	if id != "" {
		return id
	}
	return path
}

func codexLimits(rl struct {
	Primary   *codexLimit `json:"primary"`
	Secondary *codexLimit `json:"secondary"`
	PlanType  string      `json:"plan_type"`
}, at time.Time) []LimitWindow {
	var out []LimitWindow
	add := func(l *codexLimit) {
		if l == nil || l.WindowMin <= 0 {
			return
		}
		w := LimitWindow{
			CLI: "codex", Label: windowLabel(l.WindowMin),
			UsedPercent: l.UsedPercent, WindowMin: l.WindowMin,
			Plan: rl.PlanType, ObservedAt: at.Format(time.RFC3339),
		}
		if l.ResetsAt > 0 {
			w.ResetsAt = time.Unix(l.ResetsAt, 0).Format(time.RFC3339)
		}
		out = append(out, w)
	}
	add(rl.Primary)
	add(rl.Secondary)
	return out
}
