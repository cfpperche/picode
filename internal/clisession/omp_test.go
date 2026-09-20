package clisession

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// seedOmp writes one session file in omp's real layout: the cwd-encoded
// bucket under ~/.omp/agent/sessions and `<timestamp>_<id>.jsonl` inside.
func seedOmp(t *testing.T, home, bucket, name, body string) {
	t.Helper()
	p := filepath.Join(home, ".omp", "agent", "sessions", bucket, name)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func ompSession(id, cwd, title string, messages int) string {
	return `{"type":"title","v":1,"title":"` + title + `","updatedAt":"2026-09-17T15:44:46.791Z"}` + "\n" +
		`{"type":"session","version":3,"id":"` + id + `","timestamp":"2026-09-17T15:44:46.791Z","cwd":"` + cwd + `"}` + "\n" +
		`{"type":"model_change","id":"d0fa4cf3","parentId":null,"timestamp":"2026-09-17T15:44:46.859Z","model":"google/gemini-3.6-flash","resolvedModelIsFallback":false}` + "\n" +
		`{"type":"thinking_level_change","id":"t1","timestamp":"2026-09-17T15:44:46.860Z","thinkingLevel":"off"}` + "\n" +
		`{"type":"message","id":"m1","timestamp":"2026-09-17T15:44:46.900Z","message":{"role":"user","attribution":"user","content":[{"type":"text","text":"Reply with the single word OK"}],"timestamp":1758121486900}}` + "\n" +
		`{"type":"message","id":"m2","timestamp":"2026-09-17T15:44:52.310Z","message":{"role":"assistant","api":"google-genai","provider":"google","model":"gemini-3.6-flash","content":[{"type":"text","text":"OK"}],"stopReason":"stop","timestamp":1758121492310,"usage":{"input":20434,"output":75}}}` + "\n"
}

func TestOmpList(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	seedOmp(t, home, "-tmp-omp-probe",
		"2026-09-17T15-44-46-791Z_01a0b00a-b4c7-72e5-a844-b58d24ac3a07.jsonl",
		ompSession("01a0b00a-b4c7-72e5-a844-b58d24ac3a07", "/tmp/omp-probe", "", 2))
	// A titled session in another folder, latest record newest.
	seedOmp(t, home, "-home-goat-proj",
		"2026-09-17T16-00-00-000Z_01a0b00a-0000-0000-0000-000000000001.jsonl",
		ompSession("01a0b00a-0000-0000-0000-000000000001", "/home/goat/proj", "Race fix in the cart", 2))
	// A header-only file nobody prompted is not a session to resume.
	seedOmp(t, home, "-tmp-omp-probe",
		"2026-09-17T15-00-00-000Z_01a0b00a-0000-0000-0000-000000000002.jsonl",
		`{"type":"session","version":3,"id":"01a0b00a-0000-0000-0000-000000000002","timestamp":"2026-09-17T15:00:00.000Z","cwd":"/tmp/omp-probe"}`+"\n")
	// Garbage lines are skipped, not fatal.
	seedOmp(t, home, "-tmp-omp-probe", "broken.jsonl", "not json at all\n{\"half\n")

	got, err := OmpSource{}.List("")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d summaries, want 2: %+v", len(got), got)
	}
	s := got[0]
	if s.ID != "01a0b00a-0000-0000-0000-000000000001" || s.Cwd != "/home/goat/proj" || s.CLI != "omp" {
		t.Errorf("newest first + identity fields wrong: %+v", s)
	}
	if s.Name != "Race fix in the cart" {
		t.Errorf("name from title record = %q", s.Name)
	}
	if s.Model != "google/gemini-3.6-flash" { // model_change's provider-qualified id wins
		t.Errorf("model = %q", s.Model)
	}
	if !reflect.DeepEqual(s.ResumeArgs, []string{"--resume", "01a0b00a-0000-0000-0000-000000000001"}) {
		t.Errorf("resumeArgs = %v", s.ResumeArgs)
	}
	if s.Messages != 2 || s.Preview != "Reply with the single word OK" {
		t.Errorf("messages/preview = %d/%q", s.Messages, s.Preview)
	}
	if s.UpdatedAt == "" || s.CreatedAt == "" || s.Size == 0 {
		t.Errorf("timestamps/size missing: %+v", s)
	}

	// The untitled session names itself after its first user text.
	untitled := got[1]
	if untitled.ID != "01a0b00a-b4c7-72e5-a844-b58d24ac3a07" || untitled.Name != "Reply with the single word OK" {
		t.Errorf("untitled name = %q (id %s)", untitled.Name, untitled.ID)
	}

	scoped, err := OmpSource{}.List("/home/goat/proj")
	if err != nil || len(scoped) != 1 || scoped[0].ID != "01a0b00a-0000-0000-0000-000000000001" {
		t.Errorf("scoped list = %v, %v; want the proj session", scoped, err)
	}
	scoped, err = OmpSource{}.List("/somewhere-else")
	if err != nil || len(scoped) != 0 {
		t.Errorf("scoped list = %v, %v; want empty", scoped, err)
	}
}

func TestOmpListHonorsAgentDirOverride(t *testing.T) {
	t.Setenv("PI_CODING_AGENT_DIR", "/custom/omp")
	if got := ompSessionsRoot(); got != filepath.Join("/custom/omp", "sessions") {
		t.Errorf("root = %q, want the override", got)
	}
	// The default root still reads the user's home.
	t.Setenv("PI_CODING_AGENT_DIR", "")
	home := t.TempDir()
	t.Setenv("HOME", home)
	if got := ompSessionsRoot(); got != filepath.Join(home, ".omp", "agent", "sessions") {
		t.Errorf("root = %q, want the default home", got)
	}
}

func TestOmpListMissingRootIsEmpty(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	got, err := OmpSource{}.List("")
	if err != nil || len(got) != 0 {
		t.Errorf("missing root = %v, %v; want empty", got, err)
	}
}
