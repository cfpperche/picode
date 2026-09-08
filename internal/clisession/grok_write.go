package clisession

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/cfpperche/picode/internal/transcript"
)

// Grok writer (ADR-0088): creates
// $GROK_HOME/sessions/<url-escaped cwd>/<uuid>/{summary.json,
// chat_history.jsonl}. A spike against the real CLI (Grok Build 1.0.13,
// 2026-09-07) established the minimum: those two files are enough for
// `grok --resume <id>` to load the conversation and answer about it.
// updates.jsonl (the ACP update log), events.jsonl, prompt_context.json
// and the rest are runtime state the CLI rebuilds, and the folder's shared
// prompt_history.jsonl is never appended to — it belongs to sessions Grok
// itself ran, and ADR-0088 creates, never edits.
func (GrokSource) Write(ctx context.Context, t transcript.Timeline, req WriteRequest) (Summary, error) {
	if strings.TrimSpace(req.Cwd) == "" {
		return Summary{}, fmt.Errorf("a folder is required")
	}
	root := grokSessionsRoot()
	if root == "" {
		return Summary{}, ErrNotUnderRoot
	}
	// Grok versions its chat history separately from the CLI: the local
	// store's chat_format_version decides. With no session to read it
	// from, an installed CLI (a version from the setup check) is taken to
	// write the one format this writer was proven against.
	format := grokChatFormat(root)
	if format == 0 && strings.TrimSpace(req.FormatVersion) != "" {
		format = grokChatFormatVerified
	}
	if format != grokChatFormatVerified {
		return Summary{}, ErrUnknownFormat
	}
	t = t.Prepare()
	sid := req.SessionID
	if sid == "" {
		sid = transcript.NewID()
	}
	now := req.resolvedNow()
	cwd := filepath.Clean(req.Cwd)
	dir := filepath.Join(root, url.PathEscape(cwd), sid)
	// current_model_id names a model this Grok can serve, so it comes from
	// the local store, never from the source CLI: a live probe answered
	// "Model claude-opus-5 is no longer available for your account" and
	// switched. With no local session to learn from, the key is omitted
	// and Grok picks its own default.
	model := grokCurrentModel()

	var b strings.Builder
	var emitted []transcript.Event
	var werr error
	prompts := 0
	line := func(v any) {
		if werr == nil {
			werr = jsonLine(&b, v)
		}
	}
	userLine := func(text string) {
		line(map[string]any{
			"type":         "user",
			"content":      []any{map[string]any{"type": "text", "text": "<user_query>\n" + text + "\n</user_query>"}},
			"prompt_index": prompts,
		})
		prompts++
		emitted = append(emitted, transcript.Event{Kind: transcript.KindMessage, Role: "user"})
	}
	walkTurns(t.Events,
		func(e transcript.Event) { userLine(e.Text) },
		func(turn assistantTurn) {
			var texts []string
			var calls []any
			for _, blk := range turn.Blocks {
				switch blk.Kind {
				case transcript.KindMessage:
					texts = append(texts, blk.Text)
					emitted = append(emitted, transcript.Event{Kind: transcript.KindMessage, Role: "assistant"})
				case transcript.KindToolCall:
					if req.textTools() {
						texts = append(texts, toolCallText(blk.Call))
						emitted = append(emitted, transcript.Event{Kind: transcript.KindMessage, Role: "assistant"})
						continue
					}
					calls = append(calls, map[string]any{"id": blk.Call.ID, "name": blk.Call.Name, "arguments": blk.Call.InputString()})
					emitted = append(emitted, transcript.Event{Kind: transcript.KindToolCall})
				}
			}
			if len(texts) == 0 && len(calls) == 0 {
				return
			}
			rec := map[string]any{"type": "assistant", "content": strings.Join(texts, "\n\n")}
			if model != "" {
				rec["model_id"] = model
			}
			if len(calls) > 0 {
				rec["tool_calls"] = calls
			}
			line(rec)
		},
		func(e transcript.Event) {
			if req.textTools() {
				userLine(toolResultText(e.Result))
				return
			}
			line(map[string]any{"type": "tool_result", "tool_call_id": e.Result.CallID, "content": e.Result.Text, "is_error": e.Result.IsError})
			emitted = append(emitted, transcript.Event{Kind: transcript.KindToolResult})
		},
	)
	if werr != nil {
		return Summary{}, werr
	}
	counts := transcript.Timeline{Events: emitted}.Counts()
	if len(emitted) == 0 {
		return Summary{}, fmt.Errorf("nothing to hand off: the conversation has no turns")
	}

	name := title(t)
	if name == "" {
		name = "Handoff from " + t.Header.DisplayName()
	}
	stamp := now.Format("2006-01-02T15:04:05.000000000Z")
	summary := map[string]any{
		"info":                map[string]any{"id": sid, "cwd": cwd},
		"session_summary":     name,
		"generated_title":     name,
		"created_at":          stamp,
		"updated_at":          stamp,
		"last_active_at":      stamp,
		"num_messages":        counts.Messages,
		"num_chat_messages":   counts.Messages + counts.ToolCalls + counts.ToolResults,
		"chat_format_version": grokChatFormatVerified,
		"next_trace_turn":     0,
		"git_root_dir":        cwd + "/",
		"grok_home":           grokHome(),
	}
	if model != "" {
		summary["current_model_id"] = model
	}
	body, err := marshalNative(summary)
	if err != nil {
		return Summary{}, err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return Summary{}, err
	}
	chat := filepath.Join(dir, "chat_history.jsonl")
	if err := writeNewFile(chat, []byte(b.String())); err != nil {
		return Summary{}, err
	}
	if err := writeNewFile(filepath.Join(dir, "summary.json"), body); err != nil {
		_ = os.RemoveAll(dir)
		return Summary{}, err
	}
	ref := Ref{ID: sid, Path: dir, Cwd: cwd}
	if err := roundTrip(ctx, GrokSource{}, ref, counts); err != nil {
		_ = os.RemoveAll(dir)
		return Summary{}, err
	}
	return Summary{
		CLI:        "grok",
		ID:         sid,
		Path:       dir,
		ResumeArgs: []string{"--resume", sid},
		Name:       clip(name, 120),
		Cwd:        cwd,
		CreatedAt:  rfc3339(now),
		UpdatedAt:  rfc3339(now),
		Preview:    preview(t),
		Messages:   counts.Messages,
		Size:       fileSize(chat),
		Model:      model,
	}, nil
}

// grokChatFormatVerified is the only chat_format_version the reader and
// the writer were proven against.
const grokChatFormatVerified = 1

// grokHome is $GROK_HOME or ~/.grok.
func grokHome() string {
	if h := strings.TrimSpace(os.Getenv("GROK_HOME")); h != "" {
		return h
	}
	home := homeDir()
	if home == "" {
		return ""
	}
	return filepath.Join(home, ".grok")
}

// grokSessionsRoot is the sessions directory of the active Grok home.
// GrokSessionsRoot is where the Grok CLI keeps prompt history. Exported so
// climetrics reads the same path this package lists from.
func GrokSessionsRoot() string { return grokSessionsRoot() }

func grokSessionsRoot() string {
	h := grokHome()
	if h == "" {
		return ""
	}
	return filepath.Join(h, "sessions")
}

// grokCurrentModel is the model of the newest session in the local store —
// what this installation is actually serving. "" when there is none.
func grokCurrentModel() string {
	rows, err := (GrokSource{}).List("")
	if err != nil {
		return ""
	}
	sortNewest(rows)
	for _, r := range rows {
		if strings.TrimSpace(r.Model) != "" {
			return r.Model
		}
	}
	return ""
}

// grokChatFormat reads chat_format_version from the newest session on this
// machine, or 0 when there is none to read.
func grokChatFormat(root string) int {
	ents, err := os.ReadDir(root)
	if err != nil {
		return 0
	}
	newest, format := "", 0
	for _, folder := range ents {
		if !folder.IsDir() {
			continue
		}
		sessions, err := os.ReadDir(filepath.Join(root, folder.Name()))
		if err != nil {
			continue
		}
		for _, s := range sessions {
			if !s.IsDir() {
				continue
			}
			dir := filepath.Join(root, folder.Name(), s.Name())
			sum, ok := readGrokSummary(dir)
			if !ok || sum.ChatFormat == 0 {
				continue
			}
			at := mtime(filepath.Join(dir, "summary.json"))
			if at > newest {
				newest, format = at, sum.ChatFormat
			}
		}
	}
	return format
}
