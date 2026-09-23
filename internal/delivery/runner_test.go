package delivery

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cfpperche/picode/internal/store"
)

// The runner touches real branches, so its tests use a real repository: the
// declared commands are the shell's, and the move is Git's own fast-forward.
type runFixture struct {
	st    *store.Store
	repo  string // the key the queue is filed under: the common Git directory
	cwd   string
	del   store.Delivery
	entry store.QueueEntry
	head  string
	run   func(args ...string) string
}

func newRunFixture(t *testing.T, target string) runFixture {
	t.Helper()
	cwd := t.TempDir()
	git := func(args ...string) string {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = cwd
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=test", "GIT_AUTHOR_EMAIL=test@example.com",
			"GIT_COMMITTER_NAME=test", "GIT_COMMITTER_EMAIL=test@example.com")
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v (%s)", args, err, out)
		}
		return strings.TrimSpace(string(out))
	}
	git("init", "-q", "-b", "main")
	write(t, filepath.Join(cwd, "a.txt"), "one\n")
	git("add", "a.txt")
	git("commit", "-qm", "one")
	if target != "main" {
		git("branch", target)
	}
	git("checkout", "-q", "-b", "feature")
	write(t, filepath.Join(cwd, "b.txt"), "two\n")
	git("add", "b.txt")
	git("commit", "-qm", "two")
	git("checkout", "-q", "main")
	head := git("rev-parse", "feature")

	st, err := store.Open(filepath.Join(t.TempDir(), "delivery.db"))
	if err != nil {
		t.Fatal(err)
	}
	repo := filepath.Join(cwd, ".git")
	d, err := st.ApplyDelivery(repo, "agent", store.DeliveryMutation{Action: "register", RequestID: "create",
		Title: "Fix", Branch: "feature", Revision: head, Target: target})
	if err != nil {
		t.Fatal(err)
	}
	entry, err := st.ApplyQueueMutation(repo, "agent", store.QueueMutation{Action: "enqueue", RequestID: "enqueue",
		DeliveryID: d.ID, Revision: head, Target: target})
	if err != nil {
		t.Fatal(err)
	}
	entry, err = st.ApplyQueueMutation(repo, store.OwnerActor, store.QueueMutation{Action: "authorize",
		RequestID: "authorize", ID: entry.ID, ExpectedVersion: entry.Version})
	if err != nil {
		t.Fatal(err)
	}
	return runFixture{st: st, repo: repo, cwd: cwd, del: d, entry: entry, head: head, run: git}
}

func write(t *testing.T, path, body string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func declared(checks ...string) store.IntegrationSettings {
	return store.IntegrationSettings{Scope: "ws1", Mode: store.ModeLocal, FFOnly: true, Checks: checks, FromScope: "ws1", Version: 1}
}

func TestRunEntryIntegratesAfterTheDeclaredCommands(t *testing.T) {
	f := newRunFixture(t, "main")
	out, err := RunEntry(context.Background(), RunConfig{Store: f.st, Repo: f.repo, Cwd: f.cwd,
		Settings: declared("true", "true"), Delivery: f.del, Entry: f.entry})
	if err != nil {
		t.Fatal(err)
	}
	if out.State != store.QueueDone || !strings.Contains(out.Note, "after 2 declared command(s)") {
		t.Fatalf("outcome = %+v", out)
	}
	if got := f.run("rev-parse", "main"); got != f.head {
		t.Fatalf("main = %s, want %s", got, f.head)
	}
	if !strings.Contains(out.Note, "integrated "+short(f.head)) {
		t.Fatalf("note = %q", out.Note)
	}
	if out.StartedAt == "" || out.Version != f.entry.Version+2 {
		t.Fatalf("startedAt/version = %+v", out)
	}
}

// A branch no worktree holds moves by compare-and-swap, so the checkout the
// repository uses is never disturbed.
func TestRunEntryMovesABranchNoWorktreeHolds(t *testing.T) {
	f := newRunFixture(t, "release")
	out, err := RunEntry(context.Background(), RunConfig{Store: f.st, Repo: f.repo, Cwd: f.cwd,
		Settings: declared(), Delivery: f.del, Entry: f.entry})
	if err != nil || out.State != store.QueueDone {
		t.Fatalf("outcome = %+v (%v)", out, err)
	}
	if got := f.run("rev-parse", "release"); got != f.head {
		t.Fatalf("release = %s, want %s", got, f.head)
	}
	if got := f.run("rev-parse", "main"); got == f.head {
		t.Fatal("main moved although the entry named release")
	}
	if out := f.run("status", "--porcelain"); out != "" {
		t.Fatalf("the working tree changed: %s", out)
	}
}

// Every way an authorization can go stale is a named blocker, and the target
// never moves on one of them.
func TestRunEntryBlockers(t *testing.T) {
	rows := []struct {
		name   string
		change func(t *testing.T, f runFixture) RunConfig
		want   string
	}{
		{"no mode declared", func(t *testing.T, f runFixture) RunConfig {
			return RunConfig{Store: f.st, Repo: f.repo, Cwd: f.cwd, Delivery: f.del, Entry: f.entry,
				Settings: store.IntegrationSettings{FFOnly: true, FromScope: "default"}}
		}, "declares no integration mode"},
		{"the project integrates through its provider", func(t *testing.T, f runFixture) RunConfig {
			s := declared()
			s.Mode = store.ModeProvider
			return RunConfig{Store: f.st, Repo: f.repo, Cwd: f.cwd, Settings: s, Delivery: f.del, Entry: f.entry}
		}, "integrates through its own provider"},
		{"a policy the runner does not implement", func(t *testing.T, f runFixture) RunConfig {
			s := declared()
			s.FFOnly = false
			return RunConfig{Store: f.st, Repo: f.repo, Cwd: f.cwd, Settings: s, Delivery: f.del, Entry: f.entry}
		}, "does not ask for fast-forward-only integration"},
		{"the branch moved", func(t *testing.T, f runFixture) RunConfig {
			write(t, filepath.Join(f.cwd, "c.txt"), "three\n")
			f.run("checkout", "-q", "feature")
			f.run("add", "c.txt")
			f.run("commit", "-qm", "three")
			f.run("checkout", "-q", "main")
			return RunConfig{Store: f.st, Repo: f.repo, Cwd: f.cwd, Settings: declared(), Delivery: f.del, Entry: f.entry}
		}, "no longer points at the reviewed revision"},
		{"the target moved", func(t *testing.T, f runFixture) RunConfig {
			write(t, filepath.Join(f.cwd, "d.txt"), "four\n")
			f.run("add", "d.txt")
			f.run("commit", "-qm", "four")
			return RunConfig{Store: f.st, Repo: f.repo, Cwd: f.cwd, Settings: declared(), Delivery: f.del, Entry: f.entry}
		}, "a fast-forward is no longer possible"},
		{"the evidence says the checks failed", func(t *testing.T, f runFixture) RunConfig {
			original := observe
			observe = func(ctx context.Context, cwd, repo, target string, decl []Declaration) Snapshot {
				return Snapshot{Complete: true, Changes: []Change{{ID: f.del.ID, Validation: "failed"}}}
			}
			t.Cleanup(func() { observe = original })
			return RunConfig{Store: f.st, Repo: f.repo, Cwd: f.cwd, Settings: declared(), Delivery: f.del, Entry: f.entry}
		}, "the recorded evidence says the checks failed"},
	}
	for _, row := range rows {
		t.Run(row.name, func(t *testing.T) {
			f := newRunFixture(t, "main")
			cfg := row.change(t, f)
			// The row sets its own stage first: what the run must leave alone is
			// the target as the blocker finds it.
			before := f.run("rev-parse", "main")
			out, err := RunEntry(context.Background(), cfg)
			if err != nil {
				t.Fatal(err)
			}
			if out.State != store.QueueFailed || !strings.HasPrefix(out.Note, "not run: ") || !strings.Contains(out.Note, row.want) {
				t.Fatalf("outcome = %+v", out)
			}
			if got := f.run("rev-parse", "main"); got != before {
				t.Fatalf("main moved to %s on a blocker", got)
			}
			if out.StartedAt != "" {
				t.Fatalf("a blocked entry was started: %+v", out)
			}
		})
	}
}

func TestRunEntryFailsOnADeclaredCommand(t *testing.T) {
	f := newRunFixture(t, "main")
	before := f.run("rev-parse", "main")
	out, err := RunEntry(context.Background(), RunConfig{Store: f.st, Repo: f.repo, Cwd: f.cwd,
		Settings: declared("true", "echo boom >&2; false"), Delivery: f.del, Entry: f.entry})
	if err != nil {
		t.Fatal(err)
	}
	if out.State != store.QueueFailed || !strings.Contains(out.Note, "check 2 of 2 failed") || !strings.Contains(out.Note, "boom") {
		t.Fatalf("outcome = %+v", out)
	}
	if got := f.run("rev-parse", "main"); got != before {
		t.Fatalf("main moved although a declared check failed")
	}
	if out.StartedAt == "" {
		t.Fatalf("a failed run was never started: %+v", out)
	}
}

// The operation is claimed before it runs, so a second entry in the same
// repository cannot start while the first is in flight.
func TestRunEntrySerializesTheRepository(t *testing.T) {
	f := newRunFixture(t, "main")
	if _, err := f.st.ApplyQueueMutation(f.repo, store.OwnerActor, store.QueueMutation{Action: "start",
		RequestID: "manual", ID: f.entry.ID, ExpectedVersion: f.entry.Version}); err != nil {
		t.Fatal(err)
	}
	second, err := f.st.ApplyDelivery(f.repo, "agent", store.DeliveryMutation{Action: "register", RequestID: "create-2",
		Title: "Other", Branch: "feature", Revision: f.head, Target: "main"})
	if err != nil {
		t.Fatal(err)
	}
	other, err := f.st.ApplyQueueMutation(f.repo, "agent", store.QueueMutation{Action: "enqueue", RequestID: "enqueue-2",
		DeliveryID: second.ID, Revision: f.head, Target: "main"})
	if err != nil {
		t.Fatal(err)
	}
	other, err = f.st.ApplyQueueMutation(f.repo, store.OwnerActor, store.QueueMutation{Action: "authorize",
		RequestID: "authorize-2", ID: other.ID, ExpectedVersion: other.Version})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := RunEntry(context.Background(), RunConfig{Store: f.st, Repo: f.repo, Cwd: f.cwd,
		Settings: declared(), Delivery: second, Entry: other}); err == nil ||
		!strings.Contains(err.Error(), "another entry is running") {
		t.Fatalf("err = %v", err)
	}
	if got, err := f.st.GetQueueEntry(f.repo, other.ID); err != nil || got.State != store.QueueAuthorized {
		t.Fatalf("the waiting entry moved: %+v (%v)", got, err)
	}
}
