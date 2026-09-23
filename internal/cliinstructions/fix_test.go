package cliinstructions

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func read(t *testing.T, p string) string {
	t.Helper()
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// One row per case the fixes can meet (ADR-0204's decision table).
func TestProposeFix(t *testing.T) {
	cases := []struct {
		name  string
		files map[string]string
		id    string
		want  map[string]string // path → text after; nil = no fix
		err   error
	}{
		{"a pointer-only CLAUDE.md becomes the import",
			map[string]string{"AGENTS.md": "a\n", "CLAUDE.md": "# CLAUDE.md\n\nRead AGENTS.md first.\n"},
			"bridge:CLAUDE.md", map[string]string{"CLAUDE.md": "@AGENTS.md\n"}, nil},
		{"a CLAUDE.md with its own text keeps it under the import",
			map[string]string{"AGENTS.md": "a\n", "CLAUDE.md": "Read AGENTS.md.\n\nAlso: run make test.\n"},
			"bridge:CLAUDE.md", map[string]string{"CLAUDE.md": "@AGENTS.md\n\nRead AGENTS.md.\n\nAlso: run make test.\n"}, nil},
		{"a .claude/CLAUDE.md imports the folder's AGENTS.md",
			map[string]string{"AGENTS.md": "a\n", ".claude/CLAUDE.md": "See AGENTS.md.\n"},
			"bridge:.claude/CLAUDE.md", map[string]string{".claude/CLAUDE.md": "@../AGENTS.md\n"}, nil},
		{"already bridged: no fix",
			map[string]string{"AGENTS.md": "a\n", "CLAUDE.md": "@AGENTS.md\n"}, "bridge:CLAUDE.md", nil, ErrNoFix},
		{"no AGENTS.md beside it: no fix",
			map[string]string{"CLAUDE.md": "Read AGENTS.md.\n"}, "bridge:CLAUDE.md", nil, ErrNoFix},
		{"a CLAUDE.md that never names AGENTS.md: no fix",
			map[string]string{"AGENTS.md": "a\n", "CLAUDE.md": "Claude rules.\n"}, "bridge:CLAUDE.md", nil, ErrNoFix},
		{"a new personal file and a new .gitignore",
			map[string]string{}, "personal:CLAUDE.local.md",
			map[string]string{"CLAUDE.local.md": "", ".gitignore": "CLAUDE.local.md\n"}, nil},
		{"an existing personal file gets only the .gitignore line",
			map[string]string{"AGENTS.override.md": "mine\n", ".gitignore": "node_modules"}, "personal:AGENTS.override.md",
			map[string]string{".gitignore": "node_modules\nAGENTS.override.md\n"}, nil},
		{"a personal file already ignored: no fix",
			map[string]string{"CLAUDE.local.md": "mine\n", ".gitignore": "/CLAUDE.local.md\n"}, "personal:CLAUDE.local.md", nil, ErrNoFix},
		{"only the two personal names", map[string]string{}, "personal:NOTES.md", nil, ErrNoFix},
		{"an unknown kind", map[string]string{}, "rewrite:AGENTS.md", nil, ErrNoFix},
		{"outside the workspace", map[string]string{}, "personal:../CLAUDE.local.md", nil, os.ErrPermission},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			root := t.TempDir()
			write(t, root, c.files)
			fix, err := ProposeFix(root, c.id)
			if c.err != nil {
				if !errors.Is(err, c.err) {
					t.Fatalf("err = %v, want %v", err, c.err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			got := map[string]string{}
			for _, ch := range fix.Changes {
				got[ch.Path] = ch.After
			}
			if len(got) != len(c.want) {
				t.Fatalf("changes = %v, want %v", got, c.want)
			}
			for p, want := range c.want {
				if got[p] != want {
					t.Errorf("%s after = %q, want %q", p, got[p], want)
				}
			}
		})
	}
}

func TestApplyFixWritesOnlyWhatWasShown(t *testing.T) {
	root := t.TempDir()
	write(t, root, map[string]string{"AGENTS.md": "a\n", "CLAUDE.md": "Read AGENTS.md.\n"})
	fix, err := ProposeFix(root, "bridge:CLAUDE.md")
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]string{"CLAUDE.md": fix.Changes[0].Hash}
	// The file moves after the diff was shown: refused, nothing written.
	write(t, root, map[string]string{"CLAUDE.md": "Read AGENTS.md, please.\n"})
	if _, err := ApplyFix(root, "bridge:CLAUDE.md", seen); !errors.Is(err, ErrDrift) {
		t.Fatalf("drift: err = %v", err)
	}
	if got := read(t, filepath.Join(root, "CLAUDE.md")); got != "Read AGENTS.md, please.\n" {
		t.Fatalf("a refused write changed the file: %q", got)
	}
	fix, _ = ProposeFix(root, "bridge:CLAUDE.md")
	if _, err := ApplyFix(root, "bridge:CLAUDE.md", map[string]string{"CLAUDE.md": fix.Changes[0].Hash}); err != nil {
		t.Fatal(err)
	}
	if got := read(t, filepath.Join(root, "CLAUDE.md")); got != "@AGENTS.md\n" {
		t.Fatalf("after apply: %q", got)
	}
}

func TestApplyPersonalNeverOverwrites(t *testing.T) {
	root := t.TempDir()
	write(t, root, map[string]string{"CLAUDE.local.md": "my notes\n"})
	fix, err := ProposeFix(root, "personal:CLAUDE.local.md")
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]string{}
	for _, c := range fix.Changes {
		seen[c.Path] = c.Hash
	}
	if _, err := ApplyFix(root, "personal:CLAUDE.local.md", seen); err != nil {
		t.Fatal(err)
	}
	if got := read(t, filepath.Join(root, "CLAUDE.local.md")); got != "my notes\n" {
		t.Fatalf("personal file overwritten: %q", got)
	}
	if got := read(t, filepath.Join(root, ".gitignore")); got != "CLAUDE.local.md\n" {
		t.Fatalf(".gitignore = %q", got)
	}
	// A new file expected where one now exists (hash "" vs a real one): drift.
	root2 := t.TempDir()
	fix2, _ := ProposeFix(root2, "personal:AGENTS.override.md")
	seen2 := map[string]string{}
	for _, c := range fix2.Changes {
		seen2[c.Path] = c.Hash
	}
	write(t, root2, map[string]string{".gitignore": "x\n"})
	if _, err := ApplyFix(root2, "personal:AGENTS.override.md", seen2); !errors.Is(err, ErrDrift) {
		t.Fatalf("a file that appeared since the diff: err = %v", err)
	}
}

func TestFixRefusesLinks(t *testing.T) {
	root := t.TempDir()
	write(t, root, map[string]string{"AGENTS.md": "Read AGENTS.md\n", "CLAUDE.md": "->AGENTS.md"})
	if _, err := ProposeFix(root, "bridge:CLAUDE.md"); !errors.Is(err, os.ErrPermission) {
		t.Fatalf("a linked CLAUDE.md: err = %v", err)
	}
	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(root, "out")); err != nil {
		t.Fatal(err)
	}
	if _, err := ProposeFix(root, "personal:out/CLAUDE.local.md"); !errors.Is(err, os.ErrPermission) {
		t.Fatalf("a folder linked outside: err = %v", err)
	}
}

func TestFindingsCarryTheirFix(t *testing.T) {
	home, root := repo(t, map[string]string{"AGENTS.md": "a\n", "CLAUDE.md": "Read AGENTS.md.\n", "sub/AGENTS.override.md": "mine\n"})
	rep := resolve(t, Env{Home: home, Root: root})
	if f := has(rep, "claude-pointer"); f == nil || f.Fix != "bridge:CLAUDE.md" {
		t.Fatalf("claude-pointer = %+v", f)
	}
	if f := has(rep, "personal-shared"); f == nil || f.Fix != "personal:sub/AGENTS.override.md" {
		t.Fatalf("personal-shared = %+v", f)
	}
}
