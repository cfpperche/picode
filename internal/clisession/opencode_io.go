package clisession

import (
	"context"
	"database/sql"
	"encoding/json"
	"os"
	"strings"
	"time"

	"github.com/cfpperche/picode/internal/transcript"
)

// OpenCode store → timeline (ADR-0088). Read-only over the same SQLite
// access OpenCodeSource.List uses. Schema verified against a real
// installation (OpenCode 1.18.29, 2026-09):
//
//	message(id, session_id, time_created, data)   data: {role:user|assistant, time:{created},
//	                                              model:{providerID,modelID}, modelID, providerID}
//	part(id, message_id, session_id, time_created, data)
//	  {type:"text", text}                          a turn's prose
//	  {type:"reasoning", text}                     visible thinking
//	  {type:"tool", tool, callID, state:{status, input, output, metadata, title, time}}
//	  {type:"step-start"|"step-finish"|"patch"|"snapshot"|"file"}   runtime state, counted
//
// A tool part carries the call and its result together, so one part
// becomes a tool_call plus the tool_result that answers it.
func (OpenCodeSource) Read(ctx context.Context, ref Ref) (transcript.Timeline, error) {
	path := opencodeDBPath()
	if path == "" || strings.TrimSpace(ref.ID) == "" {
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

	t := transcript.Timeline{Header: transcript.Header{SourceCLI: "opencode", SourcePath: path, SourceID: ref.ID, Cwd: ref.Cwd}}
	if err := opencodeHeader(ctx, db, ref.ID, &t); err != nil {
		return transcript.Timeline{}, err
	}
	rows, err := db.QueryContext(ctx, `SELECT m.data, p.data FROM message m
		LEFT JOIN part p ON p.message_id = m.id
		WHERE m.session_id = ?
		ORDER BY m.time_created, m.id, p.time_created, p.id`, ref.ID)
	if err != nil {
		return transcript.Timeline{}, err
	}
	defer rows.Close()

	group := 0
	lastMessage := ""
	role, model := "", ""
	for rows.Next() {
		var mRaw string
		var pRaw sql.NullString
		if err := rows.Scan(&mRaw, &pRaw); err != nil {
			return transcript.Timeline{}, err
		}
		if mRaw != lastMessage {
			lastMessage = mRaw
			group++
			role, model = opencodeMessageFacts(mRaw, &t)
		}
		if !pRaw.Valid || strings.TrimSpace(pRaw.String) == "" {
			continue
		}
		opencodePart(pRaw.String, role, model, group, &t)
	}
	return t, rows.Err()
}

// opencodeHeader fills the timeline header from the session row, refusing
// a session id this store does not have.
func opencodeHeader(ctx context.Context, db *sql.DB, id string, t *transcript.Timeline) error {
	cols, err := sqliteTableColumns(db, "session")
	if err != nil {
		return err
	}
	if !cols["id"] {
		return ErrUnknownFormat
	}
	sel := []string{"id"}
	for _, name := range []string{"title", "model", "directory", "version", "time_created", "time_updated"} {
		if cols[name] {
			sel = append(sel, name)
		}
	}
	vals := make([]sql.NullString, len(sel))
	dest := make([]any, len(sel))
	for i := range dest {
		dest[i] = &vals[i]
	}
	if err := db.QueryRowContext(ctx, "SELECT "+strings.Join(sel, ", ")+" FROM session WHERE id = ?", id).Scan(dest...); err != nil {
		if err == sql.ErrNoRows {
			return ErrNotUnderRoot
		}
		return err
	}
	for i, name := range sel {
		v := strings.TrimSpace(vals[i].String)
		switch name {
		case "title":
			t.Header.Title = clip(v, 120)
		case "model":
			t.Header.Model = opencodeModel(v)
		case "directory":
			if v != "" {
				t.Header.Cwd = v
			}
		case "version":
			t.Header.FormatVersion = v
		case "time_created":
			t.Header.CreatedAt = opencodeTime(v)
		case "time_updated":
			t.Header.UpdatedAt = opencodeTime(v)
		}
	}
	return nil
}

// opencodeMessageFacts reads one message envelope: its role and the model
// that produced it.
func opencodeMessageFacts(raw string, t *transcript.Timeline) (role, model string) {
	var m struct {
		Role     string `json:"role"`
		ModelID  string `json:"modelID"`
		Provider string `json:"providerID"`
		Model    struct {
			ModelID    string `json:"modelID"`
			ID         string `json:"id"`
			ProviderID string `json:"providerID"`
		} `json:"model"`
	}
	if json.Unmarshal([]byte(raw), &m) != nil {
		t.Manifest.Drop("opencode.malformed")
		return "", ""
	}
	model = m.ModelID
	if model == "" {
		model = m.Model.ModelID
	}
	if model == "" {
		model = m.Model.ID
	}
	if model != "" && t.Header.Model == "" {
		t.Header.Model = model
	}
	if t.Header.Provider == "" {
		t.Header.Provider = m.Provider
		if t.Header.Provider == "" {
			t.Header.Provider = m.Model.ProviderID
		}
	}
	return m.Role, model
}

// opencodePart turns one part into timeline events.
func opencodePart(raw, role, model string, group int, t *transcript.Timeline) {
	var p struct {
		Type   string `json:"type"`
		Text   string `json:"text"`
		Tool   string `json:"tool"`
		CallID string `json:"callID"`
		Time   struct {
			Start float64 `json:"start"`
		} `json:"time"`
		State struct {
			Status string          `json:"status"`
			Input  json.RawMessage `json:"input"`
			Output string          `json:"output"`
			Error  string          `json:"error"`
			Title  string          `json:"title"`
		} `json:"state"`
	}
	if json.Unmarshal([]byte(raw), &p) != nil {
		t.Manifest.Drop("opencode.malformed")
		return
	}
	at := time.Time{}
	if p.Time.Start > 0 {
		at = time.UnixMilli(int64(p.Time.Start)).UTC()
	}
	switch p.Type {
	case "text":
		text := strings.TrimSpace(p.Text)
		if text == "" {
			return
		}
		r := role
		if r == "" {
			r = "user"
		}
		t.Events = append(t.Events, transcript.Event{Kind: transcript.KindMessage, Role: r, Text: text, Model: model, Timestamp: at, Group: group})
	case "reasoning":
		if strings.TrimSpace(p.Text) != "" {
			t.Events = append(t.Events, transcript.Event{Kind: transcript.KindThinking, Role: "assistant", Text: p.Text, Timestamp: at, Group: group})
		}
	case "tool":
		id := p.CallID
		if id == "" {
			id = p.Tool + "-" + p.State.Title
		}
		t.Events = append(t.Events, transcript.Event{Kind: transcript.KindToolCall, Role: "assistant", Call: &transcript.ToolCall{ID: id, Name: p.Tool, Input: argumentsJSON(p.State.Input)}, Model: model, Timestamp: at, Group: group})
		out := p.State.Output
		isErr := p.State.Status == "error"
		if isErr && strings.TrimSpace(out) == "" {
			out = p.State.Error
		}
		if p.State.Status == "pending" || p.State.Status == "running" {
			// The call never finished; Repair synthesizes the missing
			// answer rather than inventing an outcome here.
			return
		}
		t.Events = append(t.Events, transcript.Event{Kind: transcript.KindToolResult, Result: &transcript.ToolResult{CallID: id, Name: p.Tool, Text: out, IsError: isErr}, Timestamp: at, Group: group})
	default:
		t.Manifest.Drop("opencode." + p.Type)
	}
}

// opencodeTime reads OpenCode's integer epoch columns (milliseconds).
func opencodeTime(raw string) time.Time {
	var v sql.NullFloat64
	if err := v.Scan(raw); err != nil {
		return time.Time{}
	}
	s := unixEpoch(v)
	if s == "" {
		return time.Time{}
	}
	return parseTime(s)
}

// ForkArgs: `opencode --session <id> --fork --prompt <text>` — opencode
// --help 1.18 ("fork the session when continuing"). The copy's id is the
// CLI's to choose.
func (OpenCodeSource) ForkArgs(src Ref, prompt, _ string) Fork {
	args := []string{"--session", src.ID, "--fork"}
	if prompt != "" {
		args = append(args, "--prompt", prompt)
	}
	return Fork{Args: args}
}

// PromptArgs: `opencode --prompt <text>` — verified flags (opencode
// --help, 1.18.29). A new conversation cannot be given an id up front
// (`--session` continues an existing one), so sessionID is ignored and
// the lineage resolves through the terminal's pinned session.
func (OpenCodeSource) PromptArgs(prompt, _ string) []string {
	return []string{"--prompt", prompt}
}
