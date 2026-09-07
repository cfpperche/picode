package clisession

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/cfpperche/picode/internal/session"
	"github.com/cfpperche/picode/internal/transcript"
)

// pi writer (ADR-0087): creates <req.Dir>/<ISO stamp>_<uuid>.jsonl in pi's
// session version 3 — a session header, then id/parentId-chained entries:
// session_info for the title, message entries for user turns, assistant
// turns (text and toolCall blocks in one message) and toolResult messages.
// The caller chooses the directory (the server passes the adopting
// agent's private dir, ADR-0040); pi opens the file through --session.
// Thinking is never written. The file is re-read through PISource.Read
// before the write counts as done.

func (PISource) Write(ctx context.Context, t transcript.Timeline, req WriteRequest) (Summary, error) {
	if strings.TrimSpace(req.Cwd) == "" {
		return Summary{}, fmt.Errorf("a folder is required")
	}
	if strings.TrimSpace(req.Dir) == "" {
		return Summary{}, ErrNoDir
	}
	// pi's session format is version 3 whatever the CLI version is, so the
	// installed version (req.FormatVersion) carries no information here.
	t = t.Prepare()
	sid := req.SessionID
	if sid == "" {
		sid = transcript.NewID()
	}
	now := req.resolvedNow()
	cwd := filepath.Clean(req.Cwd)
	path := filepath.Join(req.Dir, now.Format("2006-01-02T15-04-05-000Z")+"_"+sid+".jsonl")

	const tsLayout = "2006-01-02T15:04:05.000Z"
	var b strings.Builder
	var emitted []transcript.Event
	var werr error
	prev := ""
	seq := 0
	entry := func(at time.Time, fields map[string]any) {
		if werr != nil {
			return
		}
		seq++
		id := transcript.RandomHex(4)
		fields["id"] = id
		if prev == "" {
			fields["parentId"] = nil
		} else {
			fields["parentId"] = prev
		}
		fields["timestamp"] = stamp(at, now.Add(time.Duration(seq)*time.Millisecond)).UTC().Format(tsLayout)
		prev = id
		werr = jsonLine(&b, fields)
	}
	werr = jsonLine(&b, map[string]any{"type": "session", "version": 3, "id": sid, "timestamp": now.UTC().Format(tsLayout), "cwd": cwd})
	if name := title(t); name != "" {
		entry(now, map[string]any{"type": "session_info", "name": name})
	}
	provider := t.Header.Provider
	if provider == "" {
		provider = t.Header.SourceCLI
	}
	model := t.Header.Model
	if model == "" {
		model = "unknown"
	}
	ids := map[string]string{}
	callID := func(src string) string {
		if id, ok := ids[src]; ok {
			return id
		}
		id := "call_" + transcript.RandomAlnum(24)
		ids[src] = id
		return id
	}
	millis := func(at time.Time) int64 { return stamp(at, now.Add(time.Duration(seq+1)*time.Millisecond)).UnixMilli() }
	userText := func(text string, at time.Time) {
		entry(at, map[string]any{"type": "message", "message": map[string]any{
			"role":      "user",
			"content":   []any{map[string]any{"type": "text", "text": text}},
			"timestamp": millis(at),
		}})
		emitted = append(emitted, transcript.Event{Kind: transcript.KindMessage, Role: "user"})
	}
	zeroUsage := map[string]any{"input": 0, "output": 0, "cacheRead": 0, "cacheWrite": 0, "totalTokens": 0,
		"cost": map[string]any{"input": 0, "output": 0, "cacheRead": 0, "cacheWrite": 0, "total": 0}}
	walkTurns(t.Events,
		func(e transcript.Event) { userText(e.Text, e.Timestamp) },
		func(turn assistantTurn) {
			var content []any
			hasCall := false
			for _, blk := range turn.Blocks {
				switch blk.Kind {
				case transcript.KindMessage:
					content = append(content, map[string]any{"type": "text", "text": blk.Text})
					emitted = append(emitted, transcript.Event{Kind: transcript.KindMessage, Role: "assistant"})
				case transcript.KindToolCall:
					if req.textTools() {
						content = append(content, map[string]any{"type": "text", "text": toolCallText(blk.Call)})
						emitted = append(emitted, transcript.Event{Kind: transcript.KindMessage, Role: "assistant"})
						continue
					}
					hasCall = true
					content = append(content, map[string]any{"type": "toolCall", "id": callID(blk.Call.ID), "name": blk.Call.Name, "arguments": blk.Call.ObjectInput()})
					emitted = append(emitted, transcript.Event{Kind: transcript.KindToolCall})
				}
			}
			if len(content) == 0 {
				return
			}
			m := turn.Model
			if m == "" {
				m = model
			}
			stop := "stop"
			if hasCall {
				stop = "toolUse"
			}
			entry(turn.Timestamp, map[string]any{"type": "message", "message": map[string]any{
				"role":       "assistant",
				"content":    content,
				"api":        "handoff",
				"provider":   provider,
				"model":      m,
				"usage":      zeroUsage,
				"stopReason": stop,
				"timestamp":  millis(turn.Timestamp),
			}})
		},
		func(e transcript.Event) {
			if req.textTools() {
				userText(toolResultText(e.Result), e.Timestamp)
				return
			}
			entry(e.Timestamp, map[string]any{"type": "message", "message": map[string]any{
				"role":       "toolResult",
				"toolCallId": callID(e.Result.CallID),
				"toolName":   e.Result.Name,
				"content":    []any{map[string]any{"type": "text", "text": e.Result.Text}},
				"isError":    e.Result.IsError,
				"timestamp":  millis(e.Timestamp),
			}})
			emitted = append(emitted, transcript.Event{Kind: transcript.KindToolResult})
		},
	)
	if werr != nil {
		return Summary{}, werr
	}
	if len(emitted) == 0 {
		return Summary{}, fmt.Errorf("nothing to hand off: the conversation has no turns")
	}
	if err := writeNewFile(path, []byte(b.String())); err != nil {
		return Summary{}, err
	}
	ref := Ref{ID: sid, Path: path, Cwd: cwd}
	if session.UnderRoot(session.Root(), path) {
		if err := roundTrip(ctx, PISource{}, ref, transcript.Timeline{Events: emitted}.Counts()); err != nil {
			_ = os.Remove(path)
			return Summary{}, err
		}
	}
	return Summary{
		CLI:       "pi",
		ID:        sid,
		Path:      path,
		Name:      title(t),
		Cwd:       cwd,
		CreatedAt: rfc3339(now),
		UpdatedAt: rfc3339(now),
		Preview:   preview(t),
		Messages:  transcript.Timeline{Events: emitted}.Counts().Messages,
		Size:      fileSize(path),
		Model:     model,
	}, nil
}
