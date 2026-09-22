package delivery

import (
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/cfpperche/picode/internal/store"
)

// The runner is the only part of PiCode that changes a branch on its own. It
// obeys the declaration a project wrote down (ADR-0182), never a script one
// repository happens to own: the checks are the declared ones, and the move is
// a fast-forward into the target the entry names.
const (
	commandTimeout = 30 * time.Minute
	// The store keeps a note to one line of at most 300 bytes.
	noteLimit = 300
)

// RunConfig carries what a run needs: the entry, the delivery it names, and the
// rules the project declared. Nothing here is inferred by the runner.
type RunConfig struct {
	Store    *store.Store
	Repo     string
	Cwd      string
	Settings store.IntegrationSettings
	Delivery store.Delivery
	Entry    store.QueueEntry
}

// RunEntry performs one authorized integration. The declared commands must pass
// in order, then the target moves to the reviewed revision by fast-forward only.
// Every outcome lands on the entry: `done` with what it did, or `failed` with
// the reason — a blocker, a failed command or a failed move. An entry that
// cannot be recorded returns an error, and the caller leaves it as it is.
func RunEntry(ctx context.Context, cfg RunConfig) (store.QueueEntry, error) {
	e := cfg.Entry
	if e.State != store.QueueAuthorized {
		return e, fmt.Errorf("entry %s is %s, not authorized", e.ID, e.State)
	}
	if reason := blockReason(ctx, cfg); reason != "" {
		return cfg.record(e, "fail", "not run: "+reason)
	}
	started, err := cfg.Store.ApplyQueueMutation(cfg.Repo, store.OwnerActor, store.QueueMutation{
		Action: "start", RequestID: "run-start-" + e.ID + "-" + strconv.Itoa(e.Version),
		ID: e.ID, ExpectedVersion: e.Version})
	if err != nil {
		return e, fmt.Errorf("cannot start %s: %w", e.ID, err)
	}
	for i, command := range cfg.Settings.Checks {
		out, err := runShell(ctx, cfg.Cwd, command)
		if err != nil {
			return cfg.record(started, "fail", fmt.Sprintf("check %d of %d failed: %s — %s",
				i+1, len(cfg.Settings.Checks), command, explain(out, err)))
		}
	}
	if err := integrate(ctx, cfg.Cwd, e.Target, e.Revision); err != nil {
		return cfg.record(started, "fail", "the fast-forward failed: "+err.Error())
	}
	note := "integrated " + short(e.Revision) + " into " + e.Target
	if n := len(cfg.Settings.Checks); n > 0 {
		note = fmt.Sprintf("%s after %d declared command(s)", note, n)
	}
	return cfg.record(started, "finish", note)
}

func (cfg RunConfig) record(e store.QueueEntry, action, note string) (store.QueueEntry, error) {
	out, err := cfg.Store.ApplyQueueMutation(cfg.Repo, store.OwnerActor, store.QueueMutation{
		Action: action, RequestID: "run-" + action + "-" + e.ID + "-" + strconv.Itoa(e.Version),
		ID: e.ID, ExpectedVersion: e.Version, Note: oneLine(note)})
	if err != nil {
		return e, err
	}
	return out, nil
}

// blockReason re-reads what an entry was authorized against. Any of these
// changing invalidates the authorization and names the reason — never silently
// broadened, never re-authorized (ADR-0182).
func blockReason(ctx context.Context, cfg RunConfig) string {
	switch {
	case cfg.Settings.FromScope == "default":
		return "the project declares no integration rules"
	case !cfg.Settings.FFOnly:
		return "the declaration does not ask for fast-forward-only integration"
	}
	head, err := git(ctx, cfg.Cwd, "rev-parse", "--verify", "refs/heads/"+cfg.Delivery.Branch)
	if err != nil || head != cfg.Entry.Revision {
		return "the branch no longer points at the reviewed revision"
	}
	target, err := git(ctx, cfg.Cwd, "rev-parse", "--verify", "refs/heads/"+cfg.Entry.Target)
	if err != nil {
		return "the target branch is gone"
	}
	ok, err := ancestor(ctx, cfg.Cwd, target, cfg.Entry.Revision)
	if err != nil {
		return "the repository could not be read"
	}
	if !ok {
		return "the target moved; a fast-forward is no longer possible"
	}
	snapshot := observe(ctx, cfg.Cwd, cfg.Repo, cfg.Entry.Target, []Declaration{{
		ID: cfg.Delivery.ID, Title: cfg.Delivery.Title, Branch: cfg.Delivery.Branch,
		Revision: cfg.Delivery.Revision, Review: cfg.Delivery.Review, Target: cfg.Delivery.Target}})
	for _, change := range snapshot.Changes {
		if change.ID == cfg.Delivery.ID && change.Validation == "failed" {
			return "the recorded evidence says the checks failed"
		}
	}
	return ""
}

// integrate moves the target to the revision, fast-forward only. A branch that a
// worktree holds is fast-forwarded there: the merge refuses a dirty tree and
// refuses anything that is not a fast-forward by itself. A branch no worktree
// holds moves by compare-and-swap, so a target that changed under the run is a
// refusal instead of a lost commit.
func integrate(ctx context.Context, cwd, target, revision string) error {
	wt, held, err := worktreeFor(ctx, cwd, target)
	if err != nil {
		return err
	}
	if held {
		if out, err := runGitOut(ctx, wt, "merge", "--ff-only", "--no-edit", revision); err != nil {
			return fmt.Errorf("%s", explain(out, err))
		}
		return nil
	}
	old, err := git(ctx, cwd, "rev-parse", "--verify", "refs/heads/"+target)
	if err != nil {
		return fmt.Errorf("the target branch is gone")
	}
	if out, err := runGitOut(ctx, cwd, "update-ref", "refs/heads/"+target, revision, old); err != nil {
		return fmt.Errorf("%s", explain(out, err))
	}
	return nil
}

// worktreeFor finds the worktree that holds a branch. Git allows one at most.
func worktreeFor(ctx context.Context, cwd, branch string) (string, bool, error) {
	out, err := runGitOut(ctx, cwd, "worktree", "list", "--porcelain")
	if err != nil {
		return "", false, fmt.Errorf("the worktrees could not be listed")
	}
	path := ""
	for _, line := range strings.Split(out, "\n") {
		switch {
		case strings.HasPrefix(line, "worktree "):
			path = strings.TrimPrefix(line, "worktree ")
		case strings.HasPrefix(line, "branch refs/heads/"):
			if strings.TrimPrefix(line, "branch refs/heads/") == branch {
				return path, true, nil
			}
		}
	}
	return "", false, nil
}

// runShell runs one declared command through the system shell, in the worktree
// the entry belongs to. A project's declaration is the owner's own text: an
// agent cannot put a command here.
func runShell(ctx context.Context, cwd, command string) (string, error) {
	c, cancel := context.WithTimeout(ctx, commandTimeout)
	defer cancel()
	cmd := exec.CommandContext(c, "/bin/sh", "-c", command)
	cmd.Dir = cwd
	var out cappedBuffer
	cmd.Stdout, cmd.Stderr = &out, &out
	err := cmd.Run()
	return strings.TrimSpace(out.String()), err
}

func runGitOut(ctx context.Context, cwd string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", append([]string{"--no-optional-locks", "-c", "core.fsmonitor=false"}, args...)...)
	cmd.Dir = cwd
	var out cappedBuffer
	cmd.Stdout, cmd.Stderr = &out, &out
	err := cmd.Run()
	return strings.TrimSpace(out.String()), err
}

// explain names why a command failed: its last line of output if it printed one,
// the error otherwise.
func explain(out string, err error) string {
	lines := strings.Split(strings.TrimSpace(out), "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		if line := strings.TrimSpace(lines[i]); line != "" {
			return line
		}
	}
	if err != nil {
		return err.Error()
	}
	return "no output"
}

func short(revision string) string {
	if len(revision) > 12 {
		return revision[:12]
	}
	return revision
}

// oneLine keeps a note to one line of at most noteLimit bytes, the shape the
// store accepts: whitespace collapses, and a note that is still too long is cut
// on a character boundary with an ellipsis that says so.
func oneLine(note string) string {
	note = strings.Join(strings.Fields(note), " ")
	if len(note) <= noteLimit {
		return note
	}
	cut := noteLimit - len("…")
	for cut > 0 && !utf8.RuneStart(note[cut]) {
		cut--
	}
	return note[:cut] + "…"
}
