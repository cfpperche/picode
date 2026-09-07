package clisession

import (
	"context"
	"encoding/json"
	"path/filepath"
	"strings"
	"time"

	"github.com/cfpperche/picode/internal/transcript"
)

// Claude Code transcript ↔ timeline (ADR-0087). Format verified against a
// real installation (Claude Code 2.1.263, 2026-09). One JSONL line per
// record; the lines that matter:
//
//	type:user       message.content: string | [{type:text}|{type:tool_result,tool_use_id,content,is_error}|{type:image}]
//	type:assistant  message.{id,model,content:[{type:text}|{type:tool_use,id,name,input}|{type:thinking,thinking,signature}]}
//	type:summary    summary → title
//	type:system     subtype:compact_boundary, then a user line holding the summary
//	isSidechain     subagent traffic (dropped) · isMeta injected context
//
// Every other type (queue-operation, file-history-snapshot, mode, …) is
// counted by name in the manifest. Assistant blocks split across lines
// share message.id; that id is the Group so writers can merge them back.
func (ClaudeCodeSource) Read(ctx context.Context, ref Ref) (transcript.Timeline, error) {
	root := filepath.Join(homeDir(), ".claude", "projects")
	path := ref.Path
	if !underRoot(root, path) || !strings.HasSuffix(path, ".jsonl") {
		return transcript.Timeline{}, ErrNotUnderRoot
	}
	t := transcript.Timeline{Header: transcript.Header{SourceCLI: "claude-code", SourcePath: path, Cwd: ref.Cwd}}
	groups := map[string]int{}
	nextGroup := 0
	groupOf := func(msgID string) int {
		if msgID == "" {
			nextGroup++
			return nextGroup
		}
		if g, ok := groups[msgID]; ok {
			return g
		}
		nextGroup++
		groups[msgID] = nextGroup
		return nextGroup
	}
	compactPending := false
	names := map[string]string{} // tool_use id → name, for results
	err := scanLines(path, func(line []byte) {
		var raw struct {
			Type      string `json:"type"`
			Subtype   string `json:"subtype"`
			Timestamp string `json:"timestamp"`
			Cwd       string `json:"cwd"`
			SessionID string `json:"sessionId"`
			SessionLo string `json:"session_id"`
			Version   string `json:"version"`
			Sidechain bool   `json:"isSidechain"`
			IsMeta    bool   `json:"isMeta"`
			IsCompact bool   `json:"isCompactSummary"`
			Summary   string `json:"summary"`
			Message   struct {
				ID      string          `json:"id"`
				Role    string          `json:"role"`
				Model   string          `json:"model"`
				Content json.RawMessage `json:"content"`
			} `json:"message"`
		}
		if json.Unmarshal(line, &raw) != nil {
			t.Manifest.Drop("claude.malformed")
			return
		}
		if t.Header.SourceID == "" {
			t.Header.SourceID = raw.SessionID
			if t.Header.SourceID == "" {
				t.Header.SourceID = raw.SessionLo
			}
		}
		if t.Header.Cwd == "" {
			t.Header.Cwd = raw.Cwd
		}
		if t.Header.FormatVersion == "" {
			t.Header.FormatVersion = raw.Version
		}
		ts := parseTime(raw.Timestamp)
		if !ts.IsZero() {
			if t.Header.CreatedAt.IsZero() {
				t.Header.CreatedAt = ts
			}
			t.Header.UpdatedAt = ts
		}
		switch raw.Type {
		case "summary":
			if t.Header.Title == "" {
				t.Header.Title = clip(raw.Summary, 120)
			}
		case "system":
			if raw.Subtype == "compact_boundary" {
				compactPending = true
			} else {
				t.Manifest.Drop("claude.system." + raw.Subtype)
			}
		case "user":
			if raw.Sidechain {
				t.Manifest.Drop("claude.sidechain")
				return
			}
			g := groupOf("")
			var str string
			if json.Unmarshal(raw.Message.Content, &str) == nil {
				claudeUserText(&t, &compactPending, raw.IsMeta || raw.IsCompact, raw.IsCompact, str, ts, g)
				return
			}
			var blocks []struct {
				Type      string          `json:"type"`
				Text      string          `json:"text"`
				ToolUseID string          `json:"tool_use_id"`
				Content   json.RawMessage `json:"content"`
				IsError   bool            `json:"is_error"`
			}
			if json.Unmarshal(raw.Message.Content, &blocks) != nil {
				t.Manifest.Drop("claude.malformed")
				return
			}
			var texts []string
			for _, b := range blocks {
				switch b.Type {
				case "text":
					if strings.HasPrefix(strings.TrimSpace(b.Text), "<system-reminder>") {
						t.Events = append(t.Events, transcript.Event{Kind: transcript.KindContext, Role: "user", Text: b.Text, Timestamp: ts, Group: g})
						continue
					}
					if strings.TrimSpace(b.Text) != "" {
						texts = append(texts, strings.TrimSpace(b.Text))
					}
				case "tool_result":
					text := textBlocks(b.Content, &t.Manifest, "claude.tool_result")
					t.Events = append(t.Events, transcript.Event{Kind: transcript.KindToolResult, Result: &transcript.ToolResult{CallID: b.ToolUseID, Name: names[b.ToolUseID], Text: text, IsError: b.IsError}, Timestamp: ts, Group: g})
				case "image":
					t.Manifest.Drop("image")
				default:
					t.Manifest.Drop("claude.user." + b.Type)
				}
			}
			if len(texts) > 0 {
				claudeUserText(&t, &compactPending, raw.IsMeta || raw.IsCompact, raw.IsCompact, strings.Join(texts, "\n"), ts, g)
			}
		case "assistant":
			if raw.Sidechain {
				t.Manifest.Drop("claude.sidechain")
				return
			}
			if compactPending {
				t.Events = append(t.Events, transcript.Event{Kind: transcript.KindCompaction, Text: "Conversation compacted.", Timestamp: ts})
				compactPending = false
			}
			if raw.Message.Model != "" {
				t.Header.Model = raw.Message.Model
			}
			g := groupOf(raw.Message.ID)
			var blocks []struct {
				Type      string          `json:"type"`
				Text      string          `json:"text"`
				ID        string          `json:"id"`
				Name      string          `json:"name"`
				Input     json.RawMessage `json:"input"`
				Thinking  string          `json:"thinking"`
				Signature string          `json:"signature"`
			}
			if json.Unmarshal(raw.Message.Content, &blocks) != nil {
				var str string
				if json.Unmarshal(raw.Message.Content, &str) == nil && strings.TrimSpace(str) != "" {
					t.Events = append(t.Events, transcript.Event{Kind: transcript.KindMessage, Role: "assistant", Text: str, Model: raw.Message.Model, Timestamp: ts, Group: g})
				}
				return
			}
			for _, b := range blocks {
				switch b.Type {
				case "text":
					if strings.TrimSpace(b.Text) != "" {
						t.Events = append(t.Events, transcript.Event{Kind: transcript.KindMessage, Role: "assistant", Text: b.Text, Model: raw.Message.Model, Timestamp: ts, Group: g})
					}
				case "tool_use":
					names[b.ID] = b.Name
					t.Events = append(t.Events, transcript.Event{Kind: transcript.KindToolCall, Role: "assistant", Call: &transcript.ToolCall{ID: b.ID, Name: b.Name, Input: argumentsJSON(b.Input)}, Model: raw.Message.Model, Timestamp: ts, Group: g})
				case "thinking", "redacted_thinking":
					t.Events = append(t.Events, transcript.Event{Kind: transcript.KindThinking, Role: "assistant", Text: b.Thinking, Timestamp: ts, Group: g})
				default:
					t.Manifest.Drop("claude.assistant." + b.Type)
				}
			}
		default:
			t.Manifest.Drop("claude." + raw.Type)
		}
	})
	if err != nil {
		return transcript.Timeline{}, err
	}
	if t.Header.SourceID == "" {
		t.Header.SourceID = strings.TrimSuffix(filepath.Base(path), ".jsonl")
	}
	if ref.ID != "" && t.Header.SourceID != ref.ID {
		t.Header.SourceID = ref.ID
	}
	return t, nil
}

// claudeUserText classifies one user text: the compaction summary that
// follows a compact boundary, injected context (isMeta, slash-command
// echoes, the "[Request interrupted by user]" marker Claude Code writes
// when a turn is cut short), or a human turn.
func claudeUserText(t *transcript.Timeline, compactPending *bool, meta, compactSummary bool, text string, ts time.Time, g int) {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return
	}
	if compactSummary || (*compactPending && strings.HasPrefix(trimmed, "This session is being continued")) {
		t.Events = append(t.Events, transcript.Event{Kind: transcript.KindCompaction, Text: trimmed, Timestamp: ts, Group: g})
		*compactPending = false
		return
	}
	if *compactPending {
		t.Events = append(t.Events, transcript.Event{Kind: transcript.KindCompaction, Text: "Conversation compacted.", Timestamp: ts})
		*compactPending = false
	}
	if meta || strings.HasPrefix(trimmed, "<command-") || strings.HasPrefix(trimmed, "<local-command") || strings.HasPrefix(trimmed, "<bash-") || strings.HasPrefix(trimmed, "<system-reminder>") || strings.HasPrefix(trimmed, "[Request interrupted by user") {
		t.Events = append(t.Events, transcript.Event{Kind: transcript.KindContext, Role: "user", Text: trimmed, Timestamp: ts, Group: g})
		return
	}
	t.Events = append(t.Events, transcript.Event{Kind: transcript.KindMessage, Role: "user", Text: trimmed, Timestamp: ts, Group: g})
}

// PromptArgs: `claude --session-id <uuid> <prompt>` — verified flags, the
// prompt is positional (claude --help, 2.1.263).
func (ClaudeCodeSource) PromptArgs(prompt, sessionID string) []string {
	if sessionID == "" {
		return []string{prompt}
	}
	return []string{"--session-id", sessionID, prompt}
}
