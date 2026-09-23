package cliinstructions

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// write lays files out under root; a value starting with "->" makes a
// symbolic link to the named sibling.
func write(t *testing.T, root string, files map[string]string) {
	t.Helper()
	for rel, body := range files {
		p := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if target, ok := strings.CutPrefix(body, "->"); ok {
			if err := os.Symlink(target, p); err != nil {
				t.Fatal(err)
			}
			continue
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func git(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir, "-c", "user.email=t@t", "-c", "user.name=t", "-c", "init.defaultBranch=main"}, args...)...)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

// repo is a fresh git repository at home/code/repo holding files.
func repo(t *testing.T, files map[string]string) (home, root string) {
	t.Helper()
	home = t.TempDir()
	root = filepath.Join(home, "code", "repo")
	write(t, root, files)
	git(t, root, "init", "-q")
	return home, root
}

func resolve(t *testing.T, env Env) *Report {
	t.Helper()
	rep, err := Resolve(env)
	if err != nil {
		t.Fatal(err)
	}
	return rep
}

func cell(t *testing.T, rep *Report, path, cli string) Cell {
	t.Helper()
	for _, f := range rep.Files {
		if f.Path == path {
			return f.Cells[cli]
		}
	}
	var have []string
	for _, f := range rep.Files {
		have = append(have, f.Path)
	}
	t.Fatalf("no row %q (have %v)", path, have)
	return Cell{}
}

func has(rep *Report, id string) *Finding {
	for i := range rep.Findings {
		if rep.Findings[i].ID == id {
			return &rep.Findings[i]
		}
	}
	return nil
}

// want is one row of the decision table: a file, a CLI, the verdict.
type want struct {
	path, cli string
	status    Status
	by        string // for shadowed
}

func check(t *testing.T, rep *Report, rows []want) {
	t.Helper()
	for _, w := range rows {
		c := cell(t, rep, w.path, w.cli)
		if c.Status != w.status || (w.by != "" && c.By != w.by) {
			t.Errorf("%s / %s = %s (by %q: %s), want %s (by %q)", w.path, w.cli, c.Status, c.By, c.Why, w.status, w.by)
		}
	}
}

// The case the study opens with: both files in one folder, and each CLI
// picks differently.
func TestBothFilesInOneFolder(t *testing.T) {
	home, root := repo(t, map[string]string{"AGENTS.md": "a\n", "CLAUDE.md": "claude rules\n"})
	write(t, home, map[string]string{".grok/trusted_folders.toml": `[folders."` + root + `"]` + "\ntrusted = true\n"})
	rep := resolve(t, Env{Home: home, Root: root})
	check(t, rep, []want{
		{"CLAUDE.md", "claude-code", StatusReads, ""},
		{"AGENTS.md", "claude-code", StatusShadowed, "CLAUDE.md"},
		{"AGENTS.md", "opencode", StatusReads, ""},
		{"CLAUDE.md", "opencode", StatusShadowed, "AGENTS.md"},
		{"AGENTS.md", "pi", StatusReads, ""},
		{"CLAUDE.md", "pi", StatusShadowed, "AGENTS.md"},
		{"AGENTS.md", "omp", StatusReads, ""}, // measured: the tie goes to AGENTS.md
		{"CLAUDE.md", "omp", StatusShadowed, "AGENTS.md"},
		{"AGENTS.md", "grok", StatusReads, ""},
		{"CLAUDE.md", "grok", StatusReads, ""},
		{"AGENTS.md", "hermes", StatusReads, ""},
		{"CLAUDE.md", "hermes", StatusShadowed, "AGENTS.md"},
		{"AGENTS.md", "codex", StatusReads, ""},
		{"CLAUDE.md", "codex", StatusNotRead, ""},
		{"AGENTS.md", "muse", StatusReads, ""},
		{"CLAUDE.md", "muse", StatusShadowed, "AGENTS.md"},
		{"AGENTS.md", "agy", StatusReads, ""},
		{"CLAUDE.md", "agy", StatusNotRead, ""},
	})
	f := has(rep, "claude-split")
	if f == nil || !strings.Contains(f.Text, "Claude Code reads CLAUDE.md here, while") || f.Action == nil || f.Action.Path != "CLAUDE.md" {
		t.Fatalf("claude-split = %+v", f)
	}
}

// 3 of the 5 repositories on the study's machine point Claude at AGENTS.md
// in prose, which keeps Claude from loading it.
func TestClaudeProsePointer(t *testing.T) {
	home, root := repo(t, map[string]string{"AGENTS.md": "a\n", "CLAUDE.md": "# CLAUDE.md\n\nRead AGENTS.md before anything.\n"})
	rep := resolve(t, Env{Home: home, Root: root})
	check(t, rep, []want{{"AGENTS.md", "claude-code", StatusShadowed, "CLAUDE.md"}})
	if f := has(rep, "claude-pointer"); f == nil || !strings.Contains(f.Text, "A line @AGENTS.md in CLAUDE.md makes it load") {
		t.Fatalf("claude-pointer = %+v", f)
	}
	if has(rep, "claude-split") != nil {
		t.Fatal("a pointer is not also reported as a split")
	}
}

func TestClaudeImportAndLinkBringAgentsIn(t *testing.T) {
	for name, claude := range map[string]string{"import": "@AGENTS.md\n\n## Claude\n", "link": "->AGENTS.md"} {
		t.Run(name, func(t *testing.T) {
			home, root := repo(t, map[string]string{"AGENTS.md": "a\n", "CLAUDE.md": claude})
			rep := resolve(t, Env{Home: home, Root: root})
			check(t, rep, []want{{"AGENTS.md", "claude-code", StatusReads, ""}})
			if has(rep, "claude-split") != nil || has(rep, "claude-pointer") != nil {
				t.Fatalf("a bridged CLAUDE.md is not a finding: %+v", rep.Findings)
			}
		})
	}
}

// docs-videos/ in this repository holds the same text under both names:
// every CLI gets the same instructions, so there is nothing to report.
func TestSameTextUnderBothNamesIsNotASplit(t *testing.T) {
	home, root := repo(t, map[string]string{"AGENTS.md": "same\n", "CLAUDE.md": "same\n", "sub/AGENTS.md": "x\n", "sub/CLAUDE.md": "y\n"})
	rep := resolve(t, Env{Home: home, Root: root})
	var splits []string
	for _, f := range rep.Findings {
		if f.ID == "claude-split" {
			splits = append(splits, f.Text)
		}
	}
	if len(splits) != 1 || !strings.HasPrefix(splits[0], "In sub/, Claude Code reads CLAUDE.md, while") {
		t.Fatalf("splits = %q", splits)
	}
}

func TestEmptyPersonalFileIsNoRow(t *testing.T) {
	home, root := repo(t, map[string]string{"AGENTS.md": "a\n"})
	write(t, home, map[string]string{".gemini/GEMINI.md": ""})
	for _, f := range resolve(t, Env{Home: home, Root: root}).Files {
		if f.Scope == "personal" {
			t.Fatalf("empty personal file listed: %s", f.Path)
		}
	}
}

func TestClaudeLocalTurnsAgentsOff(t *testing.T) {
	home, root := repo(t, map[string]string{"AGENTS.md": "a\n", "CLAUDE.local.md": "mine\n"})
	rep := resolve(t, Env{Home: home, Root: root})
	check(t, rep, []want{{"AGENTS.md", "claude-code", StatusShadowed, "CLAUDE.local.md"}})
	if f := has(rep, "claude-local"); f == nil || f.Action == nil || f.Action.Kind != "settings" || f.Action.URL != "#/clis/claude-code/settings" {
		t.Fatalf("claude-local = %+v", f)
	}
}

func TestClaudeModes(t *testing.T) {
	cases := []struct {
		mode string
		want Status
	}{
		{"claude-md-and-agents-md", StatusReads},
		{"claude-md", StatusNotRead},
		{"managed-only", StatusNotRead},
	}
	for _, c := range cases {
		t.Run(c.mode, func(t *testing.T) {
			home, root := repo(t, map[string]string{"AGENTS.md": "a\n", "CLAUDE.md": "c\n"})
			write(t, home, map[string]string{".claude/settings.json": `{"pluginConfigs":{"agents-md@builtin":{"options":{"instructionFiles":"` + c.mode + `"}}}}`})
			rep := resolve(t, Env{Home: home, Root: root})
			check(t, rep, []want{{"AGENTS.md", "claude-code", c.want, ""}})
		})
	}
}

// PiCode's own layout: a worktree at <repo>/.worktrees/<name>. Measured:
// Pi, Claude Code, Grok and Hermes read the worktree's file only; Omp also
// reads the main checkout's and the one above the repository.
func TestNestedWorktree(t *testing.T) {
	home, root := repo(t, map[string]string{"AGENTS.md": "MAIN-CHECKOUT\n"})
	write(t, filepath.Dir(root), map[string]string{"AGENTS.md": "ABOVE-REPO\n"})
	git(t, root, "add", "AGENTS.md")
	git(t, root, "commit", "-qm", "init")
	git(t, root, "worktree", "add", "-q", ".worktrees/wt", "-b", "feat/wt")
	wt := filepath.Join(root, ".worktrees", "wt")
	write(t, wt, map[string]string{"AGENTS.md": "WORKTREE-BRANCH\n"})
	rep := resolve(t, Env{Home: home, Root: wt})
	if !rep.Worktree {
		t.Fatal("the worktree was not recognised as nested")
	}
	check(t, rep, []want{
		{"AGENTS.md", "pi", StatusReads, ""},
		{"~/code/repo/AGENTS.md", "pi", StatusShadowed, "AGENTS.md"},
		{"AGENTS.md", "claude-code", StatusReads, ""},
		{"~/code/repo/AGENTS.md", "claude-code", StatusNotRead, ""},
		{"~/code/repo/AGENTS.md", "hermes", StatusNotRead, ""},
		{"AGENTS.md", "omp", StatusReads, ""},
		{"~/code/repo/AGENTS.md", "omp", StatusReads, ""},
		{"~/code/AGENTS.md", "omp", StatusReads, ""},
	})
	if f := has(rep, "omp-worktree"); f == nil || f.Action == nil || f.Action.URL != ompWorktreeIssue {
		t.Fatalf("omp-worktree = %+v", f)
	}
}

func TestHermesDotHermesHidesAgents(t *testing.T) {
	home, root := repo(t, map[string]string{"AGENTS.md": "a\n", ".hermes.md": "h\n"})
	rep := resolve(t, Env{Home: home, Root: root})
	check(t, rep, []want{
		{".hermes.md", "hermes", StatusReads, ""},
		{"AGENTS.md", "hermes", StatusShadowed, ".hermes.md"},
	})
	if has(rep, "hermes-kind") == nil {
		t.Fatalf("hermes-kind missing: %+v", rep.Findings)
	}
}

// This repository's own AGENTS.md (21,055 characters) was truncated by
// Hermes at a 64k window and whole at 200k: the threshold lands at 88k.
func TestLimits(t *testing.T) {
	if got := hermesCut(21055, 0); got != "Hermes cuts it on models with a window under 88k tokens" {
		t.Fatalf("hermesCut = %q", got)
	}
	if hermesCut(20000, 0) != "" || hermesCut(5000, 4000) == "" {
		t.Fatal("hermesCut: the floor and an explicit cap")
	}
	home, root := repo(t, map[string]string{"AGENTS.md": strings.Repeat("x", 25000) + "\n"})
	write(t, home, map[string]string{".codex/config.toml": "project_doc_max_bytes = 10000\n[profiles.x]\nproject_doc_max_bytes = 1\n"})
	rep := resolve(t, Env{Home: home, Root: root})
	for cli, fragment := range map[string]string{"hermes": "under 105k tokens", "agy": "24,000 bytes", "codex": "10,000 bytes"} {
		if c := cell(t, rep, "AGENTS.md", cli); !strings.Contains(c.Cut, fragment) {
			t.Errorf("%s cut = %q, want it to mention %q", cli, c.Cut, fragment)
		}
	}
	if f := has(rep, "limit"); f == nil || !strings.Contains(f.Text, "25,001 bytes") || f.Action == nil || f.Action.Kind != "settings" {
		t.Fatalf("limit = %+v", f)
	}
}

func TestGrokTrustAndIgnore(t *testing.T) {
	home, root := repo(t, map[string]string{"AGENTS.md": "a\n", "CLAUDE.local.md": "l\n", ".gitignore": "CLAUDE.local.md\n"})
	rep := resolve(t, Env{Home: home, Root: root})
	check(t, rep, []want{{"AGENTS.md", "grok", StatusUntrusted, ""}})
	if has(rep, "grok-untrusted") == nil {
		t.Fatal("grok-untrusted missing")
	}
	// Trusting the parent folder is not trusting the repository (measured).
	write(t, home, map[string]string{".grok/trusted_folders.toml": `[folders."` + filepath.Dir(root) + `"]` + "\ntrusted = true\n"})
	check(t, resolve(t, Env{Home: home, Root: root}), []want{{"AGENTS.md", "grok", StatusUntrusted, ""}})
	write(t, home, map[string]string{".grok/trusted_folders.toml": `[folders."` + root + `"]` + "\ntrusted = true\n"})
	check(t, resolve(t, Env{Home: home, Root: root}), []want{
		{"AGENTS.md", "grok", StatusReads, ""},
		{"CLAUDE.local.md", "grok", StatusNotRead, ""}, // gitignored
	})
}

func TestSubfoldersAndStartFolder(t *testing.T) {
	home, root := repo(t, map[string]string{"AGENTS.md": "root\n", "pkg/AGENTS.md": "pkg\n"})
	rep := resolve(t, Env{Home: home, Root: root})
	check(t, rep, []want{
		{"pkg/AGENTS.md", "claude-code", StatusOnDemand, ""},
		{"pkg/AGENTS.md", "hermes", StatusOnDemand, ""},
		{"pkg/AGENTS.md", "agy", StatusOnDemand, ""},
		{"pkg/AGENTS.md", "codex", StatusNotRead, ""},
		{"pkg/AGENTS.md", "muse", StatusUnknown, ""},
	})
	if len(rep.Folders) != 2 || rep.Folders[1] != "pkg" {
		t.Fatalf("folders = %v", rep.Folders)
	}
	rep = resolve(t, Env{Home: home, Root: root, Start: filepath.Join(root, "pkg")})
	if rep.Start != "pkg" {
		t.Fatalf("start = %q", rep.Start)
	}
	for _, cli := range []string{"codex", "opencode", "hermes", "pi", "claude-code", "agy", "muse", "omp"} {
		for _, p := range []string{"AGENTS.md", "pkg/AGENTS.md"} {
			if c := cell(t, rep, p, cli); c.Status != StatusReads {
				t.Errorf("start pkg: %s / %s = %s (%s)", p, cli, c.Status, c.Why)
			}
		}
	}
	if _, err := Resolve(Env{Home: home, Root: filepath.Join(root, "pkg"), Start: root}); err == nil {
		t.Fatal("a start folder outside the workspace must be refused")
	}
}

func TestPersonalFiles(t *testing.T) {
	home, root := repo(t, map[string]string{"AGENTS.md": "a\n"})
	write(t, home, map[string]string{".claude/CLAUDE.md": "mine\n", ".codex/AGENTS.md": "codex mine\n"})
	rep := resolve(t, Env{Home: home, Root: root})
	check(t, rep, []want{
		{"~/.claude/CLAUDE.md", "claude-code", StatusReads, ""},
		{"~/.claude/CLAUDE.md", "opencode", StatusReads, ""},
		{"~/.claude/CLAUDE.md", "omp", StatusReads, ""}, // priority 80 beats ~/.codex/AGENTS.md's 70
		{"~/.codex/AGENTS.md", "omp", StatusShadowed, "~/.claude/CLAUDE.md"},
		{"~/.codex/AGENTS.md", "codex", StatusReads, ""},
		{"~/.codex/AGENTS.md", "pi", StatusNotRead, ""},
	})
}

func TestFindingsSpeakOnlyOfInstalledCLIs(t *testing.T) {
	home, root := repo(t, map[string]string{"AGENTS.md": "a\n", ".hermes.md": "h\n", "AGENTS.override.md": "o\n"})
	only := func(ids ...string) func(string) bool {
		return func(cli string) bool { return in(ids, cli) }
	}
	rep := resolve(t, Env{Home: home, Root: root, Installed: only("codex")})
	if has(rep, "hermes-kind") != nil || has(rep, "grok-untrusted") != nil || has(rep, "override-partial") != nil {
		t.Fatalf("findings about CLIs that are not installed: %+v", rep.Findings)
	}
	rep = resolve(t, Env{Home: home, Root: root, Installed: only("codex", "claude-code")})
	if f := has(rep, "override-partial"); f == nil || !strings.Contains(f.Text, "Claude Code read AGENTS.md instead") {
		t.Fatalf("override-partial = %+v", f)
	}
}

func TestImportsFinding(t *testing.T) {
	home, root := repo(t, map[string]string{"AGENTS.md": "@docs/rules.md\n", "docs/rules.md": "r\n"})
	rep := resolve(t, Env{Home: home, Root: root})
	f := has(rep, "imports")
	if f == nil || !strings.Contains(f.Text, "1 import line") || strings.Contains(f.Text, "Claude Code read") {
		t.Fatalf("imports = %+v", f)
	}
}

func TestOutsideARepository(t *testing.T) {
	home := t.TempDir()
	root := filepath.Join(home, "notes")
	write(t, root, map[string]string{"AGENTS.md": "a\n"})
	write(t, home, map[string]string{"AGENTS.md": "home\n"})
	rep := resolve(t, Env{Home: home, Root: root})
	check(t, rep, []want{
		{"AGENTS.md", "codex", StatusReads, ""},
		{"AGENTS.md", "hermes", StatusReads, ""},
		{"~/AGENTS.md", "pi", StatusReads, ""},      // Pi walks up to /
		{"~/AGENTS.md", "omp", StatusReads, ""},     // outside a repository Omp's walk ends at home, home included
		{"~/AGENTS.md", "codex", StatusNotRead, ""}, // Codex reads the start folder alone
	})
}

func TestList(t *testing.T) {
	for n, want := range map[int]string{1: "A", 2: "A and B", 3: "A, B and C", 5: "A, B, C and 2 more"} {
		if got := list([]string{"A", "B", "C", "D", "E"}[:n]); got != want {
			t.Errorf("list(%d) = %q, want %q", n, got, want)
		}
	}
}
