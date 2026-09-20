package climetrics

import (
	"database/sql"
	"os"
	"strings"
	"time"

	"github.com/cfpperche/picode/internal/clisession"
	"github.com/cfpperche/picode/internal/session"
)

// HermesMeter reports Hermes Agent from ~/.hermes/state.db, the same store
// clisession.HermesSource lists.
//
// Hermes is the only CLI here that already models the question this whole
// package had to answer: its sessions table carries estimated_cost_usd
// beside actual_cost_usd, with cost_status, cost_source, pricing_version
// and billing_mode to say which is which. Where it states a billing mode,
// that overrides whatever the operator configured — the CLI knows how its
// own account is paid for and PiCode does not have to guess.
//
// Its totals are per session, not per message, so cost and tokens are
// spread across that session's messages by their token share. Hermes does
// timestamp every message, so the window is honest even though the
// aggregate is not.
type HermesMeter struct{}

func (HermesMeter) CLI() string   { return "hermes" }
func (HermesMeter) Label() string { return "Hermes Agent" }

func (HermesMeter) Fingerprint() string { return dbFingerprint(clisession.HermesStatePath()) }

type hermesSession struct {
	cwd, model, title string
	billing           Billing
	cost              float64
	included          bool // Hermes marked it covered by a plan, not costed
	toks              session.TokenTotals
}

func (m HermesMeter) Meter(req Request) (Window, error) {
	path := clisession.HermesStatePath()
	if path == "" {
		return absentWindow(m, req), nil
	}
	if _, err := os.Stat(path); err != nil {
		return absentWindow(m, req), nil
	}
	db, err := openReadOnly(path)
	if err != nil {
		return Window{}, err
	}
	defer db.Close()

	sessions, err := hermesSessions(db, req)
	if err != nil {
		return Window{}, err
	}
	acc := newCliAcc(req, m.CLI())
	stated, err := hermesMessages(db, sessions, acc)
	if err != nil {
		return Window{}, err
	}

	billing := req.BillingFor(m.CLI())
	if stated != "" {
		billing = stated
	}
	// Hermes distinguishes "this cost nothing" from "this is not costed": a
	// plan-covered session carries cost_status "included" and a zero. Read
	// as a price, that zero would tell the operator Hermes was free.
	costed, included := 0, 0
	for _, s := range sessions {
		if s.included {
			included++
		} else {
			costed++
		}
	}
	return Window{
		CLI:      m.CLI(),
		Stats:    acc.result(),
		Coverage: hermesCoverage(m, billing, stated != "", costed, included, acc),
	}, nil
}

// hermesCan is what Hermes Agent is capable of recording.
var hermesCan = map[Signal]bool{
	SigCost: true, SigTokens: true, SigModel: true, SigMessages: true,
	SigTurns: true, SigTools: true, SigErrors: true,
}

func hermesCoverage(m HermesMeter, b Billing, stated bool, costed, included int, acc *cliAcc) CoverageRow {
	note := "Totals are per session; cost and tokens are spread over that session's messages by token share."
	if stated {
		note += " Billing mode comes from its own record, not the operator's setting."
	}
	sig := acc.evidence(hermesCan, nil)
	switch {
	case included > 0 && costed == 0:
		sig[SigCost] = StateNotReported
		note = "Marks every session in this window plan-included and prices none of them, so its spend is unmeasured rather than zero. " + note
	case included > 0:
		sig[SigCost] = StatePartial
		note = "Priced from " + itoa(costed) + " of " + itoa(costed+included) +
			" sessions — Hermes marks the rest plan-included and does not price them. " + note
	}
	return CoverageRow{CLI: m.CLI(), Label: m.Label(), Billing: b, Signals: sig, Note: note}
}

func hermesSessions(db *sql.DB, req Request) (map[string]*hermesSession, error) {
	cols, err := clisession.SQLiteTableColumns(db, "sessions")
	if err != nil || !cols["id"] {
		return nil, err
	}
	sel := []string{"id"}
	for _, c := range []string{
		"cwd", "model", "title", "billing_mode", "cost_status",
		"estimated_cost_usd", "actual_cost_usd",
		"input_tokens", "output_tokens", "cache_read_tokens", "cache_write_tokens", "reasoning_tokens",
	} {
		if cols[c] {
			sel = append(sel, c)
		}
	}
	rows, err := db.Query("SELECT " + strings.Join(sel, ", ") + " FROM sessions")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := map[string]*hermesSession{}
	for rows.Next() {
		row, ok := scanStrings(rows, sel)
		if !ok || row["id"] == "" {
			continue
		}
		// actual_cost_usd is what was billed; estimated_cost_usd is Hermes'
		// own reckoning before the bill lands. Prefer the settled number
		// and fall back rather than showing nothing.
		cost := atof(row["actual_cost_usd"])
		if cost == 0 {
			cost = atof(row["estimated_cost_usd"])
		}
		out[row["id"]] = &hermesSession{
			cwd: row["cwd"], model: row["model"], title: row["title"],
			billing:  hermesBilling(row["billing_mode"]),
			cost:     cost,
			included: cost == 0 && strings.EqualFold(strings.TrimSpace(row["cost_status"]), "included"),
			toks: session.TokenTotals{
				Input:      atoi64(row["input_tokens"]),
				Output:     atoi64(row["output_tokens"]),
				CacheRead:  atoi64(row["cache_read_tokens"]),
				CacheWrite: atoi64(row["cache_write_tokens"]),
				Reasoning:  atoi64(row["reasoning_tokens"]),
			},
		}
	}
	return out, rows.Err()
}

// hermesBilling maps Hermes' own vocabulary onto ours. It matches on a
// prefix because the real values are compound — this machine writes
// "subscription_included", not "subscription". An unrecognised word yields
// "" so the operator's setting stands, rather than a wrong guess.
func hermesBilling(mode string) Billing {
	m := strings.ToLower(strings.TrimSpace(mode))
	switch {
	case m == "":
		return ""
	case strings.HasPrefix(m, "subscription"), strings.HasPrefix(m, "plan"), strings.HasPrefix(m, "included"):
		return BillingSubscription
	case strings.HasPrefix(m, "api"), strings.HasPrefix(m, "metered"), strings.HasPrefix(m, "usage"):
		return BillingAPI
	default:
		return ""
	}
}

// hermesMessages dates each session's aggregate. It returns the billing
// mode Hermes stated, if any session stated one.
func hermesMessages(db *sql.DB, sessions map[string]*hermesSession, acc *cliAcc) (Billing, error) {
	cols, err := clisession.SQLiteTableColumns(db, "messages")
	if err != nil || !cols["session_id"] || !cols["timestamp"] {
		return "", err
	}
	sel := []string{"session_id", "timestamp"}
	for _, c := range []string{"role", "tool_name", "token_count", "finish_reason"} {
		if cols[c] {
			sel = append(sel, c)
		}
	}
	rows, err := db.Query("SELECT " + strings.Join(sel, ", ") + " FROM messages ORDER BY id")
	if err != nil {
		return "", err
	}
	defer rows.Close()

	type msg struct {
		at      time.Time
		role    string
		tool    string
		toks    int64
		abort   bool
		results int // finish_reason present: an inspected outcome
		errs    int
	}
	byS := map[string][]msg{}
	for rows.Next() {
		row, ok := scanStrings(rows, sel)
		if !ok {
			continue
		}
		sid := row["session_id"]
		if _, want := sessions[sid]; !want {
			continue
		}
		at := hermesTime(row["timestamp"])
		if at.IsZero() {
			continue
		}
		fin := strings.ToLower(strings.TrimSpace(row["finish_reason"]))
		byS[sid] = append(byS[sid], msg{
			at: at, role: row["role"], tool: row["tool_name"],
			toks:    atoi64(row["token_count"]),
			abort:   fin == "aborted" || fin == "cancelled",
			results: boolToInt(fin != ""),
			errs:    boolToInt(strings.Contains(fin, "error")),
		})
	}
	if err := rows.Err(); err != nil {
		return "", err
	}

	var stated Billing
	for sid, msgs := range byS {
		s := sessions[sid]
		if s.billing != "" {
			stated = s.billing
		}
		var total int64
		for _, m := range msgs {
			total += m.toks
		}
		for _, m := range msgs {
			share := 0.0
			if total > 0 {
				share = float64(m.toks) / float64(total)
			} else if len(msgs) > 0 {
				share = 1 / float64(len(msgs)) // no token counts: split evenly
			}
			e := cliEntry{
				at: m.at, key: sid, cwd: s.cwd, name: s.title,
				role: m.role, model: s.model, prov: "hermes",
				cost:    s.cost * share,
				toks:    scaleTokens(s.toks, share),
				abort:   m.abort,
				results: m.results,
				errs:    m.errs,
			}
			if m.tool != "" {
				e.tools = []string{m.tool}
			}
			acc.add(e)
		}
	}
	return stated, nil
}

// hermesTime reads the REAL epoch-seconds Hermes stores, falling back to
// RFC3339 for any build that writes text instead.
func hermesTime(raw string) time.Time {
	if raw == "" {
		return time.Time{}
	}
	// Text first: atof("2026-09-08T…") is 2026, which would land in 1970.
	if strings.ContainsAny(raw, "T:-") {
		if t, err := time.Parse(time.RFC3339, raw); err == nil {
			return t
		}
		return time.Time{}
	}
	if f := atof(raw); f > 0 {
		sec := int64(f)
		return time.Unix(sec, int64((f-float64(sec))*1e9))
	}
	if t, err := time.Parse(time.RFC3339, raw); err == nil {
		return t
	}
	return time.Time{}
}

func scaleTokens(t session.TokenTotals, share float64) session.TokenTotals {
	return session.TokenTotals{
		Input:      int64(float64(t.Input) * share),
		Output:     int64(float64(t.Output) * share),
		CacheRead:  int64(float64(t.CacheRead) * share),
		CacheWrite: int64(float64(t.CacheWrite) * share),
		Reasoning:  int64(float64(t.Reasoning) * share),
	}
}
