package clisession

import (
	"context"
	"encoding/json"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/cfpperche/picode/internal/transcript"
)

// Grok Build session store ↔ timeline (ADR-0088). Format verified against
// a real installation (Grok 1.0.13, 2026-09):
//
//	~/.grok/sessions/<url-escaped cwd>/prompt_history.jsonl   prompts of every session in the folder
//	~/.grok/sessions/<url-escaped cwd>/<uuid>/summary.json    info.{id,cwd}, session_summary, created_at,
//	                                                          updated_at, num_chat_messages, current_model_id,
//	                                                          chat_format_version
//	~/.grok/sessions/<url-escaped cwd>/<uuid>/chat_history.jsonl  the transcript:
//	   {type:system,content}
//	   {type:user,content:[{type:text,text}],synthetic_reason?}   human turns wrap the text in <user_query>
//	   {type:assistant,content:<string>,tool_calls:[{id,name,arguments:<JSON string>}],model_id}
//	   {type:tool_result,tool_call_id,content:<string>}
//	   {type:reasoning,summary:[{text}],encrypted_content}
//	   {type:backend_tool_call,…}
//
// updates.jsonl (the ACP update log), events.jsonl and the rest are
// runtime state; the reader never opens them.

// grokSessionDir resolves the per-session directory for a Ref: the row's
// Path when it already is that directory, else derived from the folder
// and the id the same way Grok names it.
func grokSessionDir(root string, ref Ref) string {
	if ref.Path != "" {
		if st, err := os.Stat(ref.Path); err == nil && st.IsDir() {
			return ref.Path
		}
		if filepath.Base(ref.Path) == "prompt_history.jsonl" && ref.ID != "" {
			return filepath.Join(filepath.Dir(ref.Path), ref.ID)
		}
	}
	if ref.Cwd != "" && ref.ID != "" {
		return filepath.Join(root, url.PathEscape(ref.Cwd), ref.ID)
	}
	return ""
}

type grokSummaryFile struct {
	Info struct {
		ID  string `json:"id"`
		Cwd string `json:"cwd"`
	} `json:"info"`
	SessionSummary  string `json:"session_summary"`
	CreatedAt       string `json:"created_at"`
	UpdatedAt       string `json:"updated_at"`
	NumChatMessages int    `json:"num_chat_messages"`
	CurrentModelID  string `json:"current_model_id"`
	ChatFormat      int    `json:"chat_format_version"`
}

func readGrokSummary(dir string) (grokSummaryFile, bool) {
	var s grokSummaryFile
	b, err := os.ReadFile(filepath.Join(dir, "summary.json"))
	if err != nil || json.Unmarshal(b, &s) != nil || s.Info.ID == "" {
		return s, false
	}
	return s, true
}

// grokFolder lists one folder's sessions: the prompt-history fold (which
// knows every session that ever ran there) enriched by each session
// directory's summary.json (title, model, message count, the real
// transcript size) when Grok kept one.
func grokFolder(root, encName, dir string) []Summary {
	folder := filepath.Join(root, encName)
	rows := grokSessions(filepath.Join(folder, "prompt_history.jsonl"), dir)
	byID := map[string]int{}
	for i := range rows {
		byID[rows[i].ID] = i
	}
	ents, err := os.ReadDir(folder)
	if err != nil {
		return rows
	}
	for _, e := range ents {
		if !e.IsDir() {
			continue
		}
		sdir := filepath.Join(folder, e.Name())
		sum, ok := readGrokSummary(sdir)
		if !ok {
			continue
		}
		created, updated := rfc3339(parseTime(sum.CreatedAt)), rfc3339(parseTime(sum.UpdatedAt))
		size := fileSize(filepath.Join(sdir, "chat_history.jsonl"))
		if i, seen := byID[sum.Info.ID]; seen {
			r := &rows[i]
			r.Path = sdir
			if sum.SessionSummary != "" {
				r.Name = clip(sum.SessionSummary, 120)
			}
			if sum.CurrentModelID != "" {
				r.Model = sum.CurrentModelID
			}
			if sum.NumChatMessages > 0 {
				r.Messages = sum.NumChatMessages
			}
			if updated > r.UpdatedAt {
				r.UpdatedAt = updated
			}
			if size > 0 {
				r.Size = size
			}
			continue
		}
		cwd := sum.Info.Cwd
		if cwd == "" {
			cwd = dir
		}
		if updated == "" {
			updated = mtime(filepath.Join(sdir, "chat_history.jsonl"))
		}
		if updated == "" {
			continue
		}
		rows = append(rows, Summary{
			CLI:        "grok",
			ID:         sum.Info.ID,
			Path:       sdir,
			ResumeArgs: []string{"--resume", sum.Info.ID},
			Name:       clip(sum.SessionSummary, 120),
			Cwd:        cwd,
			CreatedAt:  created,
			UpdatedAt:  updated,
			Messages:   sum.NumChatMessages,
			Size:       size,
			Model:      sum.CurrentModelID,
		})
	}
	sortNewest(rows)
	return rows
}

func (GrokSource) Read(ctx context.Context, ref Ref) (transcript.Timeline, error) {
	root := grokSessionsRoot()
	dir := grokSessionDir(root, ref)
	if dir == "" || !underRoot(root, dir) {
		return transcript.Timeline{}, ErrNotUnderRoot
	}
	if st, err := os.Stat(dir); err != nil || !st.IsDir() {
		return transcript.Timeline{}, ErrNotUnderRoot
	}
	t := transcript.Timeline{Header: transcript.Header{SourceCLI: "grok", SourcePath: dir, SourceID: ref.ID, Cwd: ref.Cwd}}
	if sum, ok := readGrokSummary(dir); ok {
		t.Header.SourceID = sum.Info.ID
		if sum.Info.Cwd != "" {
			t.Header.Cwd = sum.Info.Cwd
		}
		t.Header.Title = clip(sum.SessionSummary, 120)
		t.Header.Model = sum.CurrentModelID
		t.Header.CreatedAt = parseTime(sum.CreatedAt)
		t.Header.UpdatedAt = parseTime(sum.UpdatedAt)
		if sum.ChatFormat != 0 {
			t.Header.FormatVersion = "chat_format_version " + strconv.Itoa(sum.ChatFormat)
		}
	}
	if t.Header.Cwd == "" {
		t.Header.Cwd = decodePathDir(filepath.Base(filepath.Dir(dir)))
	}
	group := 0
	names := map[string]string{}
	err := scanLines(filepath.Join(dir, "chat_history.jsonl"), func(line []byte) {
		var raw struct {
			Type      string          `json:"type"`
			Content   json.RawMessage `json:"content"`
			Synthetic string          `json:"synthetic_reason"`
			ToolCalls []struct {
				ID        string          `json:"id"`
				Name      string          `json:"name"`
				Arguments json.RawMessage `json:"arguments"`
			} `json:"tool_calls"`
			ToolCallID string          `json:"tool_call_id"`
			ModelID    string          `json:"model_id"`
			Summary    json.RawMessage `json:"summary"`
			IsError    bool            `json:"is_error"`
		}
		if json.Unmarshal(line, &raw) != nil {
			t.Manifest.Drop("grok.malformed")
			return
		}
		group++
		switch raw.Type {
		case "system":
			t.Events = append(t.Events, transcript.Event{Kind: transcript.KindContext, Role: "system", Text: textBlocks(raw.Content, nil, ""), Group: group})
		case "user":
			text := textBlocks(raw.Content, &t.Manifest, "grok.user")
			if text == "" {
				return
			}
			if raw.Synthetic != "" || strings.HasPrefix(text, "<user_info>") || strings.HasPrefix(text, "<system-reminder>") {
				t.Events = append(t.Events, transcript.Event{Kind: transcript.KindContext, Role: "user", Text: text, Group: group})
				return
			}
			t.Events = append(t.Events, transcript.Event{Kind: transcript.KindMessage, Role: "user", Text: grokUnwrapQuery(text), Group: group})
		case "assistant":
			if raw.ModelID != "" {
				t.Header.Model = raw.ModelID
			}
			if text := textBlocks(raw.Content, &t.Manifest, "grok.assistant"); text != "" {
				t.Events = append(t.Events, transcript.Event{Kind: transcript.KindMessage, Role: "assistant", Text: text, Model: raw.ModelID, Group: group})
			}
			for _, c := range raw.ToolCalls {
				names[c.ID] = c.Name
				t.Events = append(t.Events, transcript.Event{Kind: transcript.KindToolCall, Role: "assistant", Call: &transcript.ToolCall{ID: c.ID, Name: c.Name, Input: argumentsJSON(c.Arguments)}, Model: raw.ModelID, Group: group})
			}
		case "tool_result":
			t.Events = append(t.Events, transcript.Event{Kind: transcript.KindToolResult, Result: &transcript.ToolResult{CallID: raw.ToolCallID, Name: names[raw.ToolCallID], Text: textBlocks(raw.Content, &t.Manifest, "grok.tool_result"), IsError: raw.IsError}, Group: group})
		case "reasoning":
			if text := textBlocks(raw.Summary, nil, ""); text != "" {
				t.Events = append(t.Events, transcript.Event{Kind: transcript.KindThinking, Role: "assistant", Text: text, Group: group})
			} else {
				t.Manifest.Drop("reasoning.encrypted")
			}
		default:
			t.Manifest.Drop("grok." + raw.Type)
		}
	})
	if err != nil {
		return transcript.Timeline{}, err
	}
	return t, nil
}

// grokUnwrapQuery strips the <user_query> envelope Grok puts around a
// human turn; text without the envelope is returned as is.
func grokUnwrapQuery(text string) string {
	s := strings.TrimSpace(text)
	if strings.HasPrefix(s, "<user_query>") && strings.HasSuffix(s, "</user_query>") {
		return strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(s, "<user_query>"), "</user_query>"))
	}
	return s
}

// PromptArgs: `grok --session-id <uuid> <prompt>` — verified flags (grok
// --help, 1.0.13); the prompt is positional.
func (GrokSource) PromptArgs(prompt, sessionID string) []string {
	if sessionID == "" {
		return []string{prompt}
	}
	return []string{"--session-id", sessionID, prompt}
}
