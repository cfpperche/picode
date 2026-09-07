package clisession

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/cfpperche/picode/internal/transcript"
)

// Claude Code writer (ADR-0087): creates
// ~/.claude/projects/<encoded cwd>/<uuid>.jsonl in the shape Claude Code
// 2.1.x writes itself — one record per line, chained by parentUuid, every
// record carrying sessionId, cwd and version; tool_use blocks answered by
// a user record with a tool_result block right after their assistant
// record; unsigned thinking never written (Claude rejects it on the next
// turn). The file is re-read through ClaudeCodeSource.Read before the
// write counts as done.

var claudeNonAlnum = regexp.MustCompile(`[^A-Za-z0-9]`)

// claudeProjectDir is Claude Code's directory name for a folder: every
// byte outside [A-Za-z0-9] becomes "-" (observed: /home/goat/picode →
// -home-goat-picode).
func claudeProjectDir(cwd string) string {
	return claudeNonAlnum.ReplaceAllString(filepath.Clean(cwd), "-")
}

// claudeNewestVersion reads the "version" of the most recent transcript on
// this machine — the format marker the installed CLI last wrote.
func claudeNewestVersion(root string) string {
	newest := newestFile(jsonlFiles(root))
	if newest == "" {
		return ""
	}
	version := ""
	n := 0
	_ = scanLines(newest, func(line []byte) {
		if version != "" || n > 200 {
			return
		}
		n++
		var raw struct {
			Version string `json:"version"`
		}
		if json.Unmarshal(line, &raw) == nil && raw.Version != "" {
			version = raw.Version
		}
	})
	return version
}

func (ClaudeCodeSource) Write(ctx context.Context, t transcript.Timeline, req WriteRequest) (Summary, error) {
	if strings.TrimSpace(req.Cwd) == "" {
		return Summary{}, fmt.Errorf("a folder is required")
	}
	root := filepath.Join(homeDir(), ".claude", "projects")
	version := strings.TrimSpace(req.FormatVersion)
	if version == "" {
		version = claudeNewestVersion(root)
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
	dir := filepath.Join(root, claudeProjectDir(cwd))
	path := filepath.Join(dir, sid+".jsonl")

	var b strings.Builder
	var emitted []transcript.Event
	var werr error
	prev := ""
	seq := 0
	record := func(kind string, at time.Time, message map[string]any) {
		if werr != nil {
			return
		}
		seq++
		id := transcript.NewID()
		line := map[string]any{
			"parentUuid":  nil,
			"isSidechain": false,
			"userType":    "external",
			"cwd":         cwd,
			"sessionId":   sid,
			"version":     version,
			"entrypoint":  "cli",
			"type":        kind,
			"message":     message,
			"uuid":        id,
			"timestamp":   stamp(at, now.Add(time.Duration(seq)*time.Millisecond)).UTC().Format("2006-01-02T15:04:05.000Z"),
		}
		if prev != "" {
			line["parentUuid"] = prev
		}
		prev = id
		werr = jsonLine(&b, line)
	}
	ids := map[string]string{}
	toolID := func(src string) string {
		if id, ok := ids[src]; ok {
			return id
		}
		id := "toolu_" + transcript.RandomAlnum(24)
		ids[src] = id
		return id
	}
	userText := func(text string, at time.Time) {
		record("user", at, map[string]any{"role": "user", "content": []any{map[string]any{"type": "text", "text": text}}})
		emitted = append(emitted, transcript.Event{Kind: transcript.KindMessage, Role: "user"})
	}
	model := t.Header.Model
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
					content = append(content, map[string]any{"type": "tool_use", "id": toolID(blk.Call.ID), "name": blk.Call.Name, "input": blk.Call.ObjectInput()})
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
			if m == "" {
				m = "unknown"
			}
			stop := "end_turn"
			if hasCall {
				stop = "tool_use"
			}
			record("assistant", turn.Timestamp, map[string]any{
				"id":            "msg_" + transcript.RandomAlnum(24),
				"type":          "message",
				"role":          "assistant",
				"model":         m,
				"content":       content,
				"stop_reason":   stop,
				"stop_sequence": nil,
				"usage":         map[string]any{"input_tokens": 0, "output_tokens": 0},
			})
		},
		func(e transcript.Event) {
			if req.textTools() {
				userText(toolResultText(e.Result), e.Timestamp)
				return
			}
			record("user", e.Timestamp, map[string]any{"role": "user", "content": []any{map[string]any{
				"tool_use_id": toolID(e.Result.CallID),
				"type":        "tool_result",
				"content":     e.Result.Text,
				"is_error":    e.Result.IsError,
			}}})
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
	if err := roundTrip(ctx, ClaudeCodeSource{}, ref, transcript.Timeline{Events: emitted}.Counts()); err != nil {
		_ = os.Remove(path)
		return Summary{}, err
	}
	return Summary{
		CLI:        "claude-code",
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
