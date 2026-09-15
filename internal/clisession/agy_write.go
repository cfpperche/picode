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

// Agy writer (Fatia 4 of docs/plans/launch-muse-agy.md). A spike against
// the real CLI (Antigravity 1.2.3, 2026-09-15) established the minimum a
// conversation needs to load under `--conversation`: a summaries row, the
// brain transcript, and an existing per-conversation store file — which
// may be an empty SQLite file; the schema never mattered. A planted
// transcript answered from its own first prompt, and a second turn
// continued on it, so the CLI maintains whatever else it needs.
//
// The transcript mirrors what the CLI itself appends: USER_INPUT entries
// carry the <USER_REQUEST> wrapper the reader unwraps, PLANNER_RESPONSE
// entries the assistant text. Tool traffic has no record type in this
// format, so both modes flatten it to text turns (calls as assistant,
// results as user), the way text mode works everywhere.
func (AgySource) Write(ctx context.Context, t transcript.Timeline, req WriteRequest) (Summary, error) {
	if strings.TrimSpace(req.Cwd) == "" {
		return Summary{}, fmt.Errorf("a folder is required")
	}
	_ = ctx
	t = t.Prepare()
	sid := strings.TrimSpace(req.SessionID)
	if sid == "" {
		sid = transcript.NewID()
	}
	now := req.resolvedNow()
	cwd := filepath.Clean(req.Cwd)
	logs := filepath.Join(agyBrainRoot(), sid, ".system_generated", "logs")
	path := filepath.Join(logs, "transcript.jsonl")

	name := title(t)
	if name == "" {
		name = "Handoff from " + t.Header.DisplayName()
	}
	var lines []string
	var emitted []transcript.Event
	step := 0
	lastInput := now
	emit := func(source, typ, text string, at time.Time) {
		entry := map[string]any{
			"step_index": step, "source": source, "type": typ,
			"status": "DONE", "created_at": at.UTC().Format(time.RFC3339), "content": text,
		}
		raw, _ := json.Marshal(entry)
		lines = append(lines, string(raw))
		step++
	}
	var firstPrompt string
	prompts := 0
	for _, e := range t.Events {
		at := e.Timestamp
		if at.IsZero() {
			at = now
		}
		switch e.Kind {
		case transcript.KindMessage:
			if e.Role != "user" {
				if e.Role != "assistant" || strings.TrimSpace(e.Text) == "" {
					continue
				}
				emit("MODEL", "PLANNER_RESPONSE", e.Text, at)
				emitted = append(emitted, transcript.Event{Kind: transcript.KindMessage, Role: "assistant"})
				continue
			}
			text := strings.TrimSpace(e.Text)
			if text == "" {
				continue
			}
			if firstPrompt == "" {
				firstPrompt = text
			}
			prompts++
			lastInput = at
			emit("USER_EXPLICIT", "USER_INPUT", "<USER_REQUEST>\n"+text+"\n</USER_REQUEST>", at)
			emitted = append(emitted, transcript.Event{Kind: transcript.KindMessage, Role: "user"})
		case transcript.KindToolCall:
			if e.Call == nil {
				continue
			}
			emit("MODEL", "PLANNER_RESPONSE", toolCallText(e.Call), at)
			emitted = append(emitted, transcript.Event{Kind: transcript.KindMessage, Role: "assistant"})
		case transcript.KindToolResult:
			if e.Result == nil {
				continue
			}
			emit("USER_EXPLICIT", "USER_INPUT", "<USER_REQUEST>\n"+toolResultText(e.Result)+"\n</USER_REQUEST>", at)
			emitted = append(emitted, transcript.Event{Kind: transcript.KindMessage, Role: "user"})
		}
	}
	counts := transcript.Timeline{Events: emitted}.Counts()
	if len(emitted) == 0 {
		return Summary{}, fmt.Errorf("nothing to hand off: the conversation has no turns")
	}
	if err := os.MkdirAll(logs, 0o755); err != nil {
		return Summary{}, err
	}
	if err := writeNewFile(path, []byte(strings.Join(lines, "\n")+"\n")); err != nil {
		_ = os.RemoveAll(filepath.Join(agyBrainRoot(), sid))
		return Summary{}, err
	}
	store := filepath.Join(agyConversationsDir(), sid+".db")
	if err := touchAgyStore(store); err != nil {
		_ = os.RemoveAll(filepath.Join(agyBrainRoot(), sid))
		return Summary{}, err
	}
	if err := insertAgySummaryRow(sid, cwd, name, firstPrompt, prompts, now, lastInput); err != nil {
		_ = os.RemoveAll(filepath.Join(agyBrainRoot(), sid))
		_ = os.Remove(store)
		return Summary{}, err
	}
	ref := Ref{ID: sid, Path: path, Cwd: cwd}
	if err := roundTrip(ctx, AgySource{}, ref, counts); err != nil {
		_ = os.RemoveAll(filepath.Join(agyBrainRoot(), sid))
		_ = os.Remove(store)
		removeAgySummaryRow(sid)
		return Summary{}, err
	}
	stamp := rfc3339(now)
	return Summary{
		CLI:        "agy",
		ID:         sid,
		Path:       store,
		ResumeArgs: []string{"--conversation", sid},
		Name:       clip(name, 120),
		Cwd:        cwd,
		CreatedAt:  stamp,
		UpdatedAt:  stamp,
		Preview:    preview(t),
		Messages:   counts.Messages,
	}, nil
}

// insertAgySummaryRow writes the index row `--conversation` resolves. Only
// the NOT NULL columns plus the listing's title/preview/steps are set;
// the rest keep the store defaults.
func insertAgySummaryRow(sid, cwd, name, first string, prompts int, now, lastInput time.Time) error {
	db, err := sql.Open("sqlite", "file:"+agyDBPath()+"?mode=rw")
	if err != nil {
		return err
	}
	defer db.Close()
	const layout = "2006-01-02 15:04:05.999999999-07:00"
	_, err = db.Exec(`INSERT INTO conversation_summaries (conversation_id, title,
		preview, step_count, last_modified_time, workspace_uris,
		last_user_input_time) VALUES (?,?,?,?,?,?,?)`,
		sid, name, first, prompts, now.Format(layout),
		`["file://`+cwd+`"]`, lastInput.Format(layout))
	return err
}

func removeAgySummaryRow(sid string) {
	db, err := sql.Open("sqlite", "file:"+agyDBPath()+"?mode=rw")
	if err != nil {
		return
	}
	defer db.Close()
	_, _ = db.Exec(`DELETE FROM conversation_summaries WHERE conversation_id=?`, sid)
}

// touchAgyStore creates the per-conversation store file when missing. The
// CLI only needs it to exist — an empty SQLite file loads (spike).
func touchAgyStore(path string) error {
	if _, err := os.Stat(path); err == nil {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	db, err := sql.Open("sqlite", "file:"+path)
	if err != nil {
		return err
	}
	defer db.Close()
	// Open is lazy: the pragma forces the file header so the store file
	// exists with no schema, the shape the spike proved sufficient.
	_, err = db.Exec(`PRAGMA user_version = 0`)
	return err
}
