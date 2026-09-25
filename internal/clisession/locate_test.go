package clisession

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestLocate(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	proj := filepath.Join(home, "proj")
	claudeDir := filepath.Join(home, ".claude", "projects", encodeClaudeTestDir(proj))
	if err := os.MkdirAll(claudeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	line := `{"type":"user","message":{"role":"user","content":"hi"},"timestamp":"2026-09-01T10:00:00Z","cwd":"` + proj + `","sessionId":"cc-9"}` + "\n"
	if err := os.WriteFile(filepath.Join(claudeDir, "cc-9.jsonl"), []byte(line), 0o644); err != nil {
		t.Fatal(err)
	}
	piFile := filepath.Join(home, "s.jsonl")
	if err := os.WriteFile(piFile, []byte(`{"type":"session","id":"p1","cwd":"/w"}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	l := NewLocator()
	for _, tc := range []struct {
		name, cli, id, path, cwd string
		found                    bool
	}{
		{"claude in its folder", "claude-code", "cc-9", "", proj, true},
		{"claude by recorded file", "claude-code", "cc-9", filepath.Join(claudeDir, "cc-9.jsonl"), proj, true},
		{"claude stale path falls back", "claude-code", "cc-9", filepath.Join(claudeDir, "gone.jsonl"), proj, true},
		{"claude, folder unknown: machine-wide", "claude-code", "cc-9", "", "", true},
		{"claude, wrong folder: falls back to machine-wide", "claude-code", "cc-9", "", "/elsewhere", true},
		{"claude, unknown id", "claude-code", "nope", "", proj, false},
		{"no id is never a match", "claude-code", "", filepath.Join(claudeDir, "cc-9.jsonl"), proj, false},
		{"pi by file", "pi", "", piFile, "", true},
		{"pi file gone", "pi", "", filepath.Join(home, "gone.jsonl"), "", false},
		{"unknown CLI", "nope", "x", "", "", false},
	} {
		got, err := l.Locate(tc.cli, tc.id, tc.path, tc.cwd)
		if err != nil {
			t.Fatalf("%s: %v", tc.name, err)
		}
		if (got != nil) != tc.found {
			t.Fatalf("%s: found = %v", tc.name, got)
		}
		if got != nil && tc.cli == "claude-code" && (len(got.ResumeArgs) != 2 || got.ResumeArgs[1] != "cc-9") {
			t.Fatalf("%s: resume args = %v", tc.name, got.ResumeArgs)
		}
	}
}

func TestLocateRecordedCodexFileAndChange(t *testing.T) {
	root := t.TempDir()
	CodexTestRoot = root
	t.Cleanup(func() { CodexTestRoot = "" })
	path := filepath.Join(root, "rollout-test.jsonl")
	write := func(prompt string) {
		t.Helper()
		body := `{"type":"session_meta","payload":{"id":"codex-1","cwd":"/project"}}` + "\n" +
			`{"type":"response_item","payload":{"type":"message","role":"user","content":[{"type":"input_text","text":"` + prompt + `"}]}}` + "\n"
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("first")
	locate := func() *Summary {
		t.Helper()
		got, err := NewLocator().Locate("codex", "codex-1", path, "/project")
		if err != nil || got == nil {
			t.Fatalf("locate: %+v, %v", got, err)
		}
		return got
	}
	if got := locate(); got.Preview != "first" {
		t.Fatalf("first preview = %q", got.Preview)
	}
	write("updated prompt")
	if got := locate(); got.Preview != "updated prompt" {
		t.Fatalf("changed file preview = %q", got.Preview)
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if got, err := NewLocator().Locate("codex", "codex-1", path, "/project"); err != nil || got != nil {
		t.Fatalf("removed file = %+v, %v", got, err)
	}
}

func encodeClaudeTestDir(cwd string) string {
	return strings.NewReplacer("/", "-", ".", "-").Replace(cwd)
}

func BenchmarkLocateCodexHistory(b *testing.B) {
	root := b.TempDir()
	CodexTestRoot = root
	b.Cleanup(func() { CodexTestRoot = "" })
	var target string
	for i := 0; i < 100; i++ {
		id := "session-" + strconv.Itoa(i)
		path := filepath.Join(root, "rollout-"+id+".jsonl")
		body := `{"type":"session_meta","payload":{"id":"` + id + `","cwd":"/project"}}` + "\n"
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			b.Fatal(err)
		}
		if i == 99 {
			target = path
		}
	}
	for _, tc := range []struct {
		name, path string
	}{
		{"recorded_path", target},
		{"listing_fallback", ""},
	} {
		b.Run(tc.name, func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				got, err := NewLocator().Locate("codex", "session-99", tc.path, "/project")
				if err != nil || got == nil {
					b.Fatalf("locate = %+v, %v", got, err)
				}
			}
		})
	}
}

// A moved or unrecorded file is found by the id in its name, and a deleted
// one is gone without listing every session on the machine.
func TestLocateFileBackedByName(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	proj := filepath.Join(home, "proj")
	day := filepath.Join(home, ".codex", "sessions", "2026", "09", "20")
	if err := os.MkdirAll(day, 0o755); err != nil {
		t.Fatal(err)
	}
	id := "01a0bebb-3ca2-7833-823e-ee4d052e63f7"
	rollout := filepath.Join(day, "rollout-2026-09-20T09-12-16-"+id+".jsonl")
	body := `{"timestamp":"2026-09-20T09:12:16Z","type":"session_meta","payload":{"id":"` + id + `","cwd":"` + proj + `","timestamp":"2026-09-20T09:12:16Z"}}` + "\n" +
		`{"timestamp":"2026-09-20T09:12:17Z","type":"response_item","payload":{"type":"message","role":"user","content":[{"type":"input_text","text":"fix it"}]}}` + "\n"
	if err := os.WriteFile(rollout, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	l := NewLocator()
	got, err := l.Locate("codex", id, "", "")
	if err != nil || got == nil || got.Path != rollout {
		t.Fatalf("codex by name = %+v, %v", got, err)
	}
	got, _ = l.Locate("codex", id, filepath.Join(home, "old", "place.jsonl"), proj)
	if got == nil || got.Path != rollout {
		t.Fatalf("moved codex file = %+v", got)
	}
	if err := os.Remove(rollout); err != nil {
		t.Fatal(err)
	}
	if got, _ := NewLocator().Locate("codex", id, rollout, proj); got != nil {
		t.Fatalf("deleted transcript still found: %+v", got)
	}
	if got, _ := NewLocator().Locate("claude-code", "../x", "", ""); got != nil {
		t.Fatalf("a path-like id matched: %+v", got)
	}
}

func TestLocatePrivateOmpAgentSession(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	dataDir := filepath.Join(home, "picode")
	id := "01a0d389-b7d9-719c-8f85-49235635221d"
	path := filepath.Join(dataDir, "omp-sessions", "agent-1", "2026-09-24T13-10-13-209Z_"+id+".jsonl")
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(ompSession(id, "/old/worktree", "Pilot", 2)), 0o600); err != nil {
		t.Fatal(err)
	}
	if got, _ := NewLocator().Locate("omp", id, path, "/old/worktree"); got != nil {
		t.Fatalf("regular locator found private Omp session: %+v", got)
	}
	for _, recordedPath := range []string{path, ""} {
		got, err := NewAgentHistoryLocator(dataDir).Locate("omp", id, recordedPath, "/old/worktree")
		if err != nil || got == nil || got.Path != path {
			t.Fatalf("recorded path %q: got %+v, %v", recordedPath, got, err)
		}
	}
	if got, _ := NewAgentHistoryLocator(filepath.Join(home, "other")).Locate("omp", id, path, "/old/worktree"); got != nil {
		t.Fatalf("unrelated data directory found session: %+v", got)
	}
}

// A hook may pin a conversation with no path, or with a file inside Grok's
// session folder: StoreFor names where the conversation is kept.
func TestStoreFor(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GROK_HOME", home)
	oldH, oldO := HermesTestDB, OpenCodeTestDB
	HermesTestDB, OpenCodeTestDB = "/h/state.db", "/o/opencode.db"
	t.Cleanup(func() { HermesTestDB, OpenCodeTestDB = oldH, oldO })
	sess := filepath.Join(home, "sessions", "%2Frepo", "g1")
	if err := os.MkdirAll(sess, 0o755); err != nil {
		t.Fatal(err)
	}
	chat := filepath.Join(sess, "chat_history.jsonl")
	if err := os.WriteFile(chat, []byte("{}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct{ cli, id, path, cwd, want string }{
		{"hermes", "h1", "", "/repo", "/h/state.db"},
		{"hermes", "h1", "/other/state.db", "/repo", "/other/state.db"},
		{"opencode", "o1", "", "/repo", "/o/opencode.db"},
		{"grok", "g1", "", "/repo", sess},
		{"grok", "g1", chat, "/repo", sess},
		{"grok", "g1", sess, "", sess},
		{"claude-code", "c1", "", "/repo", ""},
		{"codex", "x1", "/x.jsonl", "/repo", "/x.jsonl"},
	} {
		if got := StoreFor(c.cli, c.id, c.path, c.cwd); got != c.want {
			t.Errorf("%s %q %q = %q, want %q", c.cli, c.id, c.path, got, c.want)
		}
	}
}
