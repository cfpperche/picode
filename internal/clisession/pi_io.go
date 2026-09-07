package clisession

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/cfpperche/picode/internal/session"
	"github.com/cfpperche/picode/internal/transcript"
)

// pi session JSONL ↔ timeline (ADR-0087). Format verified against real
// files (pi 0.85, session version 3, 2026-09):
//
//	{type:session,version:3,id,timestamp,cwd}                 header, first line
//	{type:model_change,id,parentId,provider,modelId}
//	{type:session_info,id,parentId,name}
//	{type:message,id,parentId,timestamp,message:{role:user,content:string|[{type:text}|{type:image}]}}
//	{type:message,…,message:{role:assistant,provider,model,content:[{type:text}|{type:thinking,thinking,
//	     thinkingSignature}|{type:toolCall,id,name,arguments:<object>}],stopReason,usage}}
//	{type:message,…,message:{role:toolResult,toolCallId,toolName,content:[{type:text}],isError}}
//	{type:compaction,id,parentId,summary,firstKeptEntryId}
//	{type:thinking_level_change|custom,…}                     runtime state, counted
//
// Entries form an id/parentId tree; the live conversation is the ancestry
// of the last entry, which is what pi replays on resume. Abandoned
// branches are counted in the manifest, never emitted.
func (PISource) Read(ctx context.Context, ref Ref) (transcript.Timeline, error) {
	path := ref.Path
	if !session.UnderRoot(session.Root(), path) {
		return transcript.Timeline{}, ErrNotUnderRoot
	}
	type entry struct {
		Type      string          `json:"type"`
		ID        string          `json:"id"`
		ParentID  string          `json:"parentId"`
		Timestamp string          `json:"timestamp"`
		Version   int             `json:"version"`
		Cwd       string          `json:"cwd"`
		Provider  string          `json:"provider"`
		ModelID   string          `json:"modelId"`
		Name      string          `json:"name"`
		Summary   string          `json:"summary"`
		Message   json.RawMessage `json:"message"`
	}
	t := transcript.Timeline{Header: transcript.Header{SourceCLI: "pi", SourcePath: path, SourceID: ref.ID, Cwd: ref.Cwd}}
	var entries []entry
	headerSeen := false
	err := scanLines(path, func(line []byte) {
		var e entry
		if json.Unmarshal(line, &e) != nil {
			t.Manifest.Drop("pi.malformed")
			return
		}
		if e.Type == "session" && !headerSeen {
			headerSeen = true
			if e.Version != 3 {
				t.Manifest.Warn("pi session version %d is not the verified version 3.", e.Version)
			}
			t.Header.FormatVersion = fmt.Sprintf("%d", e.Version)
			if e.ID != "" {
				t.Header.SourceID = e.ID
			}
			if e.Cwd != "" {
				t.Header.Cwd = e.Cwd
			}
			t.Header.CreatedAt = parseTime(e.Timestamp)
			return
		}
		entries = append(entries, e)
	})
	if err != nil {
		return transcript.Timeline{}, err
	}
	if !headerSeen {
		return transcript.Timeline{}, fmt.Errorf("not a pi session file")
	}
	if t.Header.FormatVersion != "3" {
		return transcript.Timeline{}, ErrUnknownFormat
	}
	// Live path: walk parents from the last entry. Entries without ids
	// (older files) fall back to file order.
	live := entries
	if len(entries) > 0 && entries[len(entries)-1].ID != "" {
		byID := map[string]int{}
		allHaveIDs := true
		for i, e := range entries {
			if e.ID == "" {
				allHaveIDs = false
				break
			}
			byID[e.ID] = i
		}
		if allHaveIDs {
			var chain []entry
			seen := map[string]bool{}
			for id := entries[len(entries)-1].ID; id != "" && !seen[id]; {
				seen[id] = true
				i, ok := byID[id]
				if !ok {
					break
				}
				chain = append(chain, entries[i])
				id = entries[i].ParentID
			}
			for i, j := 0, len(chain)-1; i < j; i, j = i+1, j-1 {
				chain[i], chain[j] = chain[j], chain[i]
			}
			if n := len(entries) - len(chain); n > 0 {
				t.Manifest.Dropped = addDrop(t.Manifest.Dropped, "pi.branch", n)
			}
			live = chain
		}
	}
	names := map[string]string{}
	for i, e := range live {
		ts := parseTime(e.Timestamp)
		if !ts.IsZero() {
			t.Header.UpdatedAt = ts
		}
		switch e.Type {
		case "model_change":
			if e.Provider != "" {
				t.Header.Provider = e.Provider
			}
			if e.ModelID != "" {
				t.Header.Model = e.ModelID
			}
		case "session_info":
			if e.Name != "" {
				t.Header.Title = clip(e.Name, 120)
			}
		case "compaction", "compaction_summary":
			text := strings.TrimSpace(e.Summary)
			if text == "" {
				text = "Session compacted."
			}
			t.Events = append(t.Events, transcript.Event{Kind: transcript.KindCompaction, Text: text, Timestamp: ts, Group: i + 1})
		case "message":
			var m struct {
				Role       string          `json:"role"`
				Provider   string          `json:"provider"`
				Model      string          `json:"model"`
				Timestamp  int64           `json:"timestamp"`
				Content    json.RawMessage `json:"content"`
				ToolCallID string          `json:"toolCallId"`
				ToolName   string          `json:"toolName"`
				IsError    bool            `json:"isError"`
			}
			if json.Unmarshal(e.Message, &m) != nil {
				t.Manifest.Drop("pi.malformed")
				continue
			}
			if m.Timestamp > 0 {
				ts = time.UnixMilli(m.Timestamp).UTC()
				t.Header.UpdatedAt = ts
			}
			g := i + 1
			switch m.Role {
			case "user":
				if text := textBlocks(m.Content, &t.Manifest, "pi.user"); text != "" {
					t.Events = append(t.Events, transcript.Event{Kind: transcript.KindMessage, Role: "user", Text: text, Timestamp: ts, Group: g})
				}
			case "assistant":
				if m.Model != "" {
					t.Header.Model = m.Model
				}
				if m.Provider != "" {
					t.Header.Provider = m.Provider
				}
				var blocks []struct {
					Type      string          `json:"type"`
					Text      string          `json:"text"`
					Thinking  string          `json:"thinking"`
					ID        string          `json:"id"`
					Name      string          `json:"name"`
					Arguments json.RawMessage `json:"arguments"`
				}
				if json.Unmarshal(m.Content, &blocks) != nil {
					if text := textBlocks(m.Content, nil, ""); text != "" {
						t.Events = append(t.Events, transcript.Event{Kind: transcript.KindMessage, Role: "assistant", Text: text, Model: m.Model, Timestamp: ts, Group: g})
					}
					continue
				}
				for _, b := range blocks {
					switch b.Type {
					case "text":
						if strings.TrimSpace(b.Text) != "" {
							t.Events = append(t.Events, transcript.Event{Kind: transcript.KindMessage, Role: "assistant", Text: b.Text, Model: m.Model, Timestamp: ts, Group: g})
						}
					case "thinking":
						t.Events = append(t.Events, transcript.Event{Kind: transcript.KindThinking, Role: "assistant", Text: b.Thinking, Timestamp: ts, Group: g})
					case "toolCall":
						names[b.ID] = b.Name
						t.Events = append(t.Events, transcript.Event{Kind: transcript.KindToolCall, Role: "assistant", Call: &transcript.ToolCall{ID: b.ID, Name: b.Name, Input: argumentsJSON(b.Arguments)}, Model: m.Model, Timestamp: ts, Group: g})
					default:
						t.Manifest.Drop("pi.assistant." + b.Type)
					}
				}
			case "toolResult":
				name := m.ToolName
				if name == "" {
					name = names[m.ToolCallID]
				}
				t.Events = append(t.Events, transcript.Event{Kind: transcript.KindToolResult, Result: &transcript.ToolResult{CallID: m.ToolCallID, Name: name, Text: textBlocks(m.Content, &t.Manifest, "pi.tool_result"), IsError: m.IsError}, Timestamp: ts, Group: g})
			default:
				t.Manifest.Drop("pi.message." + m.Role)
			}
		default:
			t.Manifest.Drop("pi." + e.Type)
		}
	}
	return t, nil
}

func addDrop(m map[string]int, key string, n int) map[string]int {
	if m == nil {
		m = map[string]int{}
	}
	m[key] += n
	return m
}

// PromptArgs: `pi --session-id <id> <message>` — verified flags (pi
// --help, 0.85): messages are positional; --session-id creates the
// session if missing.
func (PISource) PromptArgs(prompt, sessionID string) []string {
	if sessionID == "" {
		return []string{prompt}
	}
	return []string{"--session-id", sessionID, prompt}
}
