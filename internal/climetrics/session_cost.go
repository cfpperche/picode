package climetrics

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/cfpperche/picode/internal/pricing"
	"github.com/cfpperche/picode/internal/session"
)

// SessionCost is one session file's own accounting, for an agent's exit
// record (ADR-0194): what the CLI recorded, plus a list-price estimate for
// the turns it left unpriced (ADR-0185), kept apart so an estimate never
// passes for a recorded figure.
type SessionCost struct {
	Cost      float64 `json:"cost"`      // recorded plus estimated
	Estimated float64 `json:"estimated"` // the list-price part of Cost
	Unpriced  int     `json:"unpriced"`  // turns with tokens the table does not price
	Tokens    int64   `json:"tokens"`    // input + output + cache read + cache write
	Turns     int     `json:"turns"`     // assistant messages
	// Models counts assistant turns per model the session file names, so an
	// exit can say which model a terminal CLI ran (the agent row does not).
	Models map[string]int `json:"models,omitempty"`
}

// Metered says whether MeterSession prices this CLI's conversations (the
// automations editor's METERED_CLIS pins the same answer).
func Metered(cli string) bool {
	switch cli {
	case "pi", "claude-code", "codex", "omp", "muse", "grok", "hermes", "opencode":
		return true
	}
	return false
}

// MeterSession prices one conversation wherever its CLI keeps it: a file
// (Pi, Claude Code, Codex, Omp, Muse), a Grok session folder beside its
// prompt history (path is that folder, or prompt_history.jsonl for a
// session Grok kept no folder for yet, and id the session), or a row
// of OpenCode's or Hermes' SQLite store (path is the store, id the session).
// ok false means not measured — never a zero that reads as free.
func MeterSession(cli, path, id string, prices *pricing.Table) (SessionCost, bool) {
	switch cli {
	case "grok":
		dir := path
		if st, err := os.Stat(dir); err != nil || !st.IsDir() {
			if id == "" || path == "" {
				return SessionCost{}, false
			}
			dir = filepath.Join(filepath.Dir(path), id)
			if st, err := os.Stat(dir); err != nil || !st.IsDir() {
				return SessionCost{}, false
			}
		}
		p := grokSessionParse(dir)
		if p == nil {
			return SessionCost{}, false
		}
		return sumSession(p.ents, prices), true
	case "opencode":
		return opencodeSessionCost(path, id, prices)
	case "hermes":
		return hermesSessionCost(path, id, prices)
	}
	return MeterSessionFile(cli, path, prices)
}

func opencodeSessionCost(path, id string, prices *pricing.Table) (SessionCost, bool) {
	if path == "" || id == "" {
		return SessionCost{}, false
	}
	db, err := openReadOnly(path)
	if err != nil {
		return SessionCost{}, false
	}
	defer db.Close()
	rows, err := db.Query("SELECT data FROM message WHERE session_id = ?", id)
	if err != nil {
		return SessionCost{}, false
	}
	defer rows.Close()
	var ents []cliEntry
	for rows.Next() {
		var data []byte
		if rows.Scan(&data) != nil {
			continue
		}
		var m opencodeMsg
		if json.Unmarshal(data, &m) != nil {
			continue
		}
		ents = append(ents, cliEntry{role: m.Role, model: m.ModelID, cost: m.Cost, toks: session.TokenTotals{
			Input: m.Tokens.Input, Output: m.Tokens.Output, CacheRead: m.Tokens.Cache.Read,
			CacheWrite: m.Tokens.Cache.Write, Reasoning: m.Tokens.Reasoning,
		}})
	}
	if rows.Err() != nil || len(ents) == 0 {
		return SessionCost{}, false
	}
	return sumSession(ents, prices), true
}

// hermesSessionCost reads the session's own totals (Hermes keeps cost and
// tokens per session, not per message) and counts its assistant turns.
func hermesSessionCost(path, id string, prices *pricing.Table) (SessionCost, bool) {
	if path == "" || id == "" {
		return SessionCost{}, false
	}
	db, err := openReadOnly(path)
	if err != nil {
		return SessionCost{}, false
	}
	defer db.Close()
	sessions, err := hermesSessions(db, Request{})
	if err != nil {
		return SessionCost{}, false
	}
	s := sessions[id]
	if s == nil {
		return SessionCost{}, false
	}
	out := SessionCost{Tokens: s.toks.Input + s.toks.Output + s.toks.CacheRead + s.toks.CacheWrite}
	var turns int
	if err := db.QueryRow("SELECT COUNT(*) FROM messages WHERE session_id = ? AND role = 'assistant'", id).Scan(&turns); err == nil {
		out.Turns = turns
	}
	if s.model != "" && turns > 0 {
		out.Models = map[string]int{s.model: turns}
	}
	switch {
	case s.cost > 0 || s.included:
		out.Cost = s.cost
	case s.toks != (session.TokenTotals{}):
		if pc, ok := prices.Cost(s.model, s.toks.Input, s.toks.Output, s.toks.CacheRead, s.toks.CacheWrite, 0); ok && pc > 0 {
			out.Cost, out.Estimated = pc, pc
		} else if !ok {
			out.Unpriced++
		}
	}
	return out, true
}

// MeterSessionFile reads one session file of a CLI that keeps its sessions
// as files. ok is false for any other CLI, or a file that cannot be read:
// the caller records "not measured", never zero.
func MeterSessionFile(cli, path string, prices *pricing.Table) (SessionCost, bool) {
	st, err := os.Stat(path)
	if err != nil || st.IsDir() {
		return SessionCost{}, false
	}
	var p *parsed
	switch cli {
	case "pi":
		p = piParse(path, st.ModTime())
	case "claude-code":
		p = ccParse(path)
	case "codex":
		p = codexParse(path)
	case "omp":
		p = ompParse(path, st.ModTime())
	case "muse":
		p = museParse(path, st.ModTime())
	default:
		return SessionCost{}, false
	}
	if p == nil {
		return SessionCost{}, false
	}
	return sumSession(p.ents, prices), true
}

// sumSession adds up one conversation's entries: tokens, assistant turns by
// model, the CLI's own cost where it recorded one, list price otherwise.
func sumSession(ents []cliEntry, prices *pricing.Table) SessionCost {
	var out SessionCost
	for _, e := range ents {
		out.Tokens += e.toks.Input + e.toks.Output + e.toks.CacheRead + e.toks.CacheWrite
		if e.role == "assistant" {
			out.Turns++
			if e.model != "" {
				if out.Models == nil {
					out.Models = map[string]int{}
				}
				out.Models[e.model]++
			}
		}
		c := e.cost
		if c == 0 && e.role == "assistant" && e.toks != (session.TokenTotals{}) {
			if pc, ok := prices.Cost(e.model, e.toks.Input, e.toks.Output, e.toks.CacheRead, e.toks.CacheWrite, e.cw1h); ok && pc > 0 {
				c = pc
				out.Estimated += pc
			} else if !ok {
				out.Unpriced++
			}
		}
		out.Cost += c
	}
	return out
}
