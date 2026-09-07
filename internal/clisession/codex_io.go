package clisession

import (
	"context"
	"encoding/json"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/cfpperche/picode/internal/transcript"
)

// Codex rollout ↔ timeline (ADR-0088). Format verified against real
// rollouts (Codex CLI 0.78 … 0.153, 2026-09). Envelope per line:
// {timestamp, type, payload}. Types that carry conversation:
//
//	session_meta    payload.{id, session_id, cwd, timestamp, cli_version, model_provider}
//	turn_context    payload.{model, cwd}
//	response_item   payload.type: message{role, content[{type:input_text|output_text,text}]}
//	                  | function_call{name, arguments:<JSON string>, call_id}
//	                  | custom_tool_call{name, input:<string>, call_id}
//	                  | function_call_output|custom_tool_call_output{call_id, output:<string|[{text}]>}
//	                  | reasoning{summary[{text}], encrypted_content}
//	compacted       payload.message (the summary; may be empty)
//
// event_msg, token_usage_record, world_state and the rest are runtime
// state and are counted by name. Developer-role and injected user
// messages (AGENTS.md preamble, environment context) are context.
func (CodexSource) Read(ctx context.Context, ref Ref) (transcript.Timeline, error) {
	root := filepath.Join(homeDir(), ".codex", "sessions")
	path := ref.Path
	if !underRoot(root, path) || !strings.HasSuffix(path, ".jsonl") {
		return transcript.Timeline{}, ErrNotUnderRoot
	}
	t := transcript.Timeline{Header: transcript.Header{SourceCLI: "codex", SourcePath: path, Cwd: ref.Cwd}}
	model := ""
	group := 0
	names := map[string]string{}
	err := scanLines(path, func(line []byte) {
		var raw struct {
			Timestamp string          `json:"timestamp"`
			Type      string          `json:"type"`
			Payload   json.RawMessage `json:"payload"`
		}
		if json.Unmarshal(line, &raw) != nil {
			t.Manifest.Drop("codex.malformed")
			return
		}
		ts := parseTime(raw.Timestamp)
		if !ts.IsZero() {
			if t.Header.CreatedAt.IsZero() {
				t.Header.CreatedAt = ts
			}
			t.Header.UpdatedAt = ts
		}
		group++
		switch raw.Type {
		case "session_meta":
			var p struct {
				ID        string `json:"id"`
				SessionID string `json:"session_id"`
				Timestamp string `json:"timestamp"`
				Cwd       string `json:"cwd"`
				Version   string `json:"cli_version"`
				Provider  string `json:"model_provider"`
			}
			if json.Unmarshal(raw.Payload, &p) != nil {
				return
			}
			if t.Header.SourceID == "" {
				t.Header.SourceID = p.ID
				if t.Header.SourceID == "" {
					t.Header.SourceID = p.SessionID
				}
			}
			if p.Cwd != "" {
				t.Header.Cwd = p.Cwd
			}
			if t.Header.FormatVersion == "" {
				t.Header.FormatVersion = p.Version
			}
			if t.Header.Provider == "" {
				t.Header.Provider = p.Provider
			}
			if at := parseTime(p.Timestamp); !at.IsZero() {
				t.Header.CreatedAt = at
			}
		case "turn_context":
			var p struct {
				Model string `json:"model"`
				Cwd   string `json:"cwd"`
			}
			if json.Unmarshal(raw.Payload, &p) == nil {
				if p.Model != "" {
					model = p.Model
					t.Header.Model = p.Model
				}
				if p.Cwd != "" && t.Header.Cwd == "" {
					t.Header.Cwd = p.Cwd
				}
			}
		case "response_item":
			var p struct {
				Type      string          `json:"type"`
				Role      string          `json:"role"`
				Name      string          `json:"name"`
				Arguments json.RawMessage `json:"arguments"`
				Input     json.RawMessage `json:"input"`
				CallID    string          `json:"call_id"`
				Content   json.RawMessage `json:"content"`
				Output    json.RawMessage `json:"output"`
				Summary   json.RawMessage `json:"summary"`
			}
			if json.Unmarshal(raw.Payload, &p) != nil {
				t.Manifest.Drop("codex.malformed")
				return
			}
			switch p.Type {
			case "message":
				text := textBlocks(p.Content, &t.Manifest, "codex.message")
				if text == "" {
					return
				}
				switch p.Role {
				case "user":
					kind := transcript.KindMessage
					if codexContext(text) {
						kind = transcript.KindContext
					}
					t.Events = append(t.Events, transcript.Event{Kind: kind, Role: "user", Text: text, Timestamp: ts, Group: group})
				case "assistant":
					t.Events = append(t.Events, transcript.Event{Kind: transcript.KindMessage, Role: "assistant", Text: text, Model: model, Timestamp: ts, Group: group})
				default: // developer, system
					t.Events = append(t.Events, transcript.Event{Kind: transcript.KindContext, Role: p.Role, Text: text, Timestamp: ts, Group: group})
				}
			case "function_call", "custom_tool_call":
				input := p.Arguments
				if p.Type == "custom_tool_call" {
					input = p.Input
				}
				names[p.CallID] = p.Name
				t.Events = append(t.Events, transcript.Event{Kind: transcript.KindToolCall, Role: "assistant", Call: &transcript.ToolCall{ID: p.CallID, Name: p.Name, Input: argumentsJSON(input)}, Model: model, Timestamp: ts, Group: group})
			case "function_call_output", "custom_tool_call_output":
				text := textBlocks(p.Output, &t.Manifest, "codex.output")
				t.Events = append(t.Events, transcript.Event{Kind: transcript.KindToolResult, Result: &transcript.ToolResult{CallID: p.CallID, Name: names[p.CallID], Text: text}, Timestamp: ts, Group: group})
			case "reasoning":
				text := textBlocks(p.Summary, nil, "")
				if text == "" {
					t.Manifest.Drop("reasoning.encrypted")
					return
				}
				t.Events = append(t.Events, transcript.Event{Kind: transcript.KindThinking, Role: "assistant", Text: text, Timestamp: ts, Group: group})
			default:
				t.Manifest.Drop("codex.response_item." + p.Type)
			}
		case "compacted":
			var p struct {
				Message string `json:"message"`
			}
			_ = json.Unmarshal(raw.Payload, &p)
			text := strings.TrimSpace(p.Message)
			if text == "" {
				text = "Session compacted."
			}
			t.Events = append(t.Events, transcript.Event{Kind: transcript.KindCompaction, Text: text, Timestamp: ts, Group: group})
		default:
			t.Manifest.Drop("codex." + raw.Type)
		}
	})
	if err != nil {
		return transcript.Timeline{}, err
	}
	if t.Header.SourceID == "" {
		t.Header.SourceID = codexIDFromName(path)
	}
	return t, nil
}

// codexContext recognizes the user-role records Codex injects itself —
// the AGENTS.md preamble ("# AGENTS.md instructions for …") and tagged
// blocks such as <environment_context> or <permissions instructions> — as
// opposed to a human turn that merely starts with a heading or a "<".
func codexContext(text string) bool {
	if strings.HasPrefix(text, "# AGENTS.md") {
		return true
	}
	return codexTag.MatchString(text)
}

var codexTag = regexp.MustCompile(`^<[a-z][a-z0-9_]*( [a-z_]+)?>`)

// PromptArgs: `codex <prompt>` — the prompt is positional (codex --help,
// 0.153); Codex cannot pre-assign a session id, so sessionID is ignored
// and lineage resolves later through the terminal's pinned session.
func (CodexSource) PromptArgs(prompt, _ string) []string {
	return []string{prompt}
}
