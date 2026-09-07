package transcript

import (
	"encoding/json"
	"regexp"
	"strings"
	"testing"
	"time"
)

func msg(role, text string) Event { return Event{Kind: KindMessage, Role: role, Text: text} }
func call(id, name, input string) Event {
	return Event{Kind: KindToolCall, Role: "assistant", Call: &ToolCall{ID: id, Name: name, Input: json.RawMessage(input)}}
}
func result(id, text string) Event {
	return Event{Kind: KindToolResult, Result: &ToolResult{CallID: id, Text: text}}
}

func TestWindowCutsAtLastCompaction(t *testing.T) {
	tl := Timeline{Events: []Event{
		msg("user", "one"),
		{Kind: KindCompaction, Text: "first summary"},
		msg("user", "two"),
		{Kind: KindCompaction, Text: "second summary"},
		msg("user", "three"),
		msg("assistant", "done"),
	}}
	recent, summary := tl.Window(true)
	if summary != "second summary" {
		t.Fatalf("summary = %q", summary)
	}
	if len(recent.Events) != 2 || recent.Events[0].Text != "three" {
		t.Fatalf("recent events = %+v", recent.Events)
	}
	all, summary := tl.Window(false)
	if summary != "" || len(all.Events) != 4 {
		t.Fatalf("all: summary=%q events=%d (compactions must be removed, turns kept)", summary, len(all.Events))
	}
	if !tl.HasCompaction() || recent.HasCompaction() {
		t.Fatal("HasCompaction wrong")
	}
	none, summary := Timeline{Events: []Event{msg("user", "x")}}.Window(true)
	if summary != "" || len(none.Events) != 1 {
		t.Fatal("no compaction: whole timeline, empty summary")
	}
}

func TestRepairPairsAndSynthesizes(t *testing.T) {
	tl := Timeline{Events: []Event{
		msg("user", "go"),
		call("a", "bash", `{"command":"ls"}`),
		call("b", "read", `{"path":"x"}`),
		result("a", "ok"),
		msg("assistant", "next"),
		result("zzz", "orphan"),
		call("c", "write", `{"path":"y"}`),
	}}
	r := tl.Repair()
	kinds := make([]string, 0, len(r.Events))
	for _, e := range r.Events {
		k := string(e.Kind)
		if e.Result != nil {
			k += ":" + e.Result.CallID
		}
		if e.Call != nil {
			k += ":" + e.Call.ID
		}
		kinds = append(kinds, k)
	}
	want := []string{"message", "tool_call:a", "tool_call:b", "tool_result:a", "tool_result:b", "message", "tool_call:c", "tool_result:c"}
	if strings.Join(kinds, ",") != strings.Join(want, ",") {
		t.Fatalf("repair order:\n got %v\nwant %v", kinds, want)
	}
	if r.Events[4].Result.IsError != true || r.Events[4].Result.Name != "read" || r.Events[4].Result.Text == "" {
		t.Fatalf("synthesized result wrong: %+v", r.Events[4].Result)
	}
	if r.Manifest.Dropped["tool_result.orphan"] != 1 || r.Manifest.Dropped["tool_result.synthesized"] != 2 {
		t.Fatalf("manifest = %+v", r.Manifest.Dropped)
	}
	// Idempotent: repairing the repaired timeline changes nothing.
	again := r.Repair()
	if len(again.Events) != len(r.Events) || again.Manifest.Dropped["tool_result.synthesized"] != 2 {
		t.Fatalf("not idempotent: %d events, %+v", len(again.Events), again.Manifest.Dropped)
	}
	// A result that arrives late (after a message) still pairs with its
	// call and the call is not synthesized twice.
	late := Timeline{Events: []Event{call("q", "bash", `{}`), msg("assistant", "hm"), result("q", "late")}}.Repair()
	if late.Manifest.Total() != 0 || late.Events[1].Kind != KindToolResult || late.Events[2].Kind != KindMessage {
		t.Fatalf("late result must be hoisted next to its call: %+v %+v", late.Manifest.Dropped, late.Events)
	}
}

func TestPrepareStripsThinkingAndContext(t *testing.T) {
	tl := Timeline{Events: []Event{
		{Kind: KindContext, Role: "system", Text: "You are"},
		msg("user", "hi"),
		{Kind: KindThinking, Text: "hmm"},
		msg("assistant", "hello"),
		{Kind: KindCompaction, Text: "we said hello"},
	}}
	p := tl.Prepare()
	if len(p.Events) != 3 || p.Manifest.Dropped["thinking"] != 1 || p.Manifest.Dropped["context"] != 1 {
		t.Fatalf("prepare: %d events, %+v", len(p.Events), p.Manifest.Dropped)
	}
	c := p.Counts()
	if c.Messages != 3 || c.Thinking != 0 || c.Context != 0 || c.Compactions != 0 {
		t.Fatalf("counts = %+v", c)
	}
	if last := p.Events[2]; last.Role != "user" || !strings.HasSuffix(last.Text, "previous agent:\nwe said hello") {
		t.Fatalf("a leftover compaction must become a user message: %+v", last)
	}
	if tl.Manifest.Total() != 0 {
		t.Fatalf("Prepare mutated the source manifest: %+v", tl.Manifest.Dropped)
	}
}

func TestHandoffNoteWording(t *testing.T) {
	h := Header{SourceCLI: "claude-code", SourceName: "Claude Code", SourceID: "7a937bf6-8158-4753-81f3-dd81042697ff", Cwd: "/home/goat/picode", Model: "claude-opus-5"}
	now := time.Date(2026, 9, 6, 22, 36, 0, 0, time.UTC)
	got := HandoffNote(h, "", now)
	want := "Handoff from Claude Code running claude-opus-5, session 7a937bf6, 2026-09-06 22:36 UTC, folder /home/goat/picode.\nPiCode translated this conversation from another coding agent. Everything above the last user message is history: files may have changed since, and the tools named in it are that agent's, not yours. Continue from the user's last request; re-read files before editing."
	if got != want {
		t.Fatalf("note:\n%s\nwant:\n%s", got, want)
	}
	withSummary := HandoffNote(h, "We fixed the race.", now)
	if !strings.HasSuffix(withSummary, "Earlier part of the conversation, as summarized by the previous agent:\nWe fixed the race.") {
		t.Fatalf("summary block missing: %s", withSummary)
	}
	if !strings.HasPrefix(HandoffNote(Header{SourceCLI: "codex"}, "", now), "Handoff from codex, session ,") {
		t.Fatal("falls back to the catalog id when no display name is known")
	}
}

func TestBriefDeterministicAndCapped(t *testing.T) {
	var ev []Event
	for i := 0; i < 400; i++ {
		ev = append(ev, msg("user", strings.Repeat("a", 500)), msg("assistant", strings.Repeat("b", 500)))
	}
	tl := Timeline{Header: Header{SourceCLI: "codex", SourceName: "Codex", SourceID: "abc", Cwd: "/w", Model: "gpt"}, Events: ev}
	now := time.Date(2026, 9, 6, 0, 0, 0, 0, time.UTC)
	a, b := Brief(tl, "", now), Brief(tl, "", now)
	if a != b {
		t.Fatal("brief is not deterministic")
	}
	if len(a) > BriefMaxBytes {
		t.Fatalf("brief is %d bytes", len(a))
	}
	small := Brief(Timeline{Header: Header{SourceCLI: "pi"}}, "", now)
	for _, section := range []string{"## Last request\nnone", "## Where it stopped\nnone", "## Recent turns\nnone", "## Files touched\nnone", "## Commands run\nnone"} {
		if !strings.Contains(small, section) {
			t.Fatalf("empty section missing %q in:\n%s", section, small)
		}
	}
	if !strings.HasPrefix(small, "# Handoff from pi · 2026-09-06 · \n") {
		t.Fatalf("header: %q", strings.SplitN(small, "\n", 2)[0])
	}
}

func TestBriefFilesAndCommandsHeuristic(t *testing.T) {
	tl := Timeline{Header: Header{SourceName: "Claude Code"}, Events: []Event{
		msg("user", "please fix the build"),
		call("1", "Edit", `{"file_path":"/w/a.go","old_string":"x","new_string":"y"}`),
		result("1", "ok"),
		call("2", "write", `{"path":"/w/b.go","content":"..."}`),
		result("2", "ok"),
		call("3", "read_file", `{"target_file":"/w/c.go"}`),
		result("3", "..."),
		call("4", "exec_command", `{"cmd":"go test ./..."}`),
		result("4", "ok"),
		call("5", "Bash", `{"command":"make ci"}`),
		result("5", "ok"),
		call("6", "Edit", `{"file_path":"/w/a.go"}`),
		result("6", "ok"),
		call("7", "exec", `"const r = 1"`),
		result("7", "ok"),
		msg("assistant", "Build fixed."),
	}}
	got := Brief(tl, "the race was in the scheduler", time.Now())
	files := regexp.MustCompile(`(?s)## Files touched\n(.*?)\n\n`).FindStringSubmatch(got)
	if len(files) != 2 || files[1] != "- /w/a.go\n- /w/b.go" {
		t.Fatalf("files section = %q", files)
	}
	cmds := regexp.MustCompile(`(?s)## Commands run\n(.*?)\n\n`).FindStringSubmatch(got)
	if len(cmds) != 2 || cmds[1] != "- go test ./...\n- make ci" {
		t.Fatalf("commands section = %q", cmds)
	}
	if !strings.Contains(got, "## Earlier part of the conversation, as summarized by the previous agent\nthe race was in the scheduler\n\n## Last request") {
		t.Fatalf("summary section:\n%s", got)
	}
	if !strings.Contains(got, "## Last request\nplease fix the build") || !strings.Contains(got, "## Where it stopped\nBuild fixed.") {
		t.Fatalf("request/stop sections:\n%s", got)
	}
	if !strings.Contains(got, "- **user:** please fix the build\n- **assistant:** Build fixed.") {
		t.Fatalf("recent turns in order:\n%s", got)
	}
}

func TestIDsAndInputHelpers(t *testing.T) {
	id := NewID()
	if !regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`).MatchString(id) {
		t.Fatalf("not a v4 uuid: %s", id)
	}
	if NewID() == id {
		t.Fatal("ids repeat")
	}
	if ShortID(id) != id[:8] || ShortID("abc") != "abc" {
		t.Fatal("ShortID")
	}
	if len(RandomAlnum(24)) != 24 || len(RandomHex(4)) != 8 {
		t.Fatal("random helpers")
	}
	obj := ToolCall{Input: json.RawMessage(`{"a":1}`)}
	str := ToolCall{Input: json.RawMessage(`"raw text"`)}
	if obj.InputObject() == nil || str.InputObject() != nil {
		t.Fatal("InputObject")
	}
	if obj.InputString() != `{"a":1}` || str.InputString() != "raw text" || (ToolCall{}).InputString() != "" {
		t.Fatalf("InputString: %q %q", obj.InputString(), str.InputString())
	}
	if string(str.ObjectInput()) != `{"input":"raw text"}` || string(obj.ObjectInput()) != `{"a":1}` || string((ToolCall{}).ObjectInput()) != `{}` {
		t.Fatalf("ObjectInput: %s %s", str.ObjectInput(), obj.ObjectInput())
	}
	m := Manifest{}
	m.Drop("b")
	m.Drop("a")
	m.Drop("a")
	if m.Total() != 3 || strings.Join(m.Kinds(), ",") != "a,b" {
		t.Fatalf("manifest helpers: %d %v", m.Total(), m.Kinds())
	}
}
