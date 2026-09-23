package cliinstructions

import (
	"path/filepath"
	"reflect"
	"testing"
)

// The records below keep the shape measured on 2026-09-23: a Claude Code
// transcript line carrying the instructions reminder, a Codex rollout's
// injected block, and a Grok session's prompt_context.json.
func TestObservedFiles(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, map[string]string{
		"claude.jsonl": `{"type":"user","message":{"content":"<system-reminder>Contents of /repo/AGENTS.md (project instructions, checked into the codebase):\n...Contents of /home/me/.claude/CLAUDE.md (user's private global instructions):"}}` + "\n" +
			`{"type":"user","message":{"content":"Contents of /home/me/.claude/projects/x/memory/MEMORY.md (user's auto-memory, persists)"}}` + "\n" +
			`{"type":"assistant","message":{"content":"ok"}}` + "\n" +
			`{"type":"user","message":{"content":"quoting Contents of /elsewhere/AGENTS.md (project instructions) later"}}` + "\n",
		"rollout.jsonl":               `{"type":"message","content":[{"type":"input_text","text":"# AGENTS.md instructions for /repo\n\n<INSTRUCTIONS>..."}]}` + "\n",
		"grok/prompt_history.jsonl":   `{"session_id":"s1","prompt":"hi"}` + "\n",
		"grok/s1/prompt_context.json": `{"agents_md_files":[{"file_name":"AGENTS.md","file_path":"/repo/AGENTS.md","source":"project"}]}`,
		"grok/s2/prompt_context.json": `not json`,
	})
	cases := []struct {
		cli, session, path string
		want               []string
		ok                 bool
	}{
		{"claude-code", "", filepath.Join(dir, "claude.jsonl"), []string{"/repo/AGENTS.md", "/home/me/.claude/CLAUDE.md"}, true},
		{"codex", "", filepath.Join(dir, "rollout.jsonl"), []string{"/repo/AGENTS.md"}, true},
		{"grok", "s1", filepath.Join(dir, "grok", "prompt_history.jsonl"), []string{"/repo/AGENTS.md"}, true},
		{"grok", "s2", filepath.Join(dir, "grok", "prompt_history.jsonl"), nil, false},
		{"claude-code", "", filepath.Join(dir, "missing.jsonl"), nil, false},
		{"pi", "", filepath.Join(dir, "claude.jsonl"), nil, false},
	}
	for _, c := range cases {
		got, ok := ObservedFiles(c.cli, c.session, c.path)
		if ok != c.ok || (c.ok && !reflect.DeepEqual(got, c.want)) {
			t.Errorf("%s %s: got %v %v, want %v %v", c.cli, c.session, got, ok, c.want, c.ok)
		}
	}
	if !Records("grok") || Records("pi") {
		t.Fatal("Records: Claude Code, Codex and Grok only")
	}
	for abs, want := range map[string]string{"/w/AGENTS.md": "AGENTS.md", "/h/.claude/CLAUDE.md": "~/.claude/CLAUDE.md", "/etc/x.md": "/etc/x.md"} {
		if got := Display("/w", "/h", abs); got != want {
			t.Errorf("Display(%s) = %s, want %s", abs, got, want)
		}
	}
}
