package climetrics

import (
	"database/sql"
	"encoding/json"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/cfpperche/picode/internal/clisession"
)

// AgyMeter reports Antigravity from its conversation index, the same
// ~/.gemini/antigravity-cli/conversation_summaries.db clisession.AgySource
// lists.
//
// The index is honest about only three things — that a conversation
// exists, how many steps it ran, and when it was last touched — so this
// meter stays inside exactly that: step counts filed as turns on the day
// of last activity. A long conversation lands whole on one day rather
// than spread across the days it actually ran on; the coverage note says
// so, because a smeared count presented as daily truth would be the
// fabrication the dashboard refuses elsewhere.
//
// Per-conversation stores (conversations/<id>.db) stay unread: their
// schema is not verified against a real install, and an unverified parse
// is a silent miscount waiting to happen. The day one is, this meter
// grows model and tool signals from it.
type AgyMeter struct{}

func (AgyMeter) CLI() string   { return "agy" }
func (AgyMeter) Label() string { return "Antigravity" }

func (AgyMeter) Fingerprint() string { return dbFingerprint(clisession.AgyDBPath()) }

func (m AgyMeter) Meter(req Request) (Window, error) {
	path := clisession.AgyDBPath()
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

	acc := newCliAcc(req, m.CLI())
	if err := agyConversations(db, acc); err != nil {
		return Window{}, err
	}
	return Window{
		CLI:      m.CLI(),
		Stats:    acc.result(),
		Coverage: agyCoverage(m, req.BillingFor(m.CLI()), acc),
	}, nil
}

// agyCan is what the conversation index is capable of saying: steps and
// sessions. Model, cost, tokens and tools live a layer down, unverified.
var agyCan = map[Signal]bool{
	SigMessages: true, SigTurns: true,
}

func agyCoverage(m AgyMeter, b Billing, acc *cliAcc) CoverageRow {
	return CoverageRow{
		CLI: m.CLI(), Label: m.Label(), Billing: b,
		Signals: acc.evidence(agyCan, nil),
		Note:    "Step counts from its conversation index, filed on the day each conversation was last touched — a long conversation lands whole on one day. No per-message history, model, cost, tokens or tools at this layer.",
	}
}

// agyConversations files one assistant entry per recorded step, at the
// conversation's last-modified time. The keep rules mirror the picker's:
// a row is a conversation when it has steps, a workspace and a time, and
// nested (subagent) runs stay out.
func agyConversations(db *sql.DB, acc *cliAcc) error {
	cols, err := clisession.SQLiteTableColumns(db, "conversation_summaries")
	if err != nil || !cols["conversation_id"] {
		return err
	}
	var sel []string
	for _, name := range []string{"conversation_id", "parent_conversation_id", "title", "step_count", "last_modified_time", "workspace_uris"} {
		if cols[name] {
			sel = append(sel, name)
		}
	}
	rows, err := db.Query("SELECT " + strings.Join(sel, ", ") + " FROM conversation_summaries")
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		row, ok := scanStrings(rows, sel)
		if !ok {
			continue
		}
		id := strings.TrimSpace(row["conversation_id"])
		if id == "" || strings.TrimSpace(row["parent_conversation_id"]) != "" {
			continue
		}
		steps := atoi64(row["step_count"])
		if steps <= 0 {
			continue
		}
		folder := agyFolder(row["workspace_uris"])
		if folder == "" {
			continue
		}
		at := agyAt(row["last_modified_time"])
		if at.IsZero() {
			continue
		}
		for i := int64(0); i < steps; i++ {
			acc.add(cliEntry{
				at: at, key: id, cwd: folder,
				name: strings.TrimSpace(row["title"]),
				role: "assistant",
			})
		}
	}
	return rows.Err()
}

// agyFolder reads the first folder out of workspace_uris, a JSON array of
// file:// URIs — the same rule the picker applies, so a conversation the
// meter counts is one the picker can show.
func agyFolder(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	var uris []string
	if json.Unmarshal([]byte(raw), &uris) != nil {
		uris = []string{raw}
	}
	for _, u := range uris {
		u = strings.TrimSpace(u)
		if u == "" {
			continue
		}
		if !strings.HasPrefix(u, "file://") {
			return filepath.Clean(u)
		}
		parsed, err := url.Parse(u)
		if err != nil || parsed.Path == "" {
			continue
		}
		return filepath.Clean(parsed.Path)
	}
	return ""
}

// agyAt reads the index's datetimes, which arrive either space-separated
// or as RFC3339Nano depending on the driver. The zero time of an
// unstarted conversation is rejected.
func agyAt(raw string) time.Time {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Time{}
	}
	for _, layout := range []string{time.RFC3339Nano, "2006-01-02 15:04:05.999999999-07:00"} {
		if at, err := time.Parse(layout, raw); err == nil && at.Year() >= 1970 {
			return at
		}
	}
	return time.Time{}
}
