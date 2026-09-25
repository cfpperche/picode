package clisession

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/cfpperche/picode/internal/transcript"
)

// Muse writer (Fatia 4 of docs/plans/launch-muse-agy.md). A spike against
// the real CLI (Muse Code 1.3.0, 2026-09-15) established the minimum a
// session needs to list, export and resume: an index row plus a session
// log with plain records — one metadata record, one user_intent.accepted
// per user turn, assistant_message_committed plus the tool batches, in
// stream order. A synthetic one-turn session built exactly this way
// exports cleanly and `resume` resolves it (the permission frame is
// deliberately absent: there is no audit trail to retain, so resume notes
// "retained permission history is incomplete" and opens with defaults).
//
// Two sharp edges, both measured: the --session argument must parse as a
// UUID (anything else is rejected before the store lookup), and approval
// records carry model as an object, which the reader decodes late.
// museLogRecord is one session-log line in wire order. Do not reorder
// the fields and do not switch back to a map: see record().
type museLogRecord struct {
	SchemaVersion        int            `json:"schema_version"`
	ID                   string         `json:"id"`
	Stream               museLogStream  `json:"stream"`
	Sequence             int            `json:"sequence"`
	RecordedAt           int64          `json:"recorded_at"`
	RecordType           string         `json:"record_type"`
	Durability           string         `json:"durability"`
	CausationID          any            `json:"causation_id"`
	PayloadType          string         `json:"payload_type"`
	PayloadSchemaVersion int            `json:"payload_schema_version"`
	Payload              map[string]any `json:"payload"`
}

type museLogStream struct {
	Kind string `json:"kind"`
	ID   string `json:"id"`
}

// museLogLines renders the session log: plain records in stream order,
// the minimum the exporter and resume accept (spike, Fatia 4). It returns
// the log lines, the beats they carry (for the round trip), and the row
// facts (title, first prompt, user-turn count).
func museLogLines(t transcript.Timeline, sid, cwd string, now time.Time, textTools bool) ([]string, []transcript.Event, string, string, int) {
	var lines []string
	var emitted []transcript.Event
	seq := 0
	at := now.UnixMicro()
	next := func() (int64, int) {
		seq++
		at++
		return at, seq
	}
	record := func(payloadType string, payload map[string]any) {
		stamp, n := next()
		// Field order is the wire order: the exporter dispatches on the
		// line prefix, so schema_version must lead (a map's alphabetical
		// order reads as unparseable_line). encoding/json keeps struct
		// declaration order; the payload stays a map.
		env := museLogRecord{
			SchemaVersion: 1, ID: transcript.NewID(),
			Stream:   museLogStream{Kind: "session", ID: sid},
			Sequence: n, RecordedAt: stamp,
			RecordType: "event", Durability: "durable",
			PayloadType: payloadType, PayloadSchemaVersion: 1, Payload: payload,
		}
		raw, _ := json.Marshal(env)
		lines = append(lines, string(raw))
	}
	name := title(t)
	if name == "" {
		name = "Handoff from " + t.Header.DisplayName()
	}
	var firstPrompt string
	prompts := 0
	var texts []string
	var calls []map[string]any
	var results []map[string]any
	flush := func() {
		if len(texts) == 0 && len(calls) == 0 && len(results) == 0 {
			return
		}
		mid := transcript.NewID()
		for _, text := range texts {
			record("runtime.session", map[string]any{"kind": "run", "run_id": mid,
				"event": map[string]any{"kind": "assistant_message_committed", "message_id": transcript.NewID(), "text": text}})
			emitted = append(emitted, transcript.Event{Kind: transcript.KindMessage, Role: "assistant"})
		}
		if len(calls) > 0 {
			record("runtime.session", map[string]any{"kind": "run", "run_id": mid,
				"event": map[string]any{"kind": "assistant_tool_calls_committed", "tool_calls": calls}})
			for range calls {
				emitted = append(emitted, transcript.Event{Kind: transcript.KindToolCall})
			}
		}
		if len(results) > 0 {
			record("runtime.session", map[string]any{"kind": "run", "run_id": mid,
				"event": map[string]any{"kind": "tool_result_batch_committed", "batch_id": mid, "results": results}})
			for range results {
				emitted = append(emitted, transcript.Event{Kind: transcript.KindToolResult})
			}
		}
		texts, calls, results = nil, nil, nil
	}
	record("runtime.session.metadata", map[string]any{"kind": "metadata", "record": map[string]any{"workspace_root": cwd}})
	for _, e := range t.Events {
		switch e.Kind {
		case transcript.KindMessage:
			if e.Role != "user" {
				if e.Role != "assistant" || strings.TrimSpace(e.Text) == "" {
					continue
				}
				texts = append(texts, e.Text)
				continue
			}
			flush()
			text := strings.TrimSpace(e.Text)
			if text == "" {
				continue
			}
			if firstPrompt == "" {
				firstPrompt = text
			}
			prompts++
			record("runtime.user_intent.accepted", map[string]any{
				"intent_id":         transcript.NewID(),
				"model_messages":    []any{map[string]any{"content": []any{map[string]any{"kind": "text", "text": text}}}},
				"source_session_id": sid,
			})
			emitted = append(emitted, transcript.Event{Kind: transcript.KindMessage, Role: "user"})
		case transcript.KindToolCall:
			if e.Call == nil {
				continue
			}
			if textTools {
				texts = append(texts, toolCallText(e.Call))
				continue
			}
			id := e.Call.ID
			if id == "" {
				id = "call_" + strings.ReplaceAll(transcript.NewID()[:8], "-", "")
			}
			calls = append(calls, map[string]any{"call_id": id, "name": e.Call.Name, "args": string(e.Call.Input)})
		case transcript.KindToolResult:
			if e.Result == nil {
				continue
			}
			if textTools {
				flush()
				record("runtime.user_intent.accepted", map[string]any{
					"intent_id":         transcript.NewID(),
					"model_messages":    []any{map[string]any{"content": []any{map[string]any{"kind": "text", "text": toolResultText(e.Result)}}}},
					"source_session_id": sid,
				})
				emitted = append(emitted, transcript.Event{Kind: transcript.KindMessage, Role: "user"})
				continue
			}
			results = append(results, map[string]any{"tool_call_id": e.Result.CallID, "text": e.Result.Text})
		}
	}
	flush()
	return lines, emitted, name, firstPrompt, prompts
}

func (MuseSource) Write(ctx context.Context, t transcript.Timeline, req WriteRequest) (Summary, error) {
	if strings.TrimSpace(req.Cwd) == "" {
		return Summary{}, fmt.Errorf("a folder is required")
	}
	t = t.Prepare()
	sid := strings.TrimSpace(req.SessionID)
	if sid == "" {
		sid = transcript.NewID()
	}
	// The CLI rejects anything that does not parse as a UUID before the
	// store lookup, so a foreign id shape must fail here — not as an
	// unresolvable session nobody can open.
	if !validMuseID(sid) {
		return Summary{}, fmt.Errorf("not a Muse session id: %q", sid)
	}
	now := req.resolvedNow()
	cwd := filepath.Clean(req.Cwd)
	dir := filepath.Join(museSessionsRoot(), now.Format("2006/01/02"), sid)
	log := filepath.Join(dir, "session.jsonl")

	lines, emitted, name, firstPrompt, prompts := museLogLines(t, sid, cwd, now, req.textTools())
	counts := transcript.Timeline{Events: emitted}.Counts()
	if len(emitted) == 0 {
		return Summary{}, fmt.Errorf("nothing to hand off: the conversation has no turns")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return Summary{}, err
	}
	if err := writeNewFile(log, []byte(strings.Join(lines, "\n")+"\n")); err != nil {
		_ = os.RemoveAll(dir)
		return Summary{}, err
	}
	if err := insertMuseIndexRow(sid, dir, log, cwd, name, firstPrompt, prompts, now); err != nil {
		_ = os.RemoveAll(dir)
		return Summary{}, err
	}
	ref := Ref{ID: sid, Path: log, Cwd: cwd}
	if err := roundTrip(ctx, MuseSource{}, ref, counts); err != nil {
		_ = os.RemoveAll(dir)
		removeMuseIndexRow(sid)
		return Summary{}, err
	}
	stamp := rfc3339(now)
	return Summary{
		CLI:        "muse",
		ID:         sid,
		Path:       log,
		ResumeArgs: []string{"resume", sid},
		Name:       clip(name, 120),
		Cwd:        cwd,
		CreatedAt:  stamp,
		UpdatedAt:  stamp,
		Preview:    preview(t),
		Messages:   counts.Messages,
	}, nil
}

// insertMuseIndexRow writes the row `resume` and `export` resolve. Only
// the NOT NULL columns plus the ones the picker renders are set; the rest
// default NULL, the way sessions without lineage read. search_text mimics
// the CLI's own unit-separated shape.
func insertMuseIndexRow(sid, dir, log, cwd, name, first string, prompts int, now time.Time) error {
	db, err := sql.Open("sqlite", "file:"+museDBPath()+"?mode=rw")
	if err != nil {
		return err
	}
	defer db.Close()
	us := now.UnixMicro()
	lower := strings.ToLower(strings.Join([]string{name, first}, " "))
	search := strings.Join([]string{sid, sid[:8], strings.ToLower(name), "valid", lower, cwd}, "\x1f")
	_, err = db.Exec(`INSERT INTO sessions (session_id, session_stream_id, session_dir,
		session_log_path, layout, workspace_root, workspace_key, provider_id,
		title, first_user_prompt, search_text, prompt_count,
		created_at_us, updated_at_us, indexed_at_us, status, status_rank,
		latest_segment_terminated) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		sid, sid, dir, log, "session_jsonl", cwd, cwd, "meta",
		name, first, search, prompts, us, us, us, "valid", 0, 1)
	return err
}

func removeMuseIndexRow(sid string) {
	db, err := sql.Open("sqlite", "file:"+museDBPath()+"?mode=rw")
	if err != nil {
		return
	}
	defer db.Close()
	_, _ = db.Exec(`DELETE FROM sessions WHERE session_id=?`, sid)
}

// validMuseID is the UUID shape `export` and `resume` parse: 36 hex and
// dashes. transcript.NewID always qualifies.
func validMuseID(s string) bool {
	if len(s) != 36 {
		return false
	}
	for i := 0; i < 36; i++ {
		c := s[i]
		if i == 8 || i == 13 || i == 18 || i == 23 {
			if c != '-' {
				return false
			}
			continue
		}
		if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f' || c >= 'A' && c <= 'F') {
			return false
		}
	}
	return true
}
