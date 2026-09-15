package clisession

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/cfpperche/picode/internal/transcript"
)

// Antigravity brief handoff (Fatia 1 of docs/plans/launch-muse-agy.md):
// `--prompt-interactive` runs the initial prompt and continues the
// session. Verified against a real install (Antigravity 1.2.3,
// `agy --help` 2026-09-15). `--prompt` would answer once and exit, so it
// is not the handoff shape; a session id is never pre-assigned, so it is
// ignored, the way OpenCode's is.
func (AgySource) PromptArgs(prompt, _ string) []string {
	return []string{"--prompt-interactive", prompt}
}

// AgyBrainRoot is the store holding one transcript per conversation. The
// directory name is the conversation id, so 1:1 with the summaries index
// (measured 9/9 on a real install, Antigravity 1.2.3). Under AgyTestDB it
// redirects next to the test index, the way the conversations dir does.
func agyBrainRoot() string { return filepath.Join(filepath.Dir(agyDBPath()), "brain") }

// Read projects an Antigravity conversation from the CLI's own transcript
// (brain/<id>/.system_generated/logs/transcript.jsonl): USER_INPUT entries
// are user turns, PLANNER_RESPONSE entries assistant turns, both with ISO
// timestamps and plain-text content. GENERIC / LIST_DIRECTORY entries are
// tool residue (file dumps, listings) and CHECKPOINT / CONVERSATION_HISTORY
// contentless markers, so they stay out; contentless ERROR_MESSAGEs and the
// metadata wrappers around the user text are dropped the same way.
//
// Deliberately not the per-conversation SQLite: step_payload is protobuf
// with no published schema, and community decoders warn its field numbers
// are unversioned (a vendor update blanks the text). The JSONL is what the
// CLI itself writes to read back.
func (AgySource) Read(ctx context.Context, ref Ref) (transcript.Timeline, error) {
	id := strings.TrimSpace(ref.ID)
	if id == "" && ref.Path != "" {
		if base := strings.TrimSuffix(filepath.Base(ref.Path), ".db"); base != "" && base != "." && base != "/" {
			id = base
		}
	}
	if id == "" {
		return transcript.Timeline{}, fmt.Errorf("an Antigravity conversation id is required")
	}
	root := agyBrainRoot()
	path := filepath.Join(root, id, ".system_generated", "logs", "transcript.jsonl")
	if !underRoot(root, path) {
		return transcript.Timeline{}, ErrNotUnderRoot
	}
	t := transcript.Timeline{Header: transcript.Header{SourceCLI: "agy", SourceID: id, SourcePath: path, Cwd: ref.Cwd}}
	group := 0
	err := scanSession(path, ref, func(line []byte) {
		var e struct {
			Source  string `json:"source"`
			Type    string `json:"type"`
			Status  string `json:"status"`
			At      string `json:"created_at"`
			Content string `json:"content"`
		}
		if json.Unmarshal(line, &e) != nil {
			t.Manifest.Drop("agy.malformed")
			return
		}
		at := parseTime(e.At)
		switch e.Source + "/" + e.Type {
		case "USER_EXPLICIT/USER_INPUT":
			text := agyUserText(e.Content)
			if text == "" {
				return
			}
			group++
			if t.Header.Title == "" {
				t.Header.Title = clip(text, 120)
			}
			t.Events = append(t.Events, transcript.Event{Kind: transcript.KindMessage, Role: "user", Text: text, Timestamp: at, Group: group})
		case "MODEL/PLANNER_RESPONSE":
			text := strings.TrimSpace(e.Content)
			if text == "" {
				return
			}
			t.Events = append(t.Events, transcript.Event{Kind: transcript.KindMessage, Role: "assistant", Text: text, Timestamp: at, Group: group})
		case "MODEL/GENERIC", "MODEL/LIST_DIRECTORY":
			t.Manifest.Drop("agy.tool-output")
		}
	})
	if err != nil {
		return transcript.Timeline{}, err
	}
	return t, nil
}

// agyUserText unwraps the <USER_REQUEST> block the CLI files the prompt
// under; the <ADDITIONAL_METADATA> (local time) and <USER_SETTINGS_CHANGE>
// (model switches) siblings describe the run, not the conversation, so a
// transcript without the wrapper travels whole instead of losing the turn.
func agyUserText(content string) string {
	text := content
	if _, after, ok := strings.Cut(content, "<USER_REQUEST>"); ok {
		text = after
	}
	if before, _, ok := strings.Cut(text, "</USER_REQUEST>"); ok {
		text = before
	}
	return strings.TrimSpace(text)
}
