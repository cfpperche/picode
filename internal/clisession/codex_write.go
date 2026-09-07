package clisession

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/cfpperche/picode/internal/transcript"
)

// Codex writer (ADR-0088): creates
// ~/.codex/sessions/YYYY/MM/DD/rollout-<YYYY-MM-DDThh-mm-ss>-<uuid>.jsonl
// in the shape Codex CLI 0.1xx writes: a session_meta envelope whose id
// matches the file name, then response_item records — messages with
// input_text/output_text, function_call with JSON-string arguments and a
// call_id, function_call_output with the same call_id. Reasoning is
// encrypted in Codex's own files and is never written. Codex's SQLite
// index is not touched: the CLI discovers rollouts by scanning. The file
// is re-read through CodexSource.Read before the write counts as done.

// codexNewestFacts reads cli_version and the model from the most recent
// rollout on this machine: the format marker and a model this Codex can
// serve. The written session records the latter, never the source's.
func codexNewestFacts(root string) (version, model string) {
	newest := newestFile(jsonlFiles(root))
	if newest == "" {
		return "", ""
	}
	_ = scanLines(newest, func(line []byte) {
		var raw struct {
			Type    string `json:"type"`
			Payload struct {
				Version string `json:"cli_version"`
				Model   string `json:"model"`
			} `json:"payload"`
		}
		if json.Unmarshal(line, &raw) != nil {
			return
		}
		switch raw.Type {
		case "session_meta":
			if version == "" {
				version = raw.Payload.Version
			}
		case "turn_context":
			if raw.Payload.Model != "" {
				model = raw.Payload.Model
			}
		}
	})
	return version, model
}

func (CodexSource) Write(ctx context.Context, t transcript.Timeline, req WriteRequest) (Summary, error) {
	if strings.TrimSpace(req.Cwd) == "" {
		return Summary{}, fmt.Errorf("a folder is required")
	}
	root := filepath.Join(homeDir(), ".codex", "sessions")
	localVersion, localModel := codexNewestFacts(root)
	version := strings.TrimSpace(req.FormatVersion)
	if version == "" {
		version = localVersion
	}
	if version == "" {
		return Summary{}, ErrUnknownFormat
	}
	t = t.Prepare()
	sid := req.SessionID
	if sid == "" {
		sid = transcript.NewID()
	}
	now := req.resolvedNow()
	cwd := filepath.Clean(req.Cwd)
	dir := filepath.Join(root, now.Format("2006"), now.Format("01"), now.Format("02"))
	path := filepath.Join(dir, "rollout-"+now.Format("2006-01-02T15-04-05")+"-"+sid+".jsonl")

	const tsLayout = "2006-01-02T15:04:05.000Z"
	var b strings.Builder
	var emitted []transcript.Event
	var werr error
	seq := 0
	item := func(at time.Time, kind string, payload map[string]any) {
		if werr != nil {
			return
		}
		seq++
		werr = jsonLine(&b, map[string]any{
			"timestamp": stamp(at, now.Add(time.Duration(seq)*time.Millisecond)).UTC().Format(tsLayout),
			"type":      kind,
			"payload":   payload,
		})
	}
	item(now, "session_meta", map[string]any{
		"id":            sid,
		"session_id":    sid,
		"timestamp":     now.UTC().Format(tsLayout),
		"cwd":           cwd,
		"originator":    "picode-handoff",
		"cli_version":   version,
		"source":        "cli",
		"thread_source": "user",
	})
	if localModel != "" {
		// Without this, a session handed to Codex and then handed on again
		// carries no model at all: the reader learns it from turn_context.
		item(now, "turn_context", map[string]any{"cwd": cwd, "model": localModel})
	}
	ids := map[string]string{}
	callID := func(src string) string {
		if id, ok := ids[src]; ok {
			return id
		}
		id := "call_" + transcript.RandomAlnum(22)
		ids[src] = id
		return id
	}
	message := func(role, kind, text string, at time.Time) {
		item(at, "response_item", map[string]any{"type": "message", "role": role, "content": []any{map[string]any{"type": kind, "text": text}}})
		emitted = append(emitted, transcript.Event{Kind: transcript.KindMessage, Role: role})
	}
	walkTurns(t.Events,
		func(e transcript.Event) { message("user", "input_text", e.Text, e.Timestamp) },
		func(turn assistantTurn) {
			for _, blk := range turn.Blocks {
				switch blk.Kind {
				case transcript.KindMessage:
					message("assistant", "output_text", blk.Text, blk.Timestamp)
				case transcript.KindToolCall:
					if req.textTools() {
						message("assistant", "output_text", toolCallText(blk.Call), blk.Timestamp)
						continue
					}
					item(blk.Timestamp, "response_item", map[string]any{
						"type":      "function_call",
						"name":      blk.Call.Name,
						"arguments": blk.Call.InputString(),
						"call_id":   callID(blk.Call.ID),
					})
					emitted = append(emitted, transcript.Event{Kind: transcript.KindToolCall})
				}
			}
		},
		func(e transcript.Event) {
			if req.textTools() {
				message("user", "input_text", toolResultText(e.Result), e.Timestamp)
				return
			}
			item(e.Timestamp, "response_item", map[string]any{
				"type":    "function_call_output",
				"call_id": callID(e.Result.CallID),
				"output":  e.Result.Text,
			})
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
	if err := roundTrip(ctx, CodexSource{}, ref, transcript.Timeline{Events: emitted}.Counts()); err != nil {
		_ = os.Remove(path)
		return Summary{}, err
	}
	return Summary{
		CLI:        "codex",
		ID:         sid,
		Path:       path,
		ResumeArgs: []string{"resume", sid},
		Name:       title(t),
		Cwd:        cwd,
		CreatedAt:  rfc3339(now),
		UpdatedAt:  rfc3339(now),
		Preview:    preview(t),
		Messages:   transcript.Timeline{Events: emitted}.Counts().Messages,
		Size:       fileSize(path),
		Model:      localModel,
	}, nil
}
