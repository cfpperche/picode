package climetrics

import (
	"os"

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
}

// MeterSessionFile reads one session file of a CLI that keeps its sessions
// as files. ok is false for a CLI whose sessions live in a database, or a
// file that cannot be read: the caller records "not measured", never zero.
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
	var out SessionCost
	for _, e := range p.ents {
		out.Tokens += e.toks.Input + e.toks.Output + e.toks.CacheRead + e.toks.CacheWrite
		if e.role == "assistant" {
			out.Turns++
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
	return out, true
}
