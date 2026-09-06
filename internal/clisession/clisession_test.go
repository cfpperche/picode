package clisession

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/cfpperche/picode/internal/session"
)

// seedClaude writes one Claude Code transcript in the real format (lines
// abbreviated from a 2026-09 installation: queue-operation noise, camelCase
// sessionId on old lines, snake_case session_id on new ones, sidechain
// subagent turns that are not part of the conversation).
func seedClaude(t *testing.T, home, dirName, id, cwd string, lines ...string) {
	t.Helper()
	dir := filepath.Join(home, ".claude", "projects", dirName)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(dir, id+".jsonl")
	body := strings.Join(lines, "\n")
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	back := time.Now().Add(-2 * time.Hour)
	if err := os.Chtimes(p, back, back); err != nil {
		t.Fatal(err)
	}
}

func TestClaudeCodeList(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	seedClaude(t, home, "-home-goat-proj", "aaaa-bbbb", "/home/goat/proj",
		`{"type":"queue-operation","operation":"enqueue","timestamp":"2026-09-01T10:00:00.000Z","sessionId":"aaaa-bbbb"}`,
		`{"type":"user","message":{"role":"user","content":[{"text":"fix the race","type":"text"}]},"uuid":"u1","timestamp":"2026-09-01T10:00:01.000Z","cwd":"/home/goat/proj","sessionId":"aaaa-bbbb","version":"2.1.241"}`,
		`{"parentUuid":"u1","isSidechain":true,"type":"assistant","message":{"role":"assistant","model":"claude-sonnet-4-5","content":[{"type":"text","text":"subagent noise"}]},"timestamp":"2026-09-01T10:00:02.000Z","session_id":"aaaa-bbbb","cwd":"/home/goat/proj"}`,
		`{"parentUuid":"u1","isSidechain":false,"type":"assistant","message":{"role":"assistant","model":"claude-sonnet-4-5","content":[{"type":"text","text":"on it"}]},"timestamp":"2026-09-01T10:00:03.000Z","session_id":"aaaa-bbbb","cwd":"/home/goat/proj"}`,
		`{"type":"summary","summary":"Race fix in the cart reducer","leafUuid":"u1"}`,
	)
	// A transcript with no cwd anywhere is unusable (cannot be opened) and
	// must be dropped, not listed with an empty folder.
	seedClaude(t, home, "-home-goat-proj", "cccc-dddd", "",
		`{"type":"user","message":{"role":"user","content":"orphan line"},"timestamp":"2026-09-01T09:00:00.000Z"}`,
	)

	got, err := ClaudeCodeSource{}.List("")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("got %d summaries, want 1: %+v", len(got), got)
	}
	s := got[0]
	if s.ID != "aaaa-bbbb" || s.Cwd != "/home/goat/proj" || s.CLI != "claude-code" {
		t.Errorf("identity fields wrong: %+v", s)
	}
	if s.Preview != "fix the race" {
		t.Errorf("preview = %q", s.Preview)
	}
	if s.Name != "Race fix in the cart reducer" {
		t.Errorf("name from summary line = %q", s.Name)
	}
	if s.Model != "claude-sonnet-4-5" {
		t.Errorf("model = %q", s.Model)
	}
	if s.Messages != 2 { // sidechain assistant turn is not a conversation turn
		t.Errorf("messages = %d, want 2", s.Messages)
	}
	if !reflect.DeepEqual(s.ResumeArgs, []string{"--resume", "aaaa-bbbb"}) {
		t.Errorf("resumeArgs = %v", s.ResumeArgs)
	}
	if s.UpdatedAt == "" || s.CreatedAt == "" {
		t.Errorf("timestamps missing: %+v", s)
	}

	scoped, err := ClaudeCodeSource{}.List("/somewhere-else")
	if err != nil || len(scoped) != 0 {
		t.Errorf("scoped list = %v, %v; want empty", scoped, err)
	}
	scoped, err = ClaudeCodeSource{}.List("/home/goat/proj")
	if err != nil || len(scoped) != 1 {
		t.Errorf("scoped list = %v, %v; want 1", scoped, err)
	}
}

// seedCodex writes one rollout in the real format: session_meta first,
// turn_context with the model, response_item messages where developer
// prompts are not human turns.
func seedCodex(t *testing.T, home, rel, name, body string) {
	t.Helper()
	p := filepath.Join(home, ".codex", "sessions", rel, name)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestCodexList(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	seedCodex(t, home, "2026/09/01",
		"rollout-2026-09-01T08-00-00-019b0805-74e2-78a0-9c51-1206f2da1003.jsonl",
		strings.Join([]string{
			`{"timestamp":"2026-09-01T11:00:00.000Z","type":"session_meta","payload":{"id":"019b0805-74e2-78a0-9c51-1206f2da1003","timestamp":"2026-09-01T11:00:00.000Z","cwd":"/home/goat/workout","originator":"codex_vscode","cli_version":"0.65.0"}}`,
			`{"timestamp":"2026-09-01T11:00:01.000Z","type":"response_item","payload":{"type":"message","role":"developer","content":[{"type":"input_text","text":"memory guidance from prior runs"}]}}`,
			`{"timestamp":"2026-09-01T11:00:02.000Z","type":"response_item","payload":{"type":"message","role":"user","content":[{"type":"input_text","text":"plan the migration"}]}}`,
			`{"timestamp":"2026-09-01T11:00:03.000Z","type":"turn_context","payload":{"turn_id":"t1","cwd":"/home/goat/workout","model":"gpt-6-astra"}}`,
			`{"timestamp":"2026-09-01T11:00:04.000Z","type":"response_item","payload":{"type":"message","role":"assistant","content":[{"type":"output_text","text":"three steps"}]}}`,
		}, "\n"),
	)

	got, err := CodexSource{}.List("")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("got %d, want 1: %+v", len(got), got)
	}
	s := got[0]
	if s.ID != "019b0805-74e2-78a0-9c51-1206f2da1003" || s.Cwd != "/home/goat/workout" {
		t.Errorf("identity wrong: %+v", s)
	}
	if s.Preview != "plan the migration" {
		t.Errorf("preview must be the user turn, got %q", s.Preview)
	}
	if s.Messages != 2 { // developer guidance is not a conversation turn
		t.Errorf("messages = %d, want 2", s.Messages)
	}
	if s.Model != "gpt-6-astra" {
		t.Errorf("model = %q", s.Model)
	}
	if !reflect.DeepEqual(s.ResumeArgs, []string{"resume", s.ID}) {
		t.Errorf("resumeArgs = %v (verified: codex resume <id> is positional)", s.ResumeArgs)
	}
	scoped, _ := CodexSource{}.List("/nope")
	if len(scoped) != 0 {
		t.Errorf("scoped = %v, want empty", scoped)
	}
}

func TestGrokList(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	dir := filepath.Join(home, ".grok", "sessions", "%2Fhome%2Fgoat%2Fproj")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	body := strings.Join([]string{
		`{"timestamp":"2026-08-13T00:35:27.579026410Z","session_id":"019ff88b-7d95-7241-be51-acb541c88ef2","prompt":"first prompt","is_bash":false}`,
		`{"timestamp":"2026-08-13T00:37:44.000000000Z","session_id":"019ff88d-94df-7013-bed0-ef7b81874b92","prompt":"other session prompt","is_bash":false}`,
		`{"timestamp":"2026-08-13T00:40:00.000000000Z","session_id":"019ff88b-7d95-7241-be51-acb541c88ef2","prompt":"follow-up","is_bash":false}`,
	}, "\n")
	if err := os.WriteFile(filepath.Join(dir, "prompt_history.jsonl"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := GrokSource{}.List("")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d sessions, want 2: %+v", len(got), got)
	}
	// Newest activity first: the other-session prompt (00:37) is older than
	// the first session's follow-up (00:40).
	if got[0].ID != "019ff88b-7d95-7241-be51-acb541c88ef2" {
		t.Errorf("newest first violated: %+v", got[0])
	}
	first := got[0]
	if first.Cwd != "/home/goat/proj" || first.Messages != 2 || first.Preview != "first prompt" {
		t.Errorf("fold wrong: %+v", first)
	}
	if !reflect.DeepEqual(first.ResumeArgs, []string{"--resume", first.ID}) {
		t.Errorf("resumeArgs = %v", first.ResumeArgs)
	}
	scoped, _ := GrokSource{}.List("/home/goat/proj")
	if len(scoped) != 2 {
		t.Errorf("scoped by decoded dir = %d, want 2", len(scoped))
	}
	scoped, _ = GrokSource{}.List("/elsewhere")
	if len(scoped) != 0 {
		t.Errorf("other cwd = %d, want 0 (dir name filters without parsing)", len(scoped))
	}
}

func TestPIListAdaptsProviderAndModel(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	old := session.TestRoot
	session.TestRoot = filepath.Join(home, ".pi", "agent", "sessions")
	defer func() { session.TestRoot = old }()

	dir := filepath.Join(session.TestRoot, session.DirName("/home/goat/proj"))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	body := strings.Join([]string{
		`{"type":"session","id":"pi-sess-1","timestamp":"2026-09-05T10:00:00Z","cwd":"/home/goat/proj"}`,
		`{"type":"session_info","name":"Named session"}`,
		`{"type":"model_change","provider":"anthropic","modelId":"claude-sonnet-4-5"}`,
		`{"type":"message","message":{"role":"user","content":[{"type":"text","text":"hello pi"}]}}`,
		`{"type":"message","message":{"role":"assistant","provider":"anthropic","modelId":"claude-sonnet-4-5","usage":{"cost":{"total":0.05}},"content":[{"type":"text","text":"hi"}]}}`,
	}, "\n")
	if err := os.WriteFile(filepath.Join(dir, "pi-sess-1.jsonl"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := PISource{}.List("")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("got %d, want 1", len(got))
	}
	s := got[0]
	if s.Model != "anthropic/claude-sonnet-4-5" || s.Cost != 0.05 || s.Name != "Named session" {
		t.Errorf("pi adaptation wrong: %+v", s)
	}
	if len(s.ResumeArgs) != 0 {
		t.Errorf("pi resumes through its own flow, got %v", s.ResumeArgs)
	}
}

func TestSourcesRegistry(t *testing.T) {
	all := Sources()
	for _, cli := range []string{"pi", "claude-code", "codex", "grok"} {
		if _, ok := all[cli]; !ok {
			t.Errorf("source %q missing from registry", cli)
		}
	}
	if _, ok := Get("nope"); ok {
		t.Error("unknown CLI resolved")
	}
	// A machine where none of the non-pi CLIs ever ran still lists cleanly.
	home := t.TempDir()
	t.Setenv("HOME", home)
	for _, cli := range []string{"claude-code", "codex", "grok"} {
		src, ok := Get(cli)
		if !ok {
			t.Fatalf("source %q missing", cli)
		}
		rows, err := src.List("")
		if err != nil || len(rows) != 0 {
			t.Errorf("%s on an empty machine = %v, %v; want empty, no error", cli, rows, err)
		}
	}
}
