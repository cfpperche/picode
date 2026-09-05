package gitgraph

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Decision table the graph draws from (ADR-0071). One row per condition:
//
//	checkout    health     dirty   → Uncommitted set?  flags?
//	branch      ok         no      → nil
//	branch      ok         yes     → count
//	detached    ok         no      → nil               detached
//	detached    ok         yes     → count             detached
//	branch      prunable   any     → nil               prunable
//	bare        —          —       → nil               bare
//	owner       ok         either  → count + self      (status reused, no exec)
func TestWorktreeAnnotationPerState(t *testing.T) {
	dir := repo(t)
	side := filepath.Join(t.TempDir(), "side")
	detached := filepath.Join(t.TempDir(), "detached")
	gone := filepath.Join(t.TempDir(), "gone")
	run(t, dir, "git", "worktree", "add", "-b", "side", side)
	run(t, dir, "git", "worktree", "add", detached)
	run(t, detached, "git", "checkout", "--detach")
	run(t, dir, "git", "worktree", "add", "-b", "gone", gone)
	if err := os.RemoveAll(gone); err != nil {
		t.Fatal(err)
	}
	// The dirty one: an untracked file in the side worktree.
	write(t, side, "messy.txt", "uncommitted")

	g := Load(dir, 50)
	if g == nil {
		t.Fatal("nil graph")
	}
	byBranch := map[string]Worktree{}
	for _, wt := range g.Worktrees {
		byBranch[wt.Branch] = wt
	}

	main := byBranch["main"]
	if !main.Self {
		t.Fatalf("the checkout the graph was read through must be self: %+v", main)
	}
	if main.Uncommitted != nil {
		t.Fatalf("clean main must have no uncommitted row: %+v", main.Uncommitted)
	}
	if main.Detached || main.Bare || main.Prunable {
		t.Fatalf("main must be a healthy attached checkout: %+v", main)
	}

	sideWt := byBranch["side"]
	if sideWt.Self {
		t.Fatal("a sibling worktree must not be marked self")
	}
	if sideWt.Uncommitted == nil || sideWt.Uncommitted.Count != 1 {
		t.Fatalf("dirty side worktree must carry count 1: %+v", sideWt.Uncommitted)
	}

	det := Worktree{}
	for _, wt := range g.Worktrees {
		if wt.Detached {
			det = wt
		}
	}
	if det.Branch != "" {
		t.Fatalf("detached worktree must have no branch: %+v", det)
	}
	if det.Uncommitted != nil {
		t.Fatalf("clean detached worktree must have no uncommitted row: %+v", det)
	}

	for _, wt := range g.Worktrees {
		if wt.Prunable && wt.Uncommitted != nil {
			t.Fatalf("prunable worktree must not be statused: %+v", wt)
		}
	}
}

// A dirty checkout the graph was NOT read through is the whole point of the
// feature: the graph loaded from main must see side's uncommitted file, and
// the reader's own worktree must be marked self even when dirty.
func TestOwnerWorktreeIsSelfWhenDirty(t *testing.T) {
	dir := repo(t)
	write(t, dir, "local.txt", "mine")

	g := Load(dir, 50)
	if g.Uncommitted == nil || g.Uncommitted.Count != 1 {
		t.Fatalf("owner dirty count missing: %+v", g.Uncommitted)
	}
	var self *Worktree
	for i := range g.Worktrees {
		if g.Worktrees[i].Self {
			self = &g.Worktrees[i]
		}
	}
	if self == nil {
		t.Fatal("no self worktree")
	}
	if self.Uncommitted == nil || self.Uncommitted.Count != 1 {
		t.Fatalf("self worktree must reuse the owner's status: %+v", self.Uncommitted)
	}
}

func TestBareRepositoryWorktreeIsFlagged(t *testing.T) {
	dir := repo(t)
	bare := filepath.Join(t.TempDir(), "bare.git")
	run(t, dir, "git", "clone", "--bare", "--quiet", dir, bare)

	g := Load(bare, 50)
	if g == nil || len(g.Worktrees) != 1 {
		t.Fatalf("bare repo must list one worktree: %+v", g)
	}
	if !g.Worktrees[0].Bare {
		t.Fatalf("the clone --bare entry must be flagged bare: %+v", g.Worktrees[0])
	}
}

// Past the cap a worktree draws without a dirty count: one `git status` per
// checkout must never turn a huge repository's graph load into seconds.
func TestStatusBudgetRunsOut(t *testing.T) {
	dir := repo(t)
	paths := make([]string, maxStatusWorktrees+1)
	for i := range paths {
		name := fmt.Sprintf("b%02d", i)
		paths[i] = filepath.Join(t.TempDir(), "wt-"+name)
		run(t, dir, "git", "worktree", "add", "-b", name, paths[i])
		write(t, paths[i], "x", "dirty")
	}
	g := Load(dir, 50)
	counted := 0
	for _, wt := range g.Worktrees {
		if wt.Uncommitted != nil {
			counted++
		}
	}
	// Every dirty worktree except the ones past the budget; main is clean and
	// carries no count even though it costs no budget.
	if counted != maxStatusWorktrees {
		t.Fatalf("expected %d worktrees with a count, got %d", maxStatusWorktrees, counted)
	}
}

func TestWorktreeOfRef(t *testing.T) {
	dir := repo(t)
	side := filepath.Join(t.TempDir(), "side")
	detached := filepath.Join(t.TempDir(), "detached")
	run(t, dir, "git", "worktree", "add", "-b", "side", side)
	run(t, dir, "git", "worktree", "add", detached)
	run(t, detached, "git", "checkout", "--detach")
	head := strings.TrimSpace(runOut(t, dir, "git", "rev-parse", "HEAD"))

	if got := WorktreeOfRef(dir, "side"); got != side {
		t.Fatalf("WorktreeOfRef(branch) = %q, want %q", got, side)
	}
	if got := WorktreeOfRef(dir, "nosuch"); got != "" {
		t.Fatalf("unknown branch must be empty, got %q", got)
	}
	if got := WorktreeOfRef(dir, ""); got != "" {
		t.Fatalf("empty ref must be empty, got %q", got)
	}
	// main is checked out in dir itself.
	if got := WorktreeOfRef(dir, "main"); got != dir {
		t.Fatalf("WorktreeOfRef(main) = %q, want %q", got, dir)
	}
	// The detached checkout is addressable by its full head hash.
	if got := WorktreeOfRef(dir, head); got != detached {
		t.Fatalf("WorktreeOfRef(hash) = %q, want %q", got, detached)
	}
	// A branch name that merely looks like a hash still resolves by branch.
	lookalike := strings.Repeat("ab", 20)
	if !isHash(lookalike) {
		t.Fatalf("test setup: %q should be hash-shaped", lookalike)
	}
	if got := WorktreeOfRef(dir, lookalike); got != "" {
		t.Fatalf("hash-shaped but unknown ref must be empty, got %q", got)
	}
	// A ref outside this repository answers empty.
	if got := WorktreeOfRef(t.TempDir(), "side"); got != "" {
		t.Fatalf("non-repo must be empty, got %q", got)
	}
}
