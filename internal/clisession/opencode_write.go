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

// OpenCode writer (ADR-0094): OpenCode ships `opencode import <file>`, the
// inverse of `opencode export <id>`, so PiCode hands the conversation to
// the CLI's own importer instead of writing rows into a SQLite store the
// running CLI may hold open. The file is the export envelope — {info,
// messages:[{info, parts}]} — verified against a real export (OpenCode
// 1.18.29, 2026-09). The import files the session under the folder it is
// run in and keeps the ids the file declares, so the caller runs it in the
// session's folder and the round-trip read confirms what landed.

// opencodeID mints an id in OpenCode's shape: a three-letter kind prefix
// and 26 alphanumeric characters.
func opencodeID(prefix string) string { return prefix + "_" + transcript.RandomAlnum(26) }

// opencodeNewestVersion reads the `version` of the most recently updated
// session in the local store — the format marker the installed CLI wrote.
func opencodeNewestVersion() string {
	path := opencodeDBPath()
	if path == "" {
		return ""
	}
	if _, err := os.Stat(path); err != nil {
		return ""
	}
	db, err := sql.Open("sqlite", "file:"+path+"?mode=ro&_pragma=busy_timeout(1000)")
	if err != nil {
		return ""
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	cols, err := sqliteTableColumns(db, "session")
	if err != nil || !cols["version"] {
		return ""
	}
	var v sql.NullString
	if db.QueryRow(`SELECT version FROM session WHERE version <> '' ORDER BY time_updated DESC LIMIT 1`).Scan(&v) != nil {
		return ""
	}
	return strings.TrimSpace(v.String)
}

func (OpenCodeSource) Write(ctx context.Context, t transcript.Timeline, req WriteRequest) (Summary, error) {
	if strings.TrimSpace(req.Cwd) == "" {
		return Summary{}, fmt.Errorf("a folder is required")
	}
	if req.Run == nil {
		return Summary{}, ErrNoRunner
	}
	version := strings.TrimSpace(req.FormatVersion)
	if version == "" {
		version = opencodeNewestVersion()
	}
	if version == "" {
		return Summary{}, ErrUnknownFormat
	}
	t = t.Prepare()
	sid := req.SessionID
	if sid == "" || !strings.HasPrefix(sid, "ses_") {
		sid = opencodeID("ses")
	}
	now := req.resolvedNow()
	cwd := filepath.Clean(req.Cwd)
	model := t.Header.Model
	provider := t.Header.Provider
	if provider == "" {
		provider = t.Header.SourceCLI
	}

	type part map[string]any
	type message struct {
		Info  map[string]any `json:"info"`
		Parts []part         `json:"parts"`
	}
	var messages []message
	var emitted []transcript.Event
	seq := 0
	ms := func(at time.Time) int64 {
		seq++
		if at.IsZero() {
			return now.Add(time.Duration(seq) * time.Millisecond).UnixMilli()
		}
		return at.UnixMilli()
	}
	firstUserID := ""
	add := func(role string, at time.Time, parts []part, extra map[string]any) {
		if len(parts) == 0 {
			return
		}
		mid := opencodeID("msg")
		created := ms(at)
		info := map[string]any{
			"id":        mid,
			"sessionID": sid,
			"role":      role,
			"agent":     "build",
			"time":      map[string]any{"created": created},
		}
		if role == "assistant" {
			info["mode"] = "build"
			info["modelID"] = model
			info["providerID"] = provider
			info["path"] = map[string]any{"cwd": cwd, "root": cwd}
			info["cost"] = 0
			info["tokens"] = map[string]any{"total": 0, "input": 0, "output": 0, "reasoning": 0, "cache": map[string]any{"read": 0, "write": 0}}
			info["time"] = map[string]any{"created": created, "completed": created}
			info["finish"] = "stop"
			if firstUserID != "" {
				info["parentID"] = firstUserID
			}
		} else {
			info["model"] = map[string]any{"providerID": provider, "modelID": model}
			if firstUserID == "" {
				firstUserID = mid
			}
		}
		for k, v := range extra {
			info[k] = v
		}
		for i := range parts {
			parts[i]["id"] = opencodeID("prt")
			parts[i]["sessionID"] = sid
			parts[i]["messageID"] = mid
		}
		messages = append(messages, message{Info: info, Parts: parts})
	}
	textPart := func(text string) part { return part{"type": "text", "text": text} }

	walkTurns(t.Events,
		func(e transcript.Event) {
			add("user", e.Timestamp, []part{textPart(e.Text)}, nil)
			emitted = append(emitted, transcript.Event{Kind: transcript.KindMessage, Role: "user"})
		},
		func(turn assistantTurn) {
			var parts []part
			for _, blk := range turn.Blocks {
				switch blk.Kind {
				case transcript.KindMessage:
					parts = append(parts, textPart(blk.Text))
					emitted = append(emitted, transcript.Event{Kind: transcript.KindMessage, Role: "assistant"})
				case transcript.KindToolCall:
					if req.textTools() {
						parts = append(parts, textPart(toolCallText(blk.Call)))
						emitted = append(emitted, transcript.Event{Kind: transcript.KindMessage, Role: "assistant"})
						continue
					}
					// The result rides in the same part, so it is attached
					// when the matching tool_result is walked.
					parts = append(parts, part{
						"type":   "tool",
						"tool":   blk.Call.Name,
						"callID": blk.Call.ID,
						"state": map[string]any{
							"status": "completed",
							"input":  blk.Call.ObjectInput(),
							"output": "",
							"title":  clip(blk.Call.Name+" "+blk.Call.InputString(), 80),
							"time":   map[string]any{"start": ms(blk.Timestamp), "end": ms(blk.Timestamp)},
						},
					})
					emitted = append(emitted, transcript.Event{Kind: transcript.KindToolCall})
				}
			}
			add("assistant", turn.Timestamp, parts, nil)
		},
		func(e transcript.Event) {
			if req.textTools() {
				add("user", e.Timestamp, []part{textPart(toolResultText(e.Result))}, nil)
				emitted = append(emitted, transcript.Event{Kind: transcript.KindMessage, Role: "user"})
				return
			}
			// Attach the output to the tool part that issued the call.
			for mi := len(messages) - 1; mi >= 0; mi-- {
				for pi := range messages[mi].Parts {
					p := messages[mi].Parts[pi]
					if p["type"] != "tool" || p["callID"] != e.Result.CallID {
						continue
					}
					state, _ := p["state"].(map[string]any)
					if state == nil {
						continue
					}
					state["output"] = e.Result.Text
					if e.Result.IsError {
						state["status"] = "error"
						state["error"] = e.Result.Text
					}
					emitted = append(emitted, transcript.Event{Kind: transcript.KindToolResult})
					return
				}
			}
			// A result with no call in this file still travels as text.
			add("user", e.Timestamp, []part{textPart(toolResultText(e.Result))}, nil)
			emitted = append(emitted, transcript.Event{Kind: transcript.KindMessage, Role: "user"})
		},
	)
	if len(emitted) == 0 {
		return Summary{}, fmt.Errorf("nothing to hand off: the conversation has no turns")
	}

	name := title(t)
	if name == "" {
		name = "Handoff from " + t.Header.DisplayName()
	}
	envelope := map[string]any{
		"info": map[string]any{
			"id":        sid,
			"slug":      strings.ToLower(transcript.RandomAlnum(10)),
			"projectID": "global",
			"directory": cwd,
			"path":      strings.TrimPrefix(cwd, "/"),
			"title":     name,
			"agent":     "build",
			"model":     map[string]any{"id": model, "providerID": provider},
			"version":   version,
			"summary":   map[string]any{"additions": 0, "deletions": 0, "files": 0},
			"cost":      0,
			"tokens":    map[string]any{"input": 0, "output": 0, "reasoning": 0, "cache": map[string]any{"read": 0, "write": 0}},
			"time":      map[string]any{"created": now.UnixMilli(), "updated": now.UnixMilli()},
		},
		"messages": messages,
	}
	body, err := json.Marshal(envelope)
	if err != nil {
		return Summary{}, err
	}
	file, err := os.CreateTemp("", "picode-handoff-*.json")
	if err != nil {
		return Summary{}, err
	}
	tmp := file.Name()
	defer os.Remove(tmp)
	if _, err := file.Write(body); err != nil {
		file.Close()
		return Summary{}, err
	}
	if err := file.Close(); err != nil {
		return Summary{}, err
	}
	if out, err := req.Run(ctx, "import", tmp); err != nil {
		return Summary{}, fmt.Errorf("opencode import failed: %w: %s", err, clip(string(out), 200))
	}
	ref := Ref{ID: sid, Cwd: cwd}
	if err := roundTrip(ctx, OpenCodeSource{}, ref, transcript.Timeline{Events: emitted}.Counts()); err != nil {
		return Summary{}, err
	}
	return Summary{
		CLI:        "opencode",
		ID:         sid,
		Path:       opencodeDBPath(),
		ResumeArgs: []string{"--session", sid},
		Name:       clip(name, 120),
		Cwd:        cwd,
		CreatedAt:  rfc3339(now),
		UpdatedAt:  rfc3339(now),
		Preview:    preview(t),
		Messages:   transcript.Timeline{Events: emitted}.Counts().Messages,
		Model:      model,
	}, nil
}
