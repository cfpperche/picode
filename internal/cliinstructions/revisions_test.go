package cliinstructions

import (
	"os"
	"path/filepath"
	"testing"
)

// One row per way AgentRevisions answers.
func TestAgentRevisions(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	repo := t.TempDir()
	write(t, repo, map[string]string{"AGENTS.md": "# Build\nmake\n", "gone.md": "x"})
	rec := t.TempDir()
	gone := filepath.Join(repo, "gone.md")
	write(t, rec, map[string]string{
		"rollout.jsonl": `{"text":"# AGENTS.md instructions for ` + repo + `\n"}` + "\n",
		"claude.jsonl":  `{"type":"user","message":{"content":"Contents of ` + gone + ` (project instructions)"}}` + "\n",
	})
	if err := os.Remove(gone); err != nil {
		t.Fatal(err)
	}

	// Observed: the CLI's record names the file; the hash is its content now.
	obs := AgentRevisions(repo, repo, "codex", "", filepath.Join(rec, "rollout.jsonl"))
	if obs == nil || obs.Source != "observed" || len(obs.Files) != 1 || obs.Files[0].Path != "AGENTS.md" || len(obs.Files[0].SHA) != 12 || obs.Files[0].Bytes != 13 {
		t.Fatalf("observed = %+v", obs)
	}

	// Declared: a CLI with no record (Pi) gets the rules' "reads" files, and
	// the same content gives the same hash either way.
	dec := AgentRevisions(repo, "", "pi", "", "")
	if dec == nil || dec.Source != "declared" || len(dec.Files) != 1 || dec.Files[0].SHA != obs.Files[0].SHA {
		t.Fatalf("declared = %+v, observed %+v", dec, obs)
	}

	// A record PiCode cannot read falls back to the rules.
	if r := AgentRevisions(repo, repo, "codex", "", filepath.Join(rec, "missing.jsonl")); r == nil || r.Source != "declared" {
		t.Fatalf("missing record = %+v", r)
	}

	// A file the record names but that is gone keeps its path, with no hash.
	if r := AgentRevisions(repo, repo, "claude-code", "", filepath.Join(rec, "claude.jsonl")); r == nil || r.Source != "observed" || len(r.Files) != 1 || r.Files[0].SHA != "" {
		t.Fatalf("gone file = %+v", r)
	}

	// An edit changes the hash.
	write(t, repo, map[string]string{"AGENTS.md": "# Build\nmake all\n"})
	if r := AgentRevisions(repo, "", "pi", "", ""); r.Files[0].SHA == obs.Files[0].SHA {
		t.Fatal("an edit kept the same hash")
	}

	// A workspace folder that is gone answers nil.
	if r := AgentRevisions(filepath.Join(repo, "nope"), "", "pi", "", ""); r != nil {
		t.Fatalf("gone root = %+v", r)
	}
}
