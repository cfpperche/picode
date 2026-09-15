package clisession

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/cfpperche/picode/internal/session"
	"github.com/cfpperche/picode/internal/transcript"
)

// Fixtures below are abbreviated from real installations (2026-09); each
// keeps the awkward cases the readers must survive.

func write(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func kinds(tl transcript.Timeline) string {
	var out []string
	for _, e := range tl.Events {
		k := string(e.Kind)
		if e.Kind == transcript.KindMessage || e.Kind == transcript.KindContext {
			k += "/" + e.Role
		}
		out = append(out, k)
	}
	return strings.Join(out, " ")
}

func seedClaudeTranscript(t *testing.T, home, proj string) string {
	dir := filepath.Join(home, ".claude", "projects", claudeProjectDir(proj))
	path := filepath.Join(dir, "cc-1.jsonl")
	write(t, path, strings.Join([]string{
		`{"type":"queue-operation","operation":"enqueue","timestamp":"2026-09-01T10:00:00.000Z","sessionId":"cc-1"}`,
		`{"type":"summary","summary":"Fix the race in the scheduler","leafUuid":"x"}`,
		`{"type":"user","isMeta":true,"message":{"role":"user","content":"<command-name>/clear</command-name>"},"timestamp":"2026-09-01T10:00:01.000Z","cwd":"` + proj + `","sessionId":"cc-1","version":"2.1.263","uuid":"u0"}`,
		`{"type":"user","message":{"role":"user","content":[{"type":"text","text":"<system-reminder>\ninjected</system-reminder>"},{"type":"text","text":"fix the race"}]},"timestamp":"2026-09-01T10:00:02.000Z","cwd":"` + proj + `","sessionId":"cc-1","version":"2.1.263","uuid":"u1"}`,
		`{"type":"assistant","message":{"id":"msg_1","role":"assistant","model":"claude-sonnet-5","content":[{"type":"thinking","thinking":"let me look","signature":"sig"}]},"timestamp":"2026-09-01T10:00:03.000Z","sessionId":"cc-1","cwd":"` + proj + `","uuid":"a1","parentUuid":"u1"}`,
		`{"type":"assistant","message":{"id":"msg_1","role":"assistant","model":"claude-sonnet-5","content":[{"type":"text","text":"On it."}]},"timestamp":"2026-09-01T10:00:03.100Z","sessionId":"cc-1","cwd":"` + proj + `","uuid":"a2","parentUuid":"a1"}`,
		`{"type":"assistant","message":{"id":"msg_1","role":"assistant","model":"claude-sonnet-5","content":[{"type":"tool_use","id":"toolu_1","name":"Bash","input":{"command":"go test ./..."}}]},"timestamp":"2026-09-01T10:00:03.200Z","sessionId":"cc-1","cwd":"` + proj + `","uuid":"a3","parentUuid":"a2"}`,
		`{"type":"user","message":{"role":"user","content":[{"tool_use_id":"toolu_1","type":"tool_result","content":[{"type":"text","text":"ok"}]}]},"timestamp":"2026-09-01T10:00:04.000Z","sessionId":"cc-1","cwd":"` + proj + `","uuid":"u2","parentUuid":"a3","toolUseResult":{"stdout":"ok"}}`,
		`{"type":"assistant","isSidechain":true,"message":{"id":"msg_s","role":"assistant","model":"claude-haiku-4-5","content":[{"type":"text","text":"subagent noise"}]},"timestamp":"2026-09-01T10:00:05.000Z","sessionId":"cc-1","cwd":"` + proj + `"}`,
		`{"type":"system","subtype":"compact_boundary","content":"Conversation compacted","timestamp":"2026-09-01T10:00:06.000Z","sessionId":"cc-1","cwd":"` + proj + `"}`,
		`{"type":"user","isCompactSummary":true,"message":{"role":"user","content":"This session is being continued from a previous conversation. Summary: the race was in the scheduler."},"timestamp":"2026-09-01T10:00:06.100Z","sessionId":"cc-1","cwd":"` + proj + `","uuid":"u3"}`,
		`{"type":"user","message":{"role":"user","content":[{"type":"text","text":"now add a test"},{"type":"image","source":{}}]},"timestamp":"2026-09-01T10:00:07.000Z","sessionId":"cc-1","cwd":"` + proj + `","uuid":"u4"}`,
		`{"type":"assistant","message":{"id":"msg_2","role":"assistant","model":"claude-sonnet-5","content":[{"type":"tool_use","id":"toolu_2","name":"Edit","input":"raw"}]},"timestamp":"2026-09-01T10:00:08.000Z","sessionId":"cc-1","cwd":"` + proj + `","uuid":"a4"}`,
		`{"type":"user","message":{"role":"user","content":[{"type":"text","text":"[Request interrupted by user for tool use]"}]},"timestamp":"2026-09-01T10:00:09.000Z","sessionId":"cc-1","cwd":"` + proj + `","uuid":"u5"}`,
		`{"type":"file-history-snapshot","messageId":"x","snapshot":{}}`,
	}, "\n"))
	return path
}

func TestCapabilitiesOf(t *testing.T) {
	cases := map[string]Capabilities{
		"pi":          {List: true, Read: true, Write: true, Prompt: true},
		"claude-code": {List: true, Read: true, Write: true, Prompt: true},
		"codex":       {List: true, Read: true, Write: true, Prompt: true},
		"grok":        {List: true, Read: true, Write: true, Prompt: true},
		"hermes":      {List: true, Read: true, Write: true, Prompt: false},
		"opencode":    {List: true, Read: true, Write: true, Prompt: true},
		"nope":        {},
	}
	for cli, want := range cases {
		if got := CapabilitiesOf(cli); got != want {
			t.Errorf("%s: %+v, want %+v", cli, got, want)
		}
	}
}

func TestClaudeReadMapsBlocks(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	proj := "/home/goat/proj"
	path := seedClaudeTranscript(t, home, proj)
	tl, err := (ClaudeCodeSource{}).Read(context.Background(), Ref{ID: "cc-1", Path: path, Cwd: proj})
	if err != nil {
		t.Fatal(err)
	}
	want := "context/user context/user message/user thinking message/assistant tool_call tool_result compaction message/user tool_call context/user"
	if got := kinds(tl); got != want {
		t.Fatalf("kinds:\n got %s\nwant %s", got, want)
	}
	h := tl.Header
	if h.SourceID != "cc-1" || h.Cwd != proj || h.Title != "Fix the race in the scheduler" || h.FormatVersion != "2.1.263" || h.Model != "claude-sonnet-5" {
		t.Fatalf("header = %+v", h)
	}
	call := tl.Events[5].Call
	if call.ID != "toolu_1" || call.Name != "Bash" || string(call.Input) != `{"command":"go test ./..."}` {
		t.Fatalf("tool call = %+v", call)
	}
	if r := tl.Events[6].Result; r.CallID != "toolu_1" || r.Name != "Bash" || r.Text != "ok" {
		t.Fatalf("tool result = %+v", r)
	}
	if !strings.HasPrefix(tl.Events[7].Text, "This session is being continued") {
		t.Fatalf("compaction text = %q", tl.Events[7].Text)
	}
	// Blocks of one message.id share a Group; the raw-string input is kept as a JSON string.
	if tl.Events[3].Group != tl.Events[4].Group || tl.Events[4].Group != tl.Events[5].Group {
		t.Fatal("assistant blocks of msg_1 must share a group")
	}
	if string(tl.Events[9].Call.Input) != `"raw"` {
		t.Fatalf("non-object input = %s", tl.Events[9].Call.Input)
	}
	d := tl.Manifest.Dropped
	if d["claude.sidechain"] != 1 || d["image"] != 1 || d["claude.queue-operation"] != 1 || d["claude.file-history-snapshot"] != 1 {
		t.Fatalf("manifest = %+v", d)
	}
	// Reading never rewrites the source.
	before, _ := os.ReadFile(path)
	st, _ := os.Stat(path)
	_, _ = (ClaudeCodeSource{}).Read(context.Background(), Ref{ID: "cc-1", Path: path, Cwd: proj})
	after, _ := os.ReadFile(path)
	st2, _ := os.Stat(path)
	if string(before) != string(after) || !st.ModTime().Equal(st2.ModTime()) {
		t.Fatal("reader rewrote the native file")
	}
}

func seedCodexRollout(t *testing.T, home, proj string) string {
	path := filepath.Join(home, ".codex", "sessions", "2026", "09", "04", "rollout-2026-09-04T16-05-50-cx-1.jsonl")
	write(t, path, strings.Join([]string{
		`{"timestamp":"2026-09-04T16:05:50.994Z","type":"session_meta","payload":{"id":"cx-1","session_id":"cx-1","timestamp":"2026-09-04T16:05:22.418Z","cwd":"` + proj + `","originator":"codex-tui","cli_version":"0.153.2","source":"cli","model_provider":"openai"}}`,
		`{"timestamp":"2026-09-04T16:05:50.995Z","type":"event_msg","payload":{"type":"task_started"}}`,
		`{"timestamp":"2026-09-04T16:05:51.368Z","type":"response_item","payload":{"type":"message","role":"developer","content":[{"type":"input_text","text":"## Memory\nyou have"}]}}`,
		`{"timestamp":"2026-09-04T16:05:51.369Z","type":"response_item","payload":{"type":"message","role":"user","content":[{"type":"input_text","text":"# AGENTS.md instructions for /home/goat/proj\n..."}]}}`,
		`{"timestamp":"2026-09-04T16:05:51.370Z","type":"response_item","payload":{"type":"message","role":"user","content":[{"type":"input_text","text":"<environment_context>\n  <cwd>/x</cwd>\n</environment_context>"}]}}`,
		`{"timestamp":"2026-09-04T16:05:51.374Z","type":"turn_context","payload":{"cwd":"` + proj + `","model":"gpt-5.6-sol"}}`,
		`{"timestamp":"2026-09-04T16:05:51.375Z","type":"response_item","payload":{"type":"message","role":"user","content":[{"type":"input_text","text":"# Plan\nfix the race"}]}}`,
		`{"timestamp":"2026-09-04T16:06:06.180Z","type":"response_item","payload":{"type":"reasoning","summary":[],"encrypted_content":"gAAAA"}}`,
		`{"timestamp":"2026-09-04T16:06:06.811Z","type":"response_item","payload":{"type":"message","role":"assistant","content":[{"type":"output_text","text":"Looking."}]}}`,
		`{"timestamp":"2026-09-04T16:06:07.241Z","type":"response_item","payload":{"type":"custom_tool_call","status":"completed","call_id":"call_1","name":"exec","input":"const r = 1"}}`,
		`{"timestamp":"2026-09-04T16:06:07.391Z","type":"response_item","payload":{"type":"custom_tool_call_output","call_id":"call_1","output":[{"type":"input_text","text":"Script completed"}]}}`,
		`{"timestamp":"2026-09-04T16:06:08.000Z","type":"response_item","payload":{"type":"function_call","name":"shell","arguments":"{\"command\":[\"go\",\"test\"]}","call_id":"call_2"}}`,
		`{"timestamp":"2026-09-04T16:06:09.000Z","type":"response_item","payload":{"type":"function_call_output","call_id":"call_2","output":"ok"}}`,
		`{"timestamp":"2026-09-04T16:06:09.500Z","type":"response_item","payload":{"type":"reasoning","summary":[{"type":"summary_text","text":"The tests pass."}],"encrypted_content":"x"}}`,
		`{"timestamp":"2026-09-04T16:14:52.486Z","type":"compacted","payload":{"message":"","replacement_history":[]}}`,
		`{"timestamp":"2026-09-04T16:15:00.000Z","type":"response_item","payload":{"type":"message","role":"user","content":[{"type":"input_text","text":"and the docs"}]}}`,
		`{"timestamp":"2026-09-04T16:15:01.000Z","type":"token_usage_record","payload":{}}`,
	}, "\n"))
	return path
}

func TestCodexReadMapsItems(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	proj := "/home/goat/proj"
	path := seedCodexRollout(t, home, proj)
	tl, err := (CodexSource{}).Read(context.Background(), Ref{ID: "cx-1", Path: path, Cwd: proj})
	if err != nil {
		t.Fatal(err)
	}
	want := "context/developer context/user context/user message/user message/assistant tool_call tool_result tool_call tool_result thinking compaction message/user"
	if got := kinds(tl); got != want {
		t.Fatalf("kinds:\n got %s\nwant %s", got, want)
	}
	if tl.Header.SourceID != "cx-1" || tl.Header.FormatVersion != "0.153.2" || tl.Header.Model != "gpt-5.6-sol" || tl.Header.Provider != "openai" || tl.Header.Cwd != proj {
		t.Fatalf("header = %+v", tl.Header)
	}
	if tl.Events[3].Text != "# Plan\nfix the race" {
		t.Fatal("a human turn that starts with a heading is not injected context")
	}
	if c := tl.Events[5].Call; c.ID != "call_1" || c.Name != "exec" || string(c.Input) != `"const r = 1"` {
		t.Fatalf("custom_tool_call = %+v", c)
	}
	if c := tl.Events[7].Call; string(c.Input) != `{"command":["go","test"]}` {
		t.Fatalf("function_call arguments string must become an object: %s", c.Input)
	}
	if r := tl.Events[8].Result; r.CallID != "call_2" || r.Name != "shell" || r.Text != "ok" {
		t.Fatalf("output = %+v", r)
	}
	if tl.Events[10].Text != "Session compacted." {
		t.Fatalf("empty compacted message: %q", tl.Events[10].Text)
	}
	d := tl.Manifest.Dropped
	if d["reasoning.encrypted"] != 1 || d["codex.event_msg"] != 1 || d["codex.token_usage_record"] != 1 {
		t.Fatalf("manifest = %+v", d)
	}
}

func seedGrokSession(t *testing.T, home, proj, id string) string {
	folder := filepath.Join(home, ".grok", "sessions", "%2Fhome%2Fgoat%2Fproj")
	write(t, filepath.Join(folder, "prompt_history.jsonl"), `{"timestamp":"2026-08-13T00:35:27.579026410Z","session_id":"`+id+`","prompt":"first prompt","is_bash":false}`+"\n")
	dir := filepath.Join(folder, id)
	write(t, filepath.Join(dir, "summary.json"), `{"info":{"id":"`+id+`","cwd":"`+proj+`"},"session_summary":"Race fix","created_at":"2026-08-13T00:35:27.579026410","updated_at":"2026-08-13T01:00:00.000000000","num_messages":9,"num_chat_messages":4,"current_model_id":"grok-4.6","chat_format_version":1}`)
	write(t, filepath.Join(dir, "chat_history.jsonl"), strings.Join([]string{
		`{"type":"system","content":"You are Grok"}`,
		`{"type":"user","content":[{"type":"text","text":"<user_info>\nOS: linux\n</user_info>"}]}`,
		`{"type":"user","content":[{"type":"text","text":"<system-reminder>\nx</system-reminder>"}],"synthetic_reason":"system_reminder"}`,
		`{"type":"user","content":[{"type":"text","text":"<user_query>\nfirst prompt\n</user_query>"}],"prompt_index":0}`,
		`{"type":"reasoning","id":"rs_1","summary":[{"type":"summary_text","text":"thinking"}],"encrypted_content":"x"}`,
		`{"type":"assistant","content":"Checking.","tool_calls":[{"id":"call-1","name":"read_file","arguments":"{\"target_file\":\"a.go\"}"}],"model_id":"grok-4.6-build"}`,
		`{"type":"tool_result","tool_call_id":"call-1","content":"{\"results\":[]}"}`,
		`{"type":"backend_tool_call","kind":{"tool_type":"web_search"}}`,
		`{"type":"assistant","content":"Done.","model_id":"grok-4.6-build"}`,
	}, "\n"))
	return dir
}

func TestGrokReadFromSessionDirAndListMerges(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	proj := "/home/goat/proj"
	dir := seedGrokSession(t, home, proj, "019ff88b-7d95-7241-be51-acb541c88ef2")

	rows, err := (GrokSource{}).List(proj)
	if err != nil || len(rows) != 1 {
		t.Fatalf("list: %v %+v", err, rows)
	}
	r := rows[0]
	if r.Path != dir || r.Name != "Race fix" || r.Model != "grok-4.6" || r.Messages != 4 || r.Preview != "first prompt" || r.UpdatedAt != "2026-08-13T01:00:00Z" {
		t.Fatalf("merged row = %+v", r)
	}

	for _, ref := range []Ref{{ID: r.ID, Path: dir, Cwd: proj}, {ID: r.ID, Cwd: proj}, {ID: r.ID, Path: filepath.Join(filepath.Dir(dir), "prompt_history.jsonl")}} {
		tl, err := (GrokSource{}).Read(context.Background(), ref)
		if err != nil {
			t.Fatalf("read %+v: %v", ref, err)
		}
		want := "context/system context/user context/user message/user thinking message/assistant tool_call tool_result message/assistant"
		if got := kinds(tl); got != want {
			t.Fatalf("kinds:\n got %s\nwant %s", got, want)
		}
		if tl.Events[3].Text != "first prompt" {
			t.Fatalf("<user_query> not unwrapped: %q", tl.Events[3].Text)
		}
		if c := tl.Events[6].Call; c.Name != "read_file" || string(c.Input) != `{"target_file":"a.go"}` {
			t.Fatalf("call = %+v", c)
		}
		if tl.Header.Title != "Race fix" || tl.Header.Model != "grok-4.6-build" || tl.Header.FormatVersion != "chat_format_version 1" || tl.Header.SourceID != r.ID {
			t.Fatalf("header = %+v", tl.Header)
		}
		if tl.Manifest.Dropped["grok.backend_tool_call"] != 1 {
			t.Fatalf("manifest = %+v", tl.Manifest.Dropped)
		}
	}
	if _, err := (GrokSource{}).Read(context.Background(), Ref{ID: "missing", Cwd: proj}); !errors.Is(err, ErrNotUnderRoot) {
		t.Fatalf("missing dir: %v", err)
	}
}

func seedPiSession(t *testing.T, root, proj string, body string) string {
	dir := filepath.Join(root, session.DirName(proj))
	path := filepath.Join(dir, "2026-09-05T22-01-52-865Z_pi-1.jsonl")
	write(t, path, body)
	return path
}

const piFixture = `{"type":"session","version":3,"id":"pi-1","timestamp":"2026-09-05T22:01:52.865Z","cwd":"/home/goat/proj"}
{"type":"model_change","id":"e1","parentId":null,"timestamp":"2026-09-05T22:01:53.227Z","provider":"zai","modelId":"glm-5.3-flash"}
{"type":"thinking_level_change","id":"e2","parentId":"e1","timestamp":"2026-09-05T22:01:53.227Z","thinkingLevel":"high"}
{"type":"session_info","id":"e3","parentId":"e2","timestamp":"2026-09-05T22:01:54.000Z","name":"race fix"}
{"type":"message","id":"e4","parentId":"e3","timestamp":"2026-09-05T22:02:30.355Z","message":{"role":"user","content":[{"type":"text","text":"fix the race"}],"timestamp":1788645750288}}
{"type":"message","id":"e5","parentId":"e4","timestamp":"2026-09-05T22:02:51.014Z","message":{"role":"assistant","content":[{"type":"thinking","thinking":"hmm","thinkingSignature":"reasoning_content"},{"type":"text","text":"Looking."},{"type":"toolCall","id":"call_a","name":"bash","arguments":{"command":"go test"}}],"api":"openai-completions","provider":"zai","model":"glm-5.3-flash","usage":{},"stopReason":"toolUse","timestamp":1788645771000}}
{"type":"message","id":"e6","parentId":"e5","timestamp":"2026-09-05T22:02:52.327Z","message":{"role":"toolResult","toolCallId":"call_a","toolName":"bash","content":[{"type":"text","text":"ok"}],"isError":false,"timestamp":1788645772327}}
{"type":"message","id":"e7","parentId":"e6","timestamp":"2026-09-05T22:03:00.000Z","message":{"role":"assistant","content":[{"type":"text","text":"abandoned branch"}],"provider":"zai","model":"glm-5.3-flash","stopReason":"stop","timestamp":1788645780000}}
{"type":"compaction","id":"e8","parentId":"e6","timestamp":"2026-09-05T22:04:00.000Z","summary":"# Summary\nrace fixed","firstKeptEntryId":"e6"}
{"type":"message","id":"e9","parentId":"e8","timestamp":"2026-09-05T22:05:00.000Z","message":{"role":"assistant","content":[{"type":"text","text":"Done."}],"provider":"zai","model":"glm-5.3-flash","stopReason":"stop","timestamp":1788645900000}}
`

func TestPiReadMapsEntries(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	old := session.TestRoot
	session.TestRoot = filepath.Join(home, ".pi", "agent", "sessions")
	defer func() { session.TestRoot = old }()
	proj := "/home/goat/proj"
	path := seedPiSession(t, session.TestRoot, proj, piFixture)

	tl, err := (PISource{}).Read(context.Background(), Ref{ID: "pi-1", Path: path, Cwd: proj})
	if err != nil {
		t.Fatal(err)
	}
	want := "message/user thinking message/assistant tool_call tool_result compaction message/assistant"
	if got := kinds(tl); got != want {
		t.Fatalf("kinds:\n got %s\nwant %s", got, want)
	}
	if tl.Header.SourceID != "pi-1" || tl.Header.Provider != "zai" || tl.Header.Model != "glm-5.3-flash" || tl.Header.Title != "race fix" || tl.Header.FormatVersion != "3" {
		t.Fatalf("header = %+v", tl.Header)
	}
	if tl.Manifest.Dropped["pi.branch"] != 1 || tl.Manifest.Dropped["pi.thinking_level_change"] != 1 {
		t.Fatalf("manifest = %+v (the abandoned branch e7 is counted, not emitted)", tl.Manifest.Dropped)
	}
	if tl.Events[0].Timestamp.UnixMilli() != 1788645750288 {
		t.Fatalf("message timestamps come from message.timestamp (ms): %v", tl.Events[0].Timestamp)
	}
	// Version guard.
	v2 := seedPiSession(t, session.TestRoot, "/home/goat/other", strings.Replace(piFixture, `"version":3`, `"version":2`, 1))
	if _, err := (PISource{}).Read(context.Background(), Ref{Path: v2}); !errors.Is(err, ErrUnknownFormat) {
		t.Fatalf("version 2: %v", err)
	}
}

func TestHermesReadMapsRows(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("HERMES_HOME", "")
	path := seedHermesDB(t, filepath.Join(home, ".hermes"), `CREATE TABLE messages (
		id INTEGER PRIMARY KEY AUTOINCREMENT, session_id TEXT NOT NULL, role TEXT NOT NULL, content TEXT,
		tool_call_id TEXT, tool_calls TEXT, tool_name TEXT, timestamp REAL NOT NULL, reasoning TEXT,
		active INTEGER NOT NULL DEFAULT 1, compacted INTEGER NOT NULL DEFAULT 0);`)
	insertHermes(t, path, "20260906_202219_54a6a2", "cli", "Greeting", "/home/goat/proj", "", 1788736944.5, 4, 0)
	db, err := sql.Open("sqlite", "file:"+path)
	if err != nil {
		t.Fatal(err)
	}
	rows := [][]any{
		{"system", "You are Hermes", nil, nil, nil, 1788736944.6, 1, 0},
		{"user", "Read HELLO.txt", nil, nil, nil, 1788736945.0, 1, 0},
		{"assistant", "", nil, `[{"id":"call_1","call_id":"call_1","type":"function","function":{"name":"read_file","arguments":"{\"path\":\"HELLO.txt\"}"}}]`, nil, 1788736946.0, 1, 0},
		{"tool", `{"content":"the secret word"}`, "call_1", nil, "read_file", 1788736947.0, 1, 0},
		{"assistant", "The secret word is PLATYPUS.", nil, nil, nil, 1788736948.0, 1, 0},
		{"user", "old turn", nil, nil, nil, 1788736949.0, 0, 0},
		{"user", "summarized turn", nil, nil, nil, 1788736950.0, 1, 1},
	}
	for _, r := range rows {
		if _, err := db.Exec(`INSERT INTO messages (session_id, role, content, tool_call_id, tool_calls, tool_name, timestamp, active, compacted) VALUES (?,?,?,?,?,?,?,?,?)`, append([]any{"20260906_202219_54a6a2"}, r...)...); err != nil {
			t.Fatal(err)
		}
	}
	db.Close()

	tl, err := (HermesSource{}).Read(context.Background(), Ref{ID: "20260906_202219_54a6a2", Path: path, Cwd: "/home/goat/proj"})
	if err != nil {
		t.Fatal(err)
	}
	want := "context/system message/user tool_call tool_result message/assistant"
	if got := kinds(tl); got != want {
		t.Fatalf("kinds:\n got %s\nwant %s", got, want)
	}
	if c := tl.Events[2].Call; c.ID != "call_1" || c.Name != "read_file" || string(c.Input) != `{"path":"HELLO.txt"}` {
		t.Fatalf("call = %+v", c)
	}
	if r := tl.Events[3].Result; r.Name != "read_file" || r.CallID != "call_1" {
		t.Fatalf("result = %+v", r)
	}
	if tl.Header.Title != "Greeting" || tl.Header.Cwd != "/home/goat/proj" {
		t.Fatalf("header = %+v", tl.Header)
	}
	if tl.Manifest.Dropped["hermes.inactive"] != 1 || tl.Manifest.Dropped["hermes.compacted"] != 1 {
		t.Fatalf("manifest = %+v", tl.Manifest.Dropped)
	}
	if _, err := (HermesSource{}).Read(context.Background(), Ref{ID: "nope", Path: path}); !errors.Is(err, ErrNotUnderRoot) {
		t.Fatalf("unknown session: %v", err)
	}
}

func TestReadRefusesPathOutsideRoot(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	old := session.TestRoot
	session.TestRoot = filepath.Join(home, ".pi", "agent", "sessions")
	defer func() { session.TestRoot = old }()
	outside := filepath.Join(t.TempDir(), "x.jsonl")
	write(t, outside, `{"type":"session","version":3}`)
	for cli, r := range map[string]Reader{"claude-code": ClaudeCodeSource{}, "codex": CodexSource{}, "pi": PISource{}} {
		if _, err := r.Read(context.Background(), Ref{ID: "x", Path: outside}); !errors.Is(err, ErrNotUnderRoot) {
			t.Errorf("%s accepted a path outside its root: %v", cli, err)
		}
		if _, err := r.Read(context.Background(), Ref{ID: "x", Path: filepath.Join(home, ".claude", "projects", "..", "..", "x.jsonl")}); !errors.Is(err, ErrNotUnderRoot) {
			t.Errorf("%s accepted a traversal: %v", cli, err)
		}
	}
	if _, err := (GrokSource{}).Read(context.Background(), Ref{ID: "..", Cwd: "/"}); !errors.Is(err, ErrNotUnderRoot) {
		t.Errorf("grok traversal: %v", err)
	}
}

func TestScanSessionTailReadsTheRecentEnd(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "big.jsonl")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString("{\"id\":\"old\"}\n"); err != nil {
		t.Fatal(err)
	}
	if _, err := f.Seek(MaxReadBytes, 0); err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString("{\"id\":\"new\"}\n"); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	if err := scanSessionLines(path, false, func([]byte) {}); !errors.Is(err, ErrTooLarge) {
		t.Fatalf("full oversize: %v", err)
	}
	var ids []string
	if err := scanSessionLines(path, true, func(line []byte) {
		var raw struct {
			ID string `json:"id"`
		}
		if json.Unmarshal(line, &raw) == nil && raw.ID != "" {
			ids = append(ids, raw.ID)
		}
	}); err != nil {
		t.Fatal(err)
	}
	if len(ids) == 0 || ids[len(ids)-1] != "new" {
		t.Fatalf("tail ids = %v", ids)
	}
}

func TestReadRefusesOversize(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	path := filepath.Join(home, ".claude", "projects", "-p", "big.jsonl")
	write(t, path, "")
	if err := os.Truncate(path, MaxReadBytes+1); err != nil {
		t.Skip("sparse files unsupported:", err)
	}
	if _, err := (ClaudeCodeSource{}).Read(context.Background(), Ref{Path: path}); !errors.Is(err, ErrTooLarge) {
		t.Fatalf("oversize: %v", err)
	}
}

// sample is the timeline the writer tests round-trip: a note, a turn with
// text plus two calls (one object input, one raw string), their results
// (one error), and a closing assistant turn.
func sample() transcript.Timeline {
	return transcript.Timeline{
		Header: transcript.Header{SourceCLI: "claude-code", SourceName: "Claude Code", SourceID: "cc-1", Cwd: "/home/goat/proj", Title: "Race fix", Model: "claude-sonnet-5"},
		Events: []transcript.Event{
			{Kind: transcript.KindMessage, Role: "user", Text: "Handoff from Claude Code, session cc-1.", Group: 1},
			{Kind: transcript.KindMessage, Role: "user", Text: "fix the race", Group: 2},
			{Kind: transcript.KindThinking, Role: "assistant", Text: "hmm", Group: 3},
			{Kind: transcript.KindMessage, Role: "assistant", Text: "On it.", Model: "claude-sonnet-5", Group: 3},
			{Kind: transcript.KindToolCall, Role: "assistant", Call: &transcript.ToolCall{ID: "toolu_1", Name: "Bash", Input: json.RawMessage(`{"command":"go test ./..."}`)}, Model: "claude-sonnet-5", Group: 3},
			{Kind: transcript.KindToolCall, Role: "assistant", Call: &transcript.ToolCall{ID: "toolu_2", Name: "exec", Input: json.RawMessage(`"const r = 1"`)}, Model: "claude-sonnet-5", Group: 3},
			{Kind: transcript.KindToolResult, Result: &transcript.ToolResult{CallID: "toolu_1", Name: "Bash", Text: "ok"}, Group: 4},
			{Kind: transcript.KindToolResult, Result: &transcript.ToolResult{CallID: "toolu_2", Name: "exec", Text: "boom", IsError: true}, Group: 5},
			{Kind: transcript.KindContext, Role: "user", Text: "<system-reminder>x</system-reminder>", Group: 6},
			{Kind: transcript.KindMessage, Role: "assistant", Text: "Done.", Model: "claude-sonnet-5", Group: 7},
		},
	}
}

func TestClaudeWriteRoundTrip(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	now := time.Date(2026, 9, 6, 12, 0, 0, 0, time.UTC)
	got, err := (ClaudeCodeSource{}).Write(context.Background(), sample(), WriteRequest{Cwd: "/home/goat/proj", FormatVersion: "2.1.263", Now: now})
	if err != nil {
		t.Fatal(err)
	}
	wantDir := filepath.Join(home, ".claude", "projects", "-home-goat-proj")
	if filepath.Dir(got.Path) != wantDir || filepath.Base(got.Path) != got.ID+".jsonl" {
		t.Fatalf("path = %s", got.Path)
	}
	// No transcript on this machine to learn a model from, so none is
	// claimed: the source's model belongs to the handoff note, not to a
	// record that says what answers next.
	if !reflect.DeepEqual(got.ResumeArgs, []string{"--resume", got.ID}) || got.Name != "Race fix" || got.Messages != 4 || got.Model != "" {
		t.Fatalf("summary = %+v", got)
	}
	if body, _ := os.ReadFile(got.Path); strings.Contains(string(body), "claude-sonnet-5") {
		t.Fatal("the source's model must not be written as this session's model")
	}
	raw, _ := os.ReadFile(got.Path)
	lines := strings.Split(strings.TrimSpace(string(raw)), "\n")
	// note, user, assistant(text+2 tool_use), 2 tool_result lines, assistant
	if len(lines) != 6 {
		t.Fatalf("%d lines:\n%s", len(lines), raw)
	}
	var first, second, third, fourth map[string]any
	_ = json.Unmarshal([]byte(lines[0]), &first)
	_ = json.Unmarshal([]byte(lines[1]), &second)
	_ = json.Unmarshal([]byte(lines[2]), &third)
	_ = json.Unmarshal([]byte(lines[3]), &fourth)
	if first["parentUuid"] != nil || second["parentUuid"] != first["uuid"] || first["sessionId"] != got.ID || first["version"] != "2.1.263" || first["cwd"] != "/home/goat/proj" {
		t.Fatalf("chain/header fields wrong: %v %v", first, second)
	}
	content := third["message"].(map[string]any)["content"].([]any)
	if len(content) != 3 || content[1].(map[string]any)["type"] != "tool_use" || third["message"].(map[string]any)["stop_reason"] != "tool_use" {
		t.Fatalf("assistant line = %v", third)
	}
	if in := content[2].(map[string]any)["input"].(map[string]any); in["input"] != "const r = 1" {
		t.Fatalf("non-object input must be wrapped: %v", in)
	}
	res := fourth["message"].(map[string]any)["content"].([]any)[0].(map[string]any)
	if res["type"] != "tool_result" || res["tool_use_id"] != content[1].(map[string]any)["id"] {
		t.Fatalf("tool_result must follow its tool_use with the same id: %v", fourth)
	}
	if strings.Contains(string(raw), "thinking") || strings.Contains(string(raw), "system-reminder") {
		t.Fatal("thinking and context must not be written")
	}
	// The listing sees it with the same id.
	rows, _ := (ClaudeCodeSource{}).List("/home/goat/proj")
	if len(rows) != 1 || rows[0].ID != got.ID {
		t.Fatalf("listing = %+v", rows)
	}
}

func TestClaudeWriteProbesVersionAndRefusesUnknown(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	if _, err := (ClaudeCodeSource{}).Write(context.Background(), sample(), WriteRequest{Cwd: "/home/goat/proj"}); !errors.Is(err, ErrUnknownFormat) {
		t.Fatalf("no local transcript: %v", err)
	}
	if entries, _ := os.ReadDir(filepath.Join(home, ".claude")); len(entries) != 0 {
		t.Fatal("a refused write must leave nothing behind")
	}
	seedClaudeTranscript(t, home, "/home/goat/other")
	got, err := (ClaudeCodeSource{}).Write(context.Background(), sample(), WriteRequest{Cwd: "/home/goat/proj"})
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(got.Path)
	if !strings.Contains(string(raw), `"version":"2.1.263"`) {
		t.Fatal("version must come from the newest local transcript")
	}
	// The model comes from the same place: what this installation last ran.
	if got.Model != "claude-sonnet-5" || !strings.Contains(string(raw), `"model":"claude-sonnet-5"`) {
		t.Fatalf("model must come from the newest local transcript: %q", got.Model)
	}
}

func TestCodexWriteRoundTrip(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	now := time.Date(2026, 9, 6, 12, 0, 0, 0, time.UTC)
	got, err := (CodexSource{}).Write(context.Background(), sample(), WriteRequest{Cwd: "/home/goat/proj", FormatVersion: "0.153.4", Now: now})
	if err != nil {
		t.Fatal(err)
	}
	wantDir := filepath.Join(home, ".codex", "sessions", "2026", "09", "06")
	if filepath.Dir(got.Path) != wantDir || filepath.Base(got.Path) != "rollout-2026-09-06T12-00-00-"+got.ID+".jsonl" {
		t.Fatalf("path = %s", got.Path)
	}
	if !reflect.DeepEqual(got.ResumeArgs, []string{"resume", got.ID}) || got.Messages != 4 {
		t.Fatalf("summary = %+v", got)
	}
	raw, _ := os.ReadFile(got.Path)
	lines := strings.Split(strings.TrimSpace(string(raw)), "\n")
	var meta map[string]any
	_ = json.Unmarshal([]byte(lines[0]), &meta)
	payload := meta["payload"].(map[string]any)
	if meta["type"] != "session_meta" || payload["id"] != got.ID || payload["session_id"] != got.ID || payload["cli_version"] != "0.153.4" || payload["cwd"] != "/home/goat/proj" {
		t.Fatalf("session_meta = %v", meta)
	}
	joined := string(raw)
	if strings.Contains(joined, "claude-sonnet-5") || strings.Contains(joined, `"turn_context"`) {
		t.Fatal("with no local rollout to learn a model from, none is claimed")
	}
	if !strings.Contains(joined, `"type":"function_call"`) || !strings.Contains(joined, `"arguments":"{\"command\":\"go test ./...\"}"`) || !strings.Contains(joined, `"arguments":"const r = 1"`) {
		t.Fatalf("function_call arguments must be JSON strings:\n%s", joined)
	}
	if !strings.Contains(joined, `"type":"function_call_output"`) || !strings.Contains(joined, `"output":"boom"`) {
		t.Fatalf("function_call_output missing:\n%s", joined)
	}
	if strings.Contains(joined, "hmm") || strings.Contains(joined, "system-reminder") {
		t.Fatal("thinking and context must not be written")
	}
	rows, _ := (CodexSource{}).List("/home/goat/proj")
	if len(rows) != 1 || rows[0].ID != got.ID || !reflect.DeepEqual(rows[0].ResumeArgs, []string{"resume", got.ID}) {
		t.Fatalf("listing = %+v", rows)
	}
}

func TestCodexWriteRecordsThisInstallationsModel(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	proj := "/home/goat/proj"
	seedCodexRollout(t, home, proj) // a local rollout: cli_version 0.153.2, model gpt-5.6-sol

	got, err := (CodexSource{}).Write(context.Background(), sample(), WriteRequest{Cwd: proj})
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(got.Path)
	if !strings.Contains(string(raw), `"cli_version":"0.153.2"`) || !strings.Contains(string(raw), `"model":"gpt-5.6-sol"`) {
		t.Fatalf("version and model come from the newest local rollout:\n%s", raw)
	}
	if got.Model != "gpt-5.6-sol" {
		t.Fatalf("summary model = %q", got.Model)
	}
	// Handing that session on again keeps the model, which is what the
	// turn_context line is for.
	back, err := (CodexSource{}).Read(context.Background(), Ref{ID: got.ID, Path: got.Path, Cwd: proj})
	if err != nil {
		t.Fatal(err)
	}
	if back.Header.Model != "gpt-5.6-sol" {
		t.Fatalf("model lost on the way out: %+v", back.Header)
	}
}

func TestPiWriteRoundTripInDir(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	old := session.TestRoot
	session.TestRoot = filepath.Join(home, ".pi", "agent", "sessions")
	defer func() { session.TestRoot = old }()
	dir := session.AgentDir("agent-1")
	now := time.Date(2026, 9, 6, 12, 0, 0, 0, time.UTC)

	if _, err := (PISource{}).Write(context.Background(), sample(), WriteRequest{Cwd: "/home/goat/proj"}); !errors.Is(err, ErrNoDir) {
		t.Fatalf("no dir: %v", err)
	}
	got, err := (PISource{}).Write(context.Background(), sample(), WriteRequest{Cwd: "/home/goat/proj", Dir: dir, Now: now})
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Dir(got.Path) != dir || !strings.HasSuffix(got.Path, "_"+got.ID+".jsonl") || len(got.ResumeArgs) != 0 {
		t.Fatalf("path/args = %s %v", got.Path, got.ResumeArgs)
	}
	// pi's own summarizer sees folder and name; no model_change is written,
	// so the adopting agent keeps PiCode's provider/model defaults instead
	// of pi trying to restore a provider it may not have.
	sum, err := session.Summarize(got.Path)
	if err != nil || sum.Cwd != "/home/goat/proj" || sum.Name != "Race fix" || sum.Provider != "" || sum.Model != "" {
		t.Fatalf("pi's own reader: %+v %v", sum, err)
	}
	events, err := session.Transcript(got.Path)
	if err != nil {
		t.Fatal(err)
	}
	var tools int
	for _, e := range events {
		if e.Kind == "tool" {
			tools++
			if e.Status == "···" {
				t.Fatalf("tool call without result in pi's chat model: %+v", e)
			}
		}
	}
	if tools != 2 {
		t.Fatalf("pi chat sees %d tools, want 2: %+v", tools, events)
	}
	raw, _ := os.ReadFile(got.Path)
	if !strings.HasPrefix(string(raw), `{"cwd":"/home/goat/proj","id":"`+got.ID+`","timestamp":"2026-09-06T12:00:00.000Z","type":"session","version":3}`) {
		t.Fatalf("header line: %s", strings.SplitN(string(raw), "\n", 2)[0])
	}
	if strings.Contains(string(raw), `"hmm"`) {
		t.Fatal("thinking must not be written")
	}
}

func TestWriteTextToolsModeAndNeverOverwrites(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	old := session.TestRoot
	session.TestRoot = filepath.Join(home, ".pi", "agent", "sessions")
	defer func() { session.TestRoot = old }()
	writers := map[string]Writer{"claude-code": ClaudeCodeSource{}, "codex": CodexSource{}, "pi": PISource{}}
	readers := map[string]Reader{"claude-code": ClaudeCodeSource{}, "codex": CodexSource{}, "pi": PISource{}}
	for cli, w := range writers {
		req := WriteRequest{Cwd: "/home/goat/proj", FormatVersion: map[string]string{"claude-code": "2.1.263", "codex": "0.153.4", "pi": "3"}[cli], Dir: session.AgentDir("a"), Tools: ToolsText, SessionID: "fixed-" + cli}
		got, err := w.Write(context.Background(), sample(), req)
		if err != nil {
			t.Fatalf("%s: %v", cli, err)
		}
		back, err := readers[cli].Read(context.Background(), Ref{ID: got.ID, Path: got.Path, Cwd: got.Cwd})
		if err != nil {
			t.Fatalf("%s read back: %v", cli, err)
		}
		c := back.Counts()
		if c.ToolCalls != 0 || c.ToolResults != 0 || c.Messages != 8 {
			t.Fatalf("%s text mode counts = %+v", cli, c)
		}
		if !strings.Contains(back.Events[3].Text, "Tool call `Bash`") || !strings.Contains(back.Events[5].Text, "Result of `Bash`") || !strings.Contains(back.Events[6].Text, "(error)") {
			t.Fatalf("%s text rendering: %q %q %q", cli, back.Events[3].Text, back.Events[5].Text, back.Events[6].Text)
		}
		before, _ := os.ReadFile(got.Path)
		if _, err := w.Write(context.Background(), sample(), req); err == nil {
			t.Fatalf("%s overwrote an existing session", cli)
		}
		after, _ := os.ReadFile(got.Path)
		if string(before) != string(after) {
			t.Fatalf("%s changed an existing file on a refused write", cli)
		}
	}
}

func TestCrossFormatChainKeepsCounts(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	old := session.TestRoot
	session.TestRoot = filepath.Join(home, ".pi", "agent", "sessions")
	defer func() { session.TestRoot = old }()
	proj := "/home/goat/proj"
	src := seedClaudeTranscript(t, home, proj)
	tl, err := (ClaudeCodeSource{}).Read(context.Background(), Ref{ID: "cc-1", Path: src, Cwd: proj})
	if err != nil {
		t.Fatal(err)
	}
	want := tl.Prepare().Counts()
	// claude → codex → pi → claude; the portable counts survive each hop.
	cx, err := (CodexSource{}).Write(context.Background(), tl, WriteRequest{Cwd: proj, FormatVersion: "0.153.4"})
	if err != nil {
		t.Fatal(err)
	}
	tl2, _ := (CodexSource{}).Read(context.Background(), Ref{ID: cx.ID, Path: cx.Path, Cwd: proj})
	pi, err := (PISource{}).Write(context.Background(), tl2, WriteRequest{Cwd: proj, Dir: session.AgentDir("a")})
	if err != nil {
		t.Fatal(err)
	}
	tl3, _ := (PISource{}).Read(context.Background(), Ref{ID: pi.ID, Path: pi.Path, Cwd: proj})
	cc, err := (ClaudeCodeSource{}).Write(context.Background(), tl3, WriteRequest{Cwd: proj, FormatVersion: "2.1.263"})
	if err != nil {
		t.Fatal(err)
	}
	tl4, _ := (ClaudeCodeSource{}).Read(context.Background(), Ref{ID: cc.ID, Path: cc.Path, Cwd: proj})
	got := tl4.Prepare().Counts()
	// The synthesized result for toolu_2 (never answered at the source)
	// exists from the first hop on; the compaction became a user message
	// on the first hop and stays one.
	want.ToolResults = want.ToolCalls
	if got != want {
		t.Fatalf("counts after three hops = %+v, want %+v", got, want)
	}
}

// A session written from a source with no title of its own must not be
// named after the handoff note: a listing then shows 120 characters of
// "Handoff from Claude Code running…", which is how this was found in the
// Sessions tab.
func TestWrittenSessionIsNotNamedAfterTheHandoffNote(t *testing.T) {
	untitled := sample()
	untitled.Header.Title = ""
	untitled.Events = untitled.Events[:1] // the note alone
	untitled.Events = append(untitled.Events,
		transcript.Event{Kind: transcript.KindContext, Role: "user", Text: "<command-name>/doctor</command-name>", Group: 2},
		transcript.Event{Kind: transcript.KindMessage, Role: "assistant", Text: "Checked.", Group: 3})
	if got := title(untitled); got != "From Claude Code" {
		t.Fatalf("with nothing of the source's own to name it after: %q", got)
	}
	withTurn := untitled
	withTurn.Events = append(withTurn.Events, transcript.Event{Kind: transcript.KindMessage, Role: "user", Text: "fix the race", Group: 4})
	if got := title(withTurn); got != "fix the race" {
		t.Fatalf("the source's own first turn names it: %q", got)
	}
	titled := sample()
	if got := title(titled); got != "Race fix" {
		t.Fatalf("a recorded title wins: %q", got)
	}
}

func TestPromptArgs(t *testing.T) {
	cases := []struct {
		cli  string
		want []string
	}{
		{"claude-code", []string{"--session-id", "sid", "do it"}},
		{"codex", []string{"do it"}},
		{"grok", []string{"--session-id", "sid", "do it"}},
		{"pi", []string{"--session-id", "sid", "do it"}},
		{"opencode", []string{"--prompt", "do it"}},
		{"muse", []string{"do it"}},
		{"agy", []string{"--prompt-interactive", "do it"}},
	}
	for _, c := range cases {
		p, ok := PrompterFor(c.cli)
		if !ok {
			t.Fatalf("%s has no prompter", c.cli)
		}
		if got := p.PromptArgs("do it", "sid"); !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s: %v", c.cli, got)
		}
	}
	if _, ok := PrompterFor("hermes"); ok {
		t.Error("hermes cannot take an initial prompt interactively")
	}
}

// seedOpenCodeSession writes one session with its messages and parts, in
// the shape a real store holds (OpenCode 1.18.29): a tool part carries the
// call and its result together.
func seedOpenCodeSession(t *testing.T, path, sid, dir string) {
	t.Helper()
	db, err := sql.Open("sqlite", "file:"+path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(`INSERT INTO session (id, project_id, directory, title, version, time_created, time_updated, model) VALUES (?,?,?,?,?,?,?,?)`,
		sid, "global", dir, "Race fix", "1.18.29", 1786547371123, 1786547376483, `{"id":"big-pickle","providerID":"opencode"}`); err != nil {
		t.Fatal(err)
	}
	msgs := []struct {
		id, data string
		parts    []string
	}{
		{"msg_1", `{"role":"user","time":{"created":1786547371144},"model":{"providerID":"opencode","modelID":"big-pickle"}}`,
			[]string{`{"type":"text","text":"fix the race"}`}},
		{"msg_2", `{"role":"assistant","modelID":"big-pickle","providerID":"opencode","time":{"created":1786547371158}}`,
			[]string{
				`{"type":"step-start"}`,
				`{"type":"reasoning","text":"let me look"}`,
				`{"type":"text","text":"On it."}`,
				`{"type":"tool","tool":"bash","callID":"call_1","state":{"status":"completed","input":{"command":"go test ./..."},"output":"ok","title":"go test"}}`,
				`{"type":"tool","tool":"read","callID":"call_2","state":{"status":"error","input":{"filePath":"/x"},"error":"boom"}}`,
				`{"type":"step-finish"}`,
			}},
		{"msg_3", `{"role":"assistant","modelID":"big-pickle","providerID":"opencode","time":{"created":1786547375000}}`,
			[]string{`{"type":"text","text":"Done."}`, `{"type":"patch","hash":"abc"}`}},
	}
	for mi, m := range msgs {
		if _, err := db.Exec(`INSERT INTO message (id, session_id, time_created, data) VALUES (?,?,?,?)`, m.id, sid, 1786547371000+mi, m.data); err != nil {
			t.Fatal(err)
		}
		for pi, p := range m.parts {
			if _, err := db.Exec(`INSERT INTO part (id, message_id, session_id, time_created, data) VALUES (?,?,?,?,?)`,
				m.id+"-p"+strconv.Itoa(pi), m.id, sid, 1786547371000+mi*100+pi, p); err != nil {
				t.Fatal(err)
			}
		}
	}
}

func TestOpenCodeReadMapsParts(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_DATA_HOME", "")
	path := seedOpenCodeDB(t, filepath.Join(home, ".local", "share", "opencode"))
	seedOpenCodeSession(t, path, "ses_1", "/home/goat/proj")

	tl, err := (OpenCodeSource{}).Read(context.Background(), Ref{ID: "ses_1", Cwd: "/home/goat/proj"})
	if err != nil {
		t.Fatal(err)
	}
	want := "message/user thinking message/assistant tool_call tool_result tool_call tool_result message/assistant"
	if got := kinds(tl); got != want {
		t.Fatalf("kinds:\n got %s\nwant %s", got, want)
	}
	if h := tl.Header; h.Title != "Race fix" || h.Cwd != "/home/goat/proj" || h.FormatVersion != "1.18.29" || h.Model != "opencode/big-pickle" {
		t.Fatalf("header = %+v", h)
	}
	if c := tl.Events[3].Call; c.ID != "call_1" || c.Name != "bash" || string(c.Input) != `{"command":"go test ./..."}` {
		t.Fatalf("call = %+v", c)
	}
	if r := tl.Events[4].Result; r.CallID != "call_1" || r.Text != "ok" || r.IsError {
		t.Fatalf("result = %+v", r)
	}
	if r := tl.Events[6].Result; !r.IsError || r.Text != "boom" {
		t.Fatalf("error result must carry the error text: %+v", r)
	}
	d := tl.Manifest.Dropped
	if d["opencode.step-start"] != 1 || d["opencode.step-finish"] != 1 || d["opencode.patch"] != 1 {
		t.Fatalf("manifest = %+v", d)
	}
	if _, err := (OpenCodeSource{}).Read(context.Background(), Ref{ID: "ses_nope"}); !errors.Is(err, ErrNotUnderRoot) {
		t.Fatalf("unknown session: %v", err)
	}
}

// applyOpenCodeImport is what `opencode import <file>` does to the store,
// so the writer test can prove the envelope it produces is one the CLI's
// own importer would land: ids kept, session filed under the invoking
// folder, messages and parts inserted in order.
func applyOpenCodeImport(t *testing.T, path, file, cwd string) {
	t.Helper()
	raw, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	var env struct {
		Info     map[string]any `json:"info"`
		Messages []struct {
			Info  map[string]any   `json:"info"`
			Parts []map[string]any `json:"parts"`
		} `json:"messages"`
	}
	if err := json.Unmarshal(raw, &env); err != nil {
		t.Fatalf("envelope is not JSON the importer could read: %v", err)
	}
	str := func(m map[string]any, k string) string {
		v, _ := m[k].(string)
		return v
	}
	sid := str(env.Info, "id")
	if !strings.HasPrefix(sid, "ses_") || str(env.Info, "title") == "" {
		t.Fatalf("session info = %v", env.Info)
	}
	db, err := sql.Open("sqlite", "file:"+path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	model, _ := json.Marshal(env.Info["model"])
	if _, err := db.Exec(`INSERT INTO session (id, project_id, directory, title, version, time_created, time_updated, model) VALUES (?,?,?,?,?,?,?,?)`,
		sid, "global", cwd, str(env.Info, "title"), str(env.Info, "version"), 1, 2, string(model)); err != nil {
		t.Fatal(err)
	}
	for mi, m := range env.Messages {
		mid := str(m.Info, "id")
		if !strings.HasPrefix(mid, "msg_") || str(m.Info, "sessionID") != sid {
			t.Fatalf("message info = %v", m.Info)
		}
		body, _ := json.Marshal(m.Info)
		if _, err := db.Exec(`INSERT INTO message (id, session_id, time_created, data) VALUES (?,?,?,?)`, mid, sid, mi, string(body)); err != nil {
			t.Fatal(err)
		}
		for pi, p := range m.Parts {
			pid := str(p, "id")
			if !strings.HasPrefix(pid, "prt_") || str(p, "messageID") != mid {
				t.Fatalf("part = %v", p)
			}
			if p["type"] == "tool" {
				// The real importer refuses a completed tool state without
				// these: "Missing key at [state][metadata]".
				state, _ := p["state"].(map[string]any)
				if state == nil || state["metadata"] == nil || state["status"] == nil || state["input"] == nil {
					t.Fatalf("tool part state = %v", p["state"])
				}
			}
			pb, _ := json.Marshal(p)
			if _, err := db.Exec(`INSERT INTO part (id, message_id, session_id, time_created, data) VALUES (?,?,?,?,?)`, pid, mid, sid, mi*100+pi, string(pb)); err != nil {
				t.Fatal(err)
			}
		}
	}
}

func TestOpenCodeWritePublishesThroughItsOwnImport(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_DATA_HOME", "")
	path := seedOpenCodeDB(t, filepath.Join(home, ".local", "share", "opencode"))
	seedOpenCodeSession(t, path, "ses_seed", "/home/goat/other") // a version and a model to probe

	if _, err := (OpenCodeSource{}).Write(context.Background(), sample(), WriteRequest{Cwd: "/home/goat/proj"}); !errors.Is(err, ErrNoRunner) {
		t.Fatalf("no runner: %v", err)
	}

	var ranArgs []string
	req := WriteRequest{Cwd: "/home/goat/proj", Now: time.Date(2026, 9, 7, 12, 0, 0, 0, time.UTC)}
	req.Run = func(ctx context.Context, args ...string) ([]byte, error) {
		ranArgs = args
		if len(args) != 2 || args[0] != "import" {
			t.Fatalf("import args = %v", args)
		}
		applyOpenCodeImport(t, path, args[1], req.Cwd)
		return []byte("Imported session"), nil
	}
	got, err := (OpenCodeSource{}).Write(context.Background(), sample(), req)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(got.ID, "ses_") || !reflect.DeepEqual(got.ResumeArgs, []string{"--session", got.ID}) || got.Name != "Race fix" || got.Messages != 4 {
		t.Fatalf("summary = %+v", got)
	}
	// The model is this installation's, learned from a message OpenCode
	// itself wrote, so a written session can never feed its own model back.
	if got.Model != "opencode/big-pickle" {
		t.Fatalf("model = %q", got.Model)
	}
	if _, err := os.Stat(ranArgs[1]); !os.IsNotExist(err) {
		t.Fatalf("the handoff file must not outlive the import: %v", err)
	}
	// The store now holds it, with the tool call and its result in one part.
	back, err := (OpenCodeSource{}).Read(context.Background(), Ref{ID: got.ID, Cwd: got.Cwd})
	if err != nil {
		t.Fatal(err)
	}
	if c := back.Counts(); c.Messages != 4 || c.ToolCalls != 2 || c.ToolResults != 2 {
		t.Fatalf("counts after import = %+v", c)
	}
	if back.Events[0].Text != sample().Events[0].Text {
		t.Fatalf("first turn = %q", back.Events[0].Text)
	}
	if r := back.Events[6].Result; r == nil || !r.IsError || r.Text != "boom" {
		t.Fatalf("the error result must survive the import: %+v", back.Events[6])
	}
	rows, _ := (OpenCodeSource{}).List("/home/goat/proj")
	if len(rows) != 1 || rows[0].ID != got.ID {
		t.Fatalf("listing = %+v", rows)
	}
}

func TestGrokWriteRoundTripMinimalFiles(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("GROK_HOME", "")
	proj := "/home/goat/proj"

	// No local session and no installed version: the chat format is unknown.
	if _, err := (GrokSource{}).Write(context.Background(), sample(), WriteRequest{Cwd: proj}); !errors.Is(err, ErrUnknownFormat) {
		t.Fatalf("unknown format: %v", err)
	}
	seedGrokSession(t, home, proj, "019ff88b-7d95-7241-be51-acb541c88ef2")

	now := time.Date(2026, 9, 7, 12, 0, 0, 0, time.UTC)
	got, err := (GrokSource{}).Write(context.Background(), sample(), WriteRequest{Cwd: proj, Now: now})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got.ResumeArgs, []string{"--resume", got.ID}) || got.Name != "Race fix" || got.Messages != 4 {
		t.Fatalf("summary = %+v", got)
	}
	// Only the two files the spike proved `grok --resume` needs, and the
	// folder's shared prompt history is never touched.
	entries, err := os.ReadDir(got.Path)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, e := range entries {
		names = append(names, e.Name())
	}
	sort.Strings(names)
	if !reflect.DeepEqual(names, []string{"chat_history.jsonl", "summary.json"}) {
		t.Fatalf("session dir holds %v", names)
	}
	ph := filepath.Join(filepath.Dir(got.Path), "prompt_history.jsonl")
	before, err := os.ReadFile(ph)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(before), got.ID) {
		t.Fatal("the shared prompt history must not be appended to")
	}
	raw, _ := os.ReadFile(filepath.Join(got.Path, "chat_history.jsonl"))
	body := string(raw)
	if !strings.Contains(body, "<user_query>") || !strings.Contains(body, `"tool_calls"`) || !strings.Contains(body, `"type":"tool_result"`) {
		t.Fatalf("chat history:\n%s", body)
	}
	if strings.Contains(body, "hmm") || strings.Contains(body, "system-reminder") {
		t.Fatal("thinking and context must not be written")
	}
	var sum map[string]any
	sb, _ := os.ReadFile(filepath.Join(got.Path, "summary.json"))
	if json.Unmarshal(sb, &sum) != nil {
		t.Fatalf("summary.json is not JSON: %s", sb)
	}
	if sum["chat_format_version"].(float64) != 1 || sum["session_summary"] != "Race fix" {
		t.Fatalf("summary = %v", sum)
	}
	if info := sum["info"].(map[string]any); info["id"] != got.ID || info["cwd"] != proj {
		t.Fatalf("summary info = %v", info)
	}
	// PiCode's own listing and reader agree with what was written.
	rows, _ := (GrokSource{}).List(proj)
	found := false
	for _, r := range rows {
		if r.ID == got.ID {
			found = true
		}
	}
	if !found {
		t.Fatalf("listing lost the written session: %+v", rows)
	}
	back, err := (GrokSource{}).Read(context.Background(), Ref{ID: got.ID, Cwd: proj})
	if err != nil {
		t.Fatal(err)
	}
	if c := back.Counts(); c.Messages != 4 || c.ToolCalls != 2 || c.ToolResults != 2 {
		t.Fatalf("counts = %+v", c)
	}
	if back.Events[0].Text != sample().Events[0].Text {
		t.Fatalf("first turn = %q", back.Events[0].Text)
	}
}

func TestHermesWriteImportsThroughItsOwnCommand(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("HERMES_HOME", "")
	proj := "/home/goat/proj"
	statePath := seedHermesDB(t, filepath.Join(home, ".hermes"), `CREATE TABLE messages (
		id INTEGER PRIMARY KEY AUTOINCREMENT, session_id TEXT NOT NULL, role TEXT NOT NULL, content TEXT,
		tool_call_id TEXT, tool_calls TEXT, tool_name TEXT, timestamp REAL NOT NULL, reasoning TEXT,
		active INTEGER NOT NULL DEFAULT 1, compacted INTEGER NOT NULL DEFAULT 0);`)

	if _, err := (HermesSource{}).Write(context.Background(), sample(), WriteRequest{Cwd: proj}); !errors.Is(err, ErrNoRunner) {
		t.Fatalf("no runner: %v", err)
	}
	req := WriteRequest{Cwd: proj, Now: time.Date(2026, 9, 7, 12, 0, 0, 0, time.UTC)}
	req.Run = func(ctx context.Context, args ...string) ([]byte, error) { return nil, nil }
	// No Claude Code transcript on this machine: the file to hand over
	// cannot be given a version, so nothing is attempted.
	if _, err := (HermesSource{}).Write(context.Background(), sample(), req); !errors.Is(err, ErrUnknownFormat) {
		t.Fatalf("unknown claude format: %v", err)
	}
	seedClaudeTranscript(t, home, "/home/goat/other")

	// A runner that behaves like `hermes sessions import --from claude`:
	// reads the file, flattens tool calls into the assistant's text,
	// assigns its own id and reports it.
	assigned := "20260907_145708_de908b"
	var handed string
	req.Run = func(ctx context.Context, args ...string) ([]byte, error) {
		if len(args) != 5 || args[0] != "sessions" || args[1] != "import" || args[2] != "--from" || args[3] != "claude" {
			t.Fatalf("import args = %v", args)
		}
		handed = args[4]
		raw, err := os.ReadFile(handed)
		if err != nil {
			t.Fatal(err)
		}
		db, err := sql.Open("sqlite", "file:"+statePath)
		if err != nil {
			t.Fatal(err)
		}
		defer db.Close()
		if _, err := db.Exec(`INSERT INTO sessions (id, source, title, cwd, started_at, message_count) VALUES (?,?,?,?,?,?)`,
			assigned, "claude-code", "Imported from Claude Code: fix the race", proj, 1788736944.5, 0); err != nil {
			t.Fatal(err)
		}
		n := 0
		for _, line := range strings.Split(strings.TrimSpace(string(raw)), "\n") {
			var rec struct {
				Message struct {
					Role    string `json:"role"`
					Content []struct {
						Text string `json:"text"`
					} `json:"content"`
				} `json:"message"`
			}
			if json.Unmarshal([]byte(line), &rec) != nil || rec.Message.Role == "" {
				continue
			}
			text := ""
			for _, c := range rec.Message.Content {
				text += c.Text
			}
			if _, err := db.Exec(`INSERT INTO messages (session_id, role, content, timestamp) VALUES (?,?,?,?)`,
				assigned, rec.Message.Role, text, 1788736945.0+float64(n)); err != nil {
				t.Fatal(err)
			}
			n++
		}
		if _, err := db.Exec(`UPDATE sessions SET message_count = ? WHERE id = ?`, n, assigned); err != nil {
			t.Fatal(err)
		}
		return []byte("✓ Imported Claude Code session as " + assigned + "\n  Continue it with:  hermes --resume " + assigned + "\n"), nil
	}
	got, err := (HermesSource{}).Write(context.Background(), sample(), req)
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != assigned || !reflect.DeepEqual(got.ResumeArgs, []string{"--resume", assigned}) || got.Messages == 0 {
		t.Fatalf("summary = %+v", got)
	}
	if _, err := os.Stat(handed); !os.IsNotExist(err) {
		t.Fatalf("the handed file must not outlive the import: %v", err)
	}
	// An imported session is a coding session: the picker shows it.
	rows, _ := (HermesSource{}).List(proj)
	if len(rows) != 1 || rows[0].ID != assigned {
		t.Fatalf("listing = %+v", rows)
	}
	// A silent import is not a handoff.
	req.Run = func(ctx context.Context, args ...string) ([]byte, error) { return []byte("nothing to say"), nil }
	if _, err := (HermesSource{}).Write(context.Background(), sample(), req); err == nil || !strings.Contains(err.Error(), "did not report") {
		t.Fatalf("unreported id: %v", err)
	}
}

func TestHermesImportedIDFromOutput(t *testing.T) {
	out := "✓ Imported Claude Code session as 20260907_145708_de908b\n  Continue it with:  hermes --resume 20260907_145708_de908b\n"
	if got := hermesImportedID(out); got != "20260907_145708_de908b" {
		t.Fatalf("id = %q", got)
	}
	if hermesImportedID("no id here") != "" {
		t.Fatal("must not invent an id")
	}
}
