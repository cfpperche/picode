package clisession

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/cfpperche/picode/internal/transcript"
)

// ompEntry is one record of an omp session file (schema v3, measured on
// real files from omp 18.2.4, 2026-09-17).
type ompEntry struct {
	Type      string          `json:"type"`
	ID        string          `json:"id"`
	ParentID  string          `json:"parentId"`
	Timestamp string          `json:"timestamp"`
	Version   int             `json:"version"`
	Cwd       string          `json:"cwd"`
	Model     string          `json:"model"`
	Title     string          `json:"title"`
	Custom    string          `json:"customType"`
	Summary   string          `json:"summary"`
	Message   json.RawMessage `json:"message"`
}

// omp session JSONL ↔ timeline (ADR-0088). Format verified against real
// files (omp 18.2.4, session version 3, 2026-09-17) and against a
// hand-written minimal file the CLI resumed live (the Write spike):
//
//	{type:session,version:3,id,timestamp,cwd}                header, first line
//	{type:title,v:1,title,updatedAt}                          no id, not chained
//	{type:model_change,id,parentId,model:"google/gemini-3.6-flash"}
//	{type:thinking_level_change,…}                            runtime state, counted
//	{type:message,id,parentId,timestamp,message:{role:user|assistant|toolResult,…}}
//	{type:custom,customType:session_exit|tool_execution_start,…}  counted
//
// Messages carry epoch-millis timestamps; outer records carry ISO. The
// entry chain (id/parentId, live path = ancestry of the last entry) is
// pi-shaped, as are the content blocks: text, thinking, toolCall, and the
// toolResult message role. A hand-written file with only the header and
// one user message resumes — minimal writes are native writes.
func (OmpSource) Read(ctx context.Context, ref Ref) (transcript.Timeline, error) {
	path := ref.Path
	root := ompSessionsRoot()
	if root == "" || !underRoot(root, path) {
		return transcript.Timeline{}, ErrNotUnderRoot
	}
	t := transcript.Timeline{Header: transcript.Header{SourceCLI: "omp", SourcePath: path, SourceID: ref.ID, Cwd: ref.Cwd}}
	var entries []ompEntry
	headerSeen := false
	err := scanSession(path, ref, func(line []byte) {
		var e ompEntry
		if json.Unmarshal(line, &e) != nil {
			t.Manifest.Drop("omp.malformed")
			return
		}
		if e.Type == "session" && !headerSeen {
			headerSeen = true
			if e.Version != 3 {
				t.Manifest.Warn("omp session version %d is not the verified version 3.", e.Version)
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
		if e.Type == "title" {
			// Titles arrive out of band (not chained entries); the newest
			// non-empty one names the session.
			if e.Title != "" {
				t.Header.Title = clip(e.Title, 120)
			}
			return
		}
		entries = append(entries, e)
	})
	if err != nil {
		return transcript.Timeline{}, err
	}
	if !headerSeen {
		return transcript.Timeline{}, fmt.Errorf("not an omp session file")
	}
	if t.Header.FormatVersion != "3" {
		return transcript.Timeline{}, ErrUnknownFormat
	}
	live := liveChain(entries)
	if n := len(entries) - len(live); n > 0 {
		t.Manifest.Dropped = addDrop(t.Manifest.Dropped, "omp.branch", n)
	}
	names := map[string]string{}
	for i, e := range live {
		ts := parseTime(e.Timestamp)
		if !ts.IsZero() {
			t.Header.UpdatedAt = ts
		}
		switch e.Type {
		case "model_change":
			// omp records one provider-qualified id ("google/gemini-3.6-flash").
			if before, after, ok := strings.Cut(e.Model, "/"); ok {
				t.Header.Provider, t.Header.Model = before, after
			} else if e.Model != "" {
				t.Header.Model = e.Model
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
				Timestamp  json.Number     `json:"timestamp"`
				Content    json.RawMessage `json:"content"`
				ToolCallID string          `json:"toolCallId"`
				ToolName   string          `json:"toolName"`
				IsError    bool            `json:"isError"`
			}
			if json.Unmarshal(e.Message, &m) != nil {
				t.Manifest.Drop("omp.malformed")
				continue
			}
			if ms, nerr := m.Timestamp.Int64(); nerr == nil && ms > 0 {
				ts = time.UnixMilli(ms).UTC()
				t.Header.UpdatedAt = ts
			}
			g := i + 1
			switch m.Role {
			case "user":
				if text := textBlocks(m.Content, &t.Manifest, "omp.user"); text != "" {
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
						t.Manifest.Drop("omp.assistant." + b.Type)
					}
				}
			case "toolResult":
				name := m.ToolName
				if name == "" {
					name = names[m.ToolCallID]
				}
				t.Events = append(t.Events, transcript.Event{Kind: transcript.KindToolResult, Result: &transcript.ToolResult{CallID: m.ToolCallID, Name: name, Text: textBlocks(m.Content, &t.Manifest, "omp.tool_result"), IsError: m.IsError}, Timestamp: ts, Group: g})
			default:
				t.Manifest.Drop("omp.message." + m.Role)
			}
		case "custom":
			if e.Custom != "" {
				t.Manifest.Drop("omp.custom." + e.Custom)
			}
		default:
			t.Manifest.Drop("omp." + e.Type)
		}
	}
	return t, nil
}

// ForkArgs: `omp --fork <path|id> <prompt>` — absent from omp --help but
// parsed by omp 18.2.11 (src/cli/flag-tables.ts, src/main.ts: a value with
// a slash or .jsonl is a file, anything else an id prefix), and in its
// changelog (#11944). The file is preferred: it cannot match two sessions.
func (OmpSource) ForkArgs(src Ref, prompt, _ string) Fork {
	from := src.Path
	if from == "" {
		from = src.ID
	}
	return Fork{Args: withPrompt([]string{"--fork", from}, prompt)}
}

// PromptArgs: `omp <prompt>` — messages are positional (omp --help,
// 18.2.4); there is no flag that pre-assigns a new session id, so the
// brief handoff starts an unnamed session and omp titles it itself.
func (OmpSource) PromptArgs(prompt, sessionID string) []string {
	return []string{prompt}
}

// liveChain keeps the ancestry of the last entry (file order when ids are
// missing) — the replay omp does on resume.
func liveChain(entries []ompEntry) []ompEntry {
	if len(entries) == 0 || entries[len(entries)-1].ID == "" {
		return entries
	}
	byID := map[string]int{}
	for i, e := range entries {
		if e.ID == "" {
			return entries
		}
		byID[e.ID] = i
	}
	var chain []ompEntry
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
	return chain
}
