package clisession

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/cfpperche/picode/internal/transcript"
)

// omp writer (ADR-0088): creates
// <root>/<cwd-encoded>/<ISO stamp>_<id>.jsonl in omp's session version 3 —
// the measured-minimal native shape: a session header, optionally a title
// record (no id, not chained — omp's own shape) and a model_change
// annotation when the source knew the model, then id/parentId-chained
// message entries (text, toolCall blocks and toolResult messages; thinking
// is never written). The root is omp's own sessions directory — omp has no
// --session flag, so req.Dir is ignored. A hand-written file with just the
// header and one user message resumed live (the Write spike, 18.2.4), and
// the file is re-read through OmpSource.Read before the write counts.

func (OmpSource) Write(ctx context.Context, t transcript.Timeline, req WriteRequest) (Summary, error) {
	if strings.TrimSpace(req.Cwd) == "" {
		return Summary{}, fmt.Errorf("a folder is required")
	}
	root := ompSessionsRoot()
	if root == "" {
		return Summary{}, fmt.Errorf("the omp sessions root could not be resolved")
	}
	t = t.Prepare()
	sid := req.SessionID
	if sid == "" {
		sid = transcript.NewID()
	}
	now := req.resolvedNow()
	cwd := filepath.Clean(req.Cwd)
	bucket := filepath.Join(root, strings.ReplaceAll(filepath.ToSlash(cwd), "/", "-"))
	path := filepath.Join(bucket, now.UTC().Format("2006-01-02T15-04-05-000Z")+"_"+sid+".jsonl")

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
		// omp's own title records carry no id and join no chain.
		werr = jsonLine(&b, map[string]any{"type": "title", "v": 1, "title": name, "updatedAt": now.UTC().Format(tsLayout)})
	}
	// One model_change annotation when the source knew the model, the same
	// record omp itself writes on every switch. History only: the resumed
	// session picks its own model.
	model := t.Header.Model
	if t.Header.Provider != "" && model != "" {
		model = t.Header.Provider + "/" + model
	}
	if model != "" {
		entry(now, map[string]any{"type": "model_change", "model": model})
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
			msg := map[string]any{
				"role":       "assistant",
				"content":    content,
				"stopReason": "stop",
				"timestamp":  millis(turn.Timestamp),
			}
			if hasCall {
				msg["stopReason"] = "toolUse"
			}
			if m != "" {
				msg["model"] = m
			}
			entry(turn.Timestamp, map[string]any{"type": "message", "message": msg})
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
	if err := roundTrip(ctx, OmpSource{}, ref, transcript.Timeline{Events: emitted}.Counts()); err != nil {
		_ = os.Remove(path)
		return Summary{}, err
	}
	return Summary{
		CLI:        "omp",
		ID:         sid,
		Path:       path,
		ResumeArgs: []string{"--resume", sid},
		Name:       title(t),
		Cwd:        cwd,
		CreatedAt:  rfc3339(now),
		UpdatedAt:  rfc3339(now),
		Preview:    preview(t),
		Messages:   transcript.Timeline{Events: emitted}.Counts().Messages,
		Size:       fileSize(path),
		Model:      model,
	}, nil
}
