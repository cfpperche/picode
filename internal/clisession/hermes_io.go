package clisession

import (
	"context"
	"database/sql"
	"encoding/json"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/cfpperche/picode/internal/transcript"
)

// Hermes Agent state.db → timeline (ADR-0088). Read-only, over the same
// column-tolerant SQLite access HermesSource.List uses. Schema verified
// against a real installation (Hermes Agent 0.18.2, 2026-09):
//
//	messages(session_id, role, content, tool_call_id, tool_calls, tool_name, timestamp, reasoning, active, compacted)
//	  role user|assistant|tool|system; tool_calls is an OpenAI-style array
//	  [{id, call_id, type:"function", function:{name, arguments:<JSON string>}}]
//
// Hermes has no native import and no interactive launch with an initial
// prompt (`hermes --help`: only -z one-shot and `chat -q`), so it is a
// source only: it implements Reader, not Writer or Prompter.
func (HermesSource) Read(ctx context.Context, ref Ref) (transcript.Timeline, error) {
	path := hermesStatePath()
	if path == "" || ref.ID == "" {
		return transcript.Timeline{}, ErrNotUnderRoot
	}
	if ref.Path != "" && ref.Path != path {
		return transcript.Timeline{}, ErrNotUnderRoot
	}
	if _, err := os.Stat(path); err != nil {
		return transcript.Timeline{}, ErrNotUnderRoot
	}
	db, err := sql.Open("sqlite", "file:"+path+"?mode=ro&_pragma=busy_timeout(1000)")
	if err != nil {
		return transcript.Timeline{}, err
	}
	defer db.Close()
	db.SetMaxOpenConns(1)

	t := transcript.Timeline{Header: transcript.Header{SourceCLI: "hermes", SourcePath: path, SourceID: ref.ID, Cwd: ref.Cwd}}
	scols, err := hermesSessionColumns(db)
	if err != nil {
		return transcript.Timeline{}, err
	}
	if scols["id"] {
		sel := []string{"id"}
		for _, c := range []string{"title", "model", "cwd", "git_repo_root", "started_at", "ended_at"} {
			if scols[c] {
				sel = append(sel, c)
			}
		}
		row := db.QueryRowContext(ctx, "SELECT "+strings.Join(sel, ", ")+" FROM sessions WHERE id = ?", ref.ID)
		dest := make([]any, len(sel))
		vals := make([]sql.NullString, len(sel))
		for i := range dest {
			dest[i] = &vals[i]
		}
		if err := row.Scan(dest...); err != nil {
			if err == sql.ErrNoRows {
				return transcript.Timeline{}, ErrNotUnderRoot
			}
			return transcript.Timeline{}, err
		}
		for i, c := range sel {
			v := vals[i].String
			switch c {
			case "title":
				t.Header.Title = clip(v, 120)
			case "model":
				t.Header.Model = v
			case "cwd":
				if v != "" {
					t.Header.Cwd = v
				}
			case "git_repo_root":
				if t.Header.Cwd == "" {
					t.Header.Cwd = v
				}
			case "started_at":
				t.Header.CreatedAt = hermesTime(v)
			case "ended_at":
				t.Header.UpdatedAt = hermesTime(v)
			}
		}
	}

	mcols, err := hermesColumns(db, "messages")
	if err != nil {
		return transcript.Timeline{}, err
	}
	if !mcols["session_id"] || !mcols["role"] {
		return transcript.Timeline{}, ErrUnknownFormat
	}
	sel := []string{"role"}
	for _, c := range []string{"content", "tool_call_id", "tool_calls", "tool_name", "timestamp", "active", "compacted"} {
		if mcols[c] {
			sel = append(sel, c)
		}
	}
	rows, err := db.QueryContext(ctx, "SELECT "+strings.Join(sel, ", ")+" FROM messages WHERE session_id = ? ORDER BY id", ref.ID)
	if err != nil {
		return transcript.Timeline{}, err
	}
	defer rows.Close()
	group := 0
	names := map[string]string{}
	for rows.Next() {
		vals := make([]sql.NullString, len(sel))
		dest := make([]any, len(sel))
		for i := range dest {
			dest[i] = &vals[i]
		}
		if err := rows.Scan(dest...); err != nil {
			return transcript.Timeline{}, err
		}
		get := func(name string) string {
			for i, c := range sel {
				if c == name {
					return vals[i].String
				}
			}
			return ""
		}
		if get("active") == "0" {
			t.Manifest.Drop("hermes.inactive")
			continue
		}
		if get("compacted") == "1" {
			t.Manifest.Drop("hermes.compacted")
			continue
		}
		group++
		ts := hermesTime(get("timestamp"))
		if !ts.IsZero() {
			t.Header.UpdatedAt = ts
		}
		content := strings.TrimSpace(get("content"))
		switch get("role") {
		case "system":
			t.Events = append(t.Events, transcript.Event{Kind: transcript.KindContext, Role: "system", Text: content, Timestamp: ts, Group: group})
		case "user":
			if content != "" {
				t.Events = append(t.Events, transcript.Event{Kind: transcript.KindMessage, Role: "user", Text: content, Timestamp: ts, Group: group})
			}
		case "assistant":
			if content != "" {
				t.Events = append(t.Events, transcript.Event{Kind: transcript.KindMessage, Role: "assistant", Text: content, Model: t.Header.Model, Timestamp: ts, Group: group})
			}
			if raw := get("tool_calls"); raw != "" {
				var calls []struct {
					ID       string `json:"id"`
					CallID   string `json:"call_id"`
					Function struct {
						Name      string          `json:"name"`
						Arguments json.RawMessage `json:"arguments"`
					} `json:"function"`
				}
				if json.Unmarshal([]byte(raw), &calls) != nil {
					t.Manifest.Drop("hermes.tool_calls.malformed")
					continue
				}
				for _, c := range calls {
					id := c.CallID
					if id == "" {
						id = c.ID
					}
					names[id] = c.Function.Name
					t.Events = append(t.Events, transcript.Event{Kind: transcript.KindToolCall, Role: "assistant", Call: &transcript.ToolCall{ID: id, Name: c.Function.Name, Input: argumentsJSON(c.Function.Arguments)}, Model: t.Header.Model, Timestamp: ts, Group: group})
				}
			}
		case "tool":
			id := get("tool_call_id")
			name := get("tool_name")
			if name == "" {
				name = names[id]
			}
			t.Events = append(t.Events, transcript.Event{Kind: transcript.KindToolResult, Result: &transcript.ToolResult{CallID: id, Name: name, Text: content}, Timestamp: ts, Group: group})
		default:
			t.Manifest.Drop("hermes.role." + get("role"))
		}
	}
	return t, rows.Err()
}

func hermesColumns(db *sql.DB, table string) (map[string]bool, error) {
	rows, err := db.Query("PRAGMA table_info(" + table + ")")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]bool{}
	for rows.Next() {
		var cid int
		var name, ctype string
		var notnull, pk int
		var dflt sql.NullString
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk); err != nil {
			return nil, err
		}
		out[name] = true
	}
	return out, nil
}

// hermesTime reads Hermes' REAL epoch-seconds columns.
func hermesTime(s string) time.Time {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil || f <= 0 {
		return time.Time{}
	}
	sec := int64(f)
	return time.Unix(sec, int64((f-float64(sec))*1e9)).UTC()
}
