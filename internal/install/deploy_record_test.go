//go:build unix

package install

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ADR-0085: every deploy appends who/what/where to the deploy log, one
// JSON object per line, so session-loss post-mortems start from facts.
func TestAppendDeployRecord(t *testing.T) {
	data := t.TempDir()
	t.Setenv("PICODE_TERM_ID", "opencode-8da0d0")
	t.Setenv("PICODE_INSECURE", "")
	os.Chdir(data) // cwd rides the record; chdir is process-wide but this test owns it
	t.Cleanup(func() { os.Chdir("/") })

	AppendDeployRecord(data)
	AppendDeployRecord(data) // two deploys = two lines

	raw, err := os.ReadFile(filepath.Join(data, "var", "deploy-log.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimRight(string(raw), "\n"), "\n")
	if len(lines) != 2 {
		t.Fatalf("deploy log lines = %d, want 2", len(lines))
	}
	var rec deployRecord
	if err := json.Unmarshal([]byte(lines[0]), &rec); err != nil {
		t.Fatal(err)
	}
	if rec.TermID != "opencode-8da0d0" {
		t.Errorf("termId = %q", rec.TermID)
	}
	if rec.Version == "" || rec.At == "" || rec.Cwd == "" {
		t.Errorf("incomplete record: %+v", rec)
	}

	// Empty data dir is a silent no-op, never a panic.
	AppendDeployRecord("")
}

type recordAt interface{ GetAt() string }

func decodeLine(t *testing.T, line string) deployRecord {
	t.Helper()
	var rec deployRecord
	if err := json.Unmarshal([]byte(line), &rec); err != nil {
		t.Fatal(err)
	}
	return rec
}
