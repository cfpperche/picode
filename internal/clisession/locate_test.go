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
