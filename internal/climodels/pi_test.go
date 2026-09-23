package climodels

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

const piTable = `provider          model                                               context  max-out  thinking  images
anthropic         claude-haiku-4-5                                    200K     64K      yes       yes
openrouter        qwen/qwen3 coder free                               1.0M     32.8K    no        no
No models match your filter today, try again later please ok
`

func TestParsePiTable(t *testing.T) {
	rows := ParsePiTable(piTable)
	if len(rows) != 2 {
		t.Fatalf("rows = %+v", rows)
	}
	a, b := rows[0], rows[1]
	if a.Selector != "anthropic/claude-haiku-4-5" || a.Context != 200_000 || a.MaxOut != 64_000 || !a.Reasoning || len(a.Input) != 1 || a.Kind != "chat" {
		t.Errorf("row 0 = %+v", a)
	}
	// A model id with spaces keeps them; the vendor's labels survive as written.
	if b.ID != "qwen/qwen3 coder free" || b.ContextLabel != "1.0M" || b.MaxOutLabel != "32.8K" || b.Context != 1_000_000 || b.MaxOut != 32_800 || b.Reasoning || b.Input != nil {
		t.Errorf("row 1 = %+v", b)
	}
	if n := len(ParsePiTable("")); n != 0 {
		t.Errorf("empty table = %d rows", n)
	}
}

// Every measured input moves the fingerprint; a file Pi does not read does not.
func TestPiFingerprintFollowsItsInputs(t *testing.T) {
	h := t.TempDir()
	old := home
	home = func() string { return h }
	t.Cleanup(func() { home = old })
	t.Setenv("PI_CODING_AGENT_DIR", "")
	agent := filepath.Join(h, ".pi", "agent")
	if err := os.MkdirAll(filepath.Join(agent, "npm"), 0o755); err != nil {
		t.Fatal(err)
	}
	cmd := filepath.Join(t.TempDir(), "no-such-pi")
	prev := fingerprint("pi", cmd, "")
	for _, f := range []string{"auth.json", "models.json", "models-store.json", "settings.json", filepath.Join("npm", "package-lock.json")} {
		if err := os.WriteFile(filepath.Join(agent, f), []byte("x"), 0o600); err != nil {
			t.Fatal(err)
		}
		next := fingerprint("pi", cmd, "")
		if next == prev {
			t.Errorf("writing %s did not move the fingerprint", f)
		}
		prev = next
	}
	if err := os.WriteFile(filepath.Join(agent, "trust.json"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if fingerprint("pi", cmd, "") != prev {
		t.Error("trust.json is not an input")
	}
}

// A kept answer is served until an input moves or Forget drops it.
func TestReadKeepsAndForgets(t *testing.T) {
	h := t.TempDir()
	old := home
	home = func() string { return h }
	t.Cleanup(func() { home = old; Forget("pi") })
	t.Setenv("PI_CODING_AGENT_DIR", "")
	agent := filepath.Join(h, ".pi", "agent")
	if err := os.MkdirAll(agent, 0o755); err != nil {
		t.Fatal(err)
	}
	// A stand-in pi: a script that counts its runs and prints the table.
	dir := t.TempDir()
	count := filepath.Join(dir, "runs")
	script := filepath.Join(dir, "pi")
	body := "#!/bin/sh\necho x >> " + count + "\ncat <<'T'\n" + piTable + "T\n"
	if err := os.WriteFile(script, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	runs := func() int { b, _ := os.ReadFile(count); return len(b) / 2 }
	for i := 0; i < 2; i++ {
		rep, err := ReadCommand(t.Context(), "pi", script, "", false)
		if err != nil || len(rep.Models) != 2 {
			t.Fatalf("read %d = %+v, %v", i, rep, err)
		}
	}
	if runs() != 1 {
		t.Fatalf("pi ran %d times for two reads, want 1", runs())
	}
	// A sign-in rewrites auth.json: the next read asks again.
	time.Sleep(10 * time.Millisecond)
	if err := os.WriteFile(filepath.Join(agent, "auth.json"), []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, _ = ReadCommand(t.Context(), "pi", script, "", false)
	if runs() != 2 {
		t.Fatalf("pi ran %d times after auth.json moved, want 2", runs())
	}
	Forget("pi")
	_, _ = ReadCommand(t.Context(), "pi", script, "", false)
	if runs() != 3 {
		t.Fatalf("pi ran %d times after Forget, want 3", runs())
	}
	if _, err := ReadCommand(t.Context(), "pi", filepath.Join(dir, "missing"), "", false); err == nil {
		t.Error("a missing pi is an error, never an empty catalog")
	}
}
