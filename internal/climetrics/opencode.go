package climetrics

import (
	"database/sql"
	"encoding/json"
	"os"
	"strings"
	"time"

	"github.com/cfpperche/picode/internal/clisession"
	"github.com/cfpperche/picode/internal/session"
)

// OpenCodeMeter reports OpenCode from its SQLite store, the same file
// clisession.OpenCodeSource lists. It is the richest of the set: every assistant
// message row carries its own cost, token split, provider and model, so no
// derivation is needed and no snapshot is missing — the one CLI whose
// cost is simply `reported`.
//
// The session table also carries summary_files, which makes OpenCode the
// only CLI here that counts changed *files*. Impact deliberately does not
// aggregate it — see the note on Impact — so only its line counts are read.
type OpenCodeMeter struct{}

func (OpenCodeMeter) CLI() string   { return "opencode" }
func (OpenCodeMeter) Label() string { return "OpenCode" }

// Fingerprint is the store's size and mtime. One stat, no query: the whole
// point of the poll is that an untouched database costs nothing.
func (OpenCodeMeter) Fingerprint() string { return dbFingerprint(clisession.OpenCodeDBPath()) }

func (m OpenCodeMeter) Meter(req Request) (Window, error) {
	path := clisession.OpenCodeDBPath()
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

	dirs, impact, err := opencodeSessions(db, req)
	if err != nil {
		return Window{}, err
	}
	acc := newCliAcc(req, m.CLI())
	if err := opencodeMessages(db, req, dirs, acc); err != nil {
		return Window{}, err
	}

	hasImpact := impact.LinesAdded != 0 || impact.LinesRemoved != 0
	w := Window{CLI: m.CLI(), Stats: acc.result(), Coverage: opencodeCoverage(m, req.BillingFor(m.CLI()), acc, hasImpact)}
	if hasImpact {
		i := impact
		w.Impact = &i
	}
	return w, nil
}

// opencodeCan is what OpenCode is capable of recording. Tool calls live in
// its part rows, which this window does not read.
var opencodeCan = map[Signal]bool{
	SigCost: true, SigTokens: true, SigModel: true, SigMessages: true,
	SigTurns: true, SigErrors: true, SigImpact: true,
}

func opencodeCoverage(m OpenCodeMeter, b Billing, acc *cliAcc, hasImpact bool) CoverageRow {
	return CoverageRow{
		CLI: m.CLI(), Label: m.Label(), Billing: b,
		Signals: acc.evidence(opencodeCan, map[Signal]bool{SigImpact: hasImpact}),
		Note:    "Tool calls live in its part rows, which this window does not read; it records no request duration or quota window.",
	}
}

// opencodeSessions returns the in-scope session ids mapped to their folder,
// and the window's code impact. Impact comes off the session row rather
// than a message, so it is a session-lifetime figure: a session that
// started before the window contributes its whole diff. That is the same
// trade the file counts are worth and it is named in the coverage note.
func opencodeSessions(db *sql.DB, req Request) (map[string]string, Impact, error) {
	cols, err := clisession.SQLiteTableColumns(db, "session")
	if err != nil || !cols["id"] {
		return nil, Impact{}, err
	}
	sel := []string{"id"}
	for _, c := range []string{"directory", "summary_additions", "summary_deletions", "time_updated"} {
		if cols[c] {
			sel = append(sel, c)
		}
	}
	rows, err := db.Query("SELECT " + strings.Join(sel, ", ") + " FROM session")
	if err != nil {
		return nil, Impact{}, err
	}
	defer rows.Close()

	dirs := map[string]string{}
	var impact Impact
	for rows.Next() {
		vals := make([]any, len(sel))
		holders := make([]sql.RawBytes, len(sel))
		for i := range vals {
			vals[i] = &holders[i]
		}
		if rows.Scan(vals...) != nil {
			continue
		}
		row := map[string]string{}
		for i, name := range sel {
			row[name] = string(holders[i])
		}
		id := row["id"]
		if id == "" {
			continue
		}
		dir := row["directory"]
		dirs[id] = dir
		// Only count a session's diff when it was touched inside the window.
		if ms := atoi64(row["time_updated"]); ms > 0 {
			if t := time.UnixMilli(ms); t.Before(req.From) || !t.Before(req.To) {
				continue
			}
		}
		impact.LinesAdded += atoi64(row["summary_additions"])
		impact.LinesRemoved += atoi64(row["summary_deletions"])
	}
	return dirs, impact, rows.Err()
}

// opencodeMsg is the JSON OpenCode stores in message.data.
type opencodeMsg struct {
	Role       string  `json:"role"`
	Cost       float64 `json:"cost"`
	ModelID    string  `json:"modelID"`
	ProviderID string  `json:"providerID"`
	Error      any     `json:"error"`
	Tokens     struct {
		Input     int64 `json:"input"`
		Output    int64 `json:"output"`
		Reasoning int64 `json:"reasoning"`
		Cache     struct {
			Read  int64 `json:"read"`
			Write int64 `json:"write"`
		} `json:"cache"`
	} `json:"tokens"`
}

func opencodeMessages(db *sql.DB, req Request, dirs map[string]string, acc *cliAcc) error {
	cols, err := clisession.SQLiteTableColumns(db, "message")
	if err != nil || !cols["session_id"] || !cols["data"] {
		return err
	}
	rows, err := db.Query("SELECT session_id, time_created, data FROM message")
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var sid string
		var created int64
		var data []byte
		if rows.Scan(&sid, &created, &data) != nil {
			continue
		}
		dir, ok := dirs[sid]
		if !ok {
			continue
		}
		var m opencodeMsg
		if json.Unmarshal(data, &m) != nil {
			continue
		}
		e := cliEntry{
			at:    time.UnixMilli(created),
			key:   sid,
			cwd:   dir,
			role:  m.Role,
			model: m.ModelID,
			prov:  m.ProviderID,
			cost:  m.Cost,
			toks: session.TokenTotals{
				Input:      m.Tokens.Input,
				Output:     m.Tokens.Output,
				CacheRead:  m.Tokens.Cache.Read,
				CacheWrite: m.Tokens.Cache.Write,
				Reasoning:  m.Tokens.Reasoning,
			},
		}
		// Every assistant row is an inspected outcome: `error` is an
		// optional key on the message JSON, absent when the turn succeeded.
		if m.Role == "assistant" {
			e.results = 1
		}
		if m.Error != nil {
			e.errs = 1
		}
		acc.add(e)
	}
	return rows.Err()
}

func openReadOnly(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", "file:"+path+"?mode=ro&_pragma=busy_timeout(1000)")
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	return db, nil
}

// dbFingerprint describes a SQLite store by size and mtime. It deliberately
// does not query: a poll must cost a stat, and any write to the database
// moves both.
func dbFingerprint(path string) string {
	if path == "" {
		return ""
	}
	st, err := os.Stat(path)
	if os.IsNotExist(err) {
		return "0:0"
	}
	if err != nil {
		return ""
	}
	return itoa64(st.Size()) + ":" + itoa64(st.ModTime().UnixNano())
}
