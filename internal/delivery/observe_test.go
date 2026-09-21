package delivery

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func fixture(t *testing.T) (string, string) {
	t.Helper()
	dir := t.TempDir()
	run(t, dir, "init", "-b", "main")
	run(t, dir, "-c", "user.name=Test", "-c", "user.email=test@example.invalid", "-c", "commit.gpgsign=false", "commit", "--allow-empty", "-m", "initial")
	return dir, filepath.Join(dir, ".git")
}
func run(t *testing.T, cwd string, args ...string) string {
	t.Helper()
	c := exec.Command("git", args...)
	c.Dir = cwd
	b, e := c.CombinedOutput()
	if e != nil {
		t.Fatalf("git %v: %v %s", args, e, b)
	}
	return strings.TrimSpace(string(b))
}
func commit(t *testing.T, dir string) string {
	run(t, dir, "-c", "user.name=Test", "-c", "user.email=test@example.invalid", "-c", "commit.gpgsign=false", "commit", "--allow-empty", "-m", "next-"+run(t, dir, "symbolic-ref", "--short", "HEAD"))
	return run(t, dir, "rev-parse", "HEAD")
}
func receipt(t *testing.T, repo string, r Receipt) {
	t.Helper()
	d := filepath.Join(repo, "picode-delivery")
	if e := os.MkdirAll(d, 0700); e != nil {
		t.Fatal(e)
	}
	b, _ := json.Marshal(r)
	if e := os.WriteFile(filepath.Join(d, r.ID+".json"), b, 0600); e != nil {
		t.Fatal(e)
	}
}
func record(t *testing.T, dir, repo, kind, source, outcome string) Receipt {
	t.Helper()
	now := time.Now().Add(-time.Second).UTC().Format(time.RFC3339Nano)
	r := Receipt{SchemaVersion: 1, ID: "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa", Kind: kind, RepositoryKey: repo, SourceRef: "feature", Source: source, Tree: run(t, dir, "rev-parse", source+"^{tree}"), TargetBefore: run(t, dir, "rev-parse", "main"), StartedAt: now, FinishedAt: now, Outcome: outcome, Clean: true}
	receipt(t, repo, r)
	return r
}
func TestObservationDecisionTable(t *testing.T) {
	dir, repo := fixture(t)
	run(t, dir, "checkout", "-b", "feature")
	sha := commit(t, dir)
	decl := []Declaration{{ID: "d", Title: "Fix", Branch: "feature", Revision: sha, Review: "requested"}}
	check := func(integration, validation, checkout string) {
		t.Helper()
		s := Observe(context.Background(), dir, repo, "main", decl)
		if len(s.Changes) != 1 {
			t.Fatalf("%+v", s)
		}
		c := s.Changes[0]
		if c.Integration != integration || c.Validation != validation || c.Checkout != checkout {
			t.Fatalf("%+v", c)
		}
	}
	check("not-integrated", "unknown", "clean") // O03, O06, O08
	record(t, dir, repo, "scoped", sha, "passed")
	check("not-integrated", "scoped-passed", "clean") // O09
	os.WriteFile(filepath.Join(dir, "dirty"), []byte("x"), 0600)
	check("not-integrated", "needs-recheck", "dirty")
	os.Remove(filepath.Join(dir, "dirty")) // O04,O11
	run(t, dir, "checkout", "main")
	run(t, dir, "merge", "--ff-only", "feature")
	record(t, dir, repo, "full-ci", sha, "failed")
	check("integrated", "failed", "not-present") // O05,O12
	run(t, dir, "branch", "-D", "feature")
	s := Observe(context.Background(), dir, repo, "main", decl)
	if s.Changes[0].SourceStatus != "removed" || s.Changes[0].Integration != "integrated" {
		t.Fatal(s)
	} // O14
	s = Observe(context.Background(), dir, repo, "absent", nil)
	if s.Complete || s.TargetOID != "" {
		t.Fatal(s)
	} // O02
	s = Observe(context.Background(), t.TempDir(), repo, "main", nil)
	if s.Complete || len(s.Issues) == 0 {
		t.Fatal(s)
	} // O01
}
func TestDivergenceAndUnknownHistory(t *testing.T) {
	dir, repo := fixture(t)
	run(t, dir, "checkout", "-b", "feature")
	sha := commit(t, dir)
	run(t, dir, "checkout", "main")
	commit(t, dir)
	s := Observe(context.Background(), dir, repo, "main", nil)
	if s.Changes[0].Integration != "update-needed" {
		t.Fatal(s)
	} // O07
	s = Observe(context.Background(), dir, repo, "main", []Declaration{{ID: "bad", Branch: "feature", Revision: strings.Repeat("a", 40)}})
	if s.Complete || s.Changes[0].Integration != "unknown" {
		t.Fatal(s)
	} // O15
	r := record(t, dir, repo, "scoped", sha, "started")
	r.FinishedAt = ""
	receipt(t, repo, r)
	s = Observe(context.Background(), dir, repo, "main", nil)
	if s.Changes[0].Validation == "passed" {
		t.Fatal(s)
	}
}
func TestReceiptConfinement(t *testing.T) {
	for _, mode := range []string{"symlink-directory", "symlink-file", "malformed", "oversized", "version", "repository", "time", "permissions"} {
		t.Run(mode, func(t *testing.T) {
			if mode == "permissions" && runtime.GOOS == "windows" {
				t.Skip("POSIX mode bits do not express Windows ACLs")
			}
			dir, repo := fixture(t)
			sha := run(t, dir, "rev-parse", "HEAD")
			r := record(t, dir, repo, "full-ci", sha, "passed")
			d := filepath.Join(repo, "picode-delivery")
			p := filepath.Join(d, r.ID+".json")
			switch mode {
			case "symlink-directory":
				os.Remove(p)
				os.Remove(d)
				if err := os.Symlink(t.TempDir(), d); err != nil {
					if runtime.GOOS == "windows" {
						t.Skipf("symlink privilege unavailable: %v", err)
					}
					t.Fatal(err)
				}
			case "symlink-file":
				os.Remove(p)
				if err := os.Symlink(filepath.Join(dir, "outside"), p); err != nil {
					if runtime.GOOS == "windows" {
						t.Skipf("symlink privilege unavailable: %v", err)
					}
					t.Fatal(err)
				}
			case "malformed":
				os.WriteFile(p, []byte("{"), 0600)
			case "oversized":
				os.WriteFile(p, []byte(strings.Repeat("x", 65537)), 0600)
			case "version":
				r.SchemaVersion = 99
				receipt(t, repo, r)
			case "repository":
				r.RepositoryKey = "elsewhere"
				receipt(t, repo, r)
			case "time":
				r.FinishedAt = "2000-01-01T00:00:00Z"
				receipt(t, repo, r)
			case "permissions":
				os.Chmod(p, 0644)
			}
			rows, issues := ReadReceipts(repo)
			if len(rows) != 0 || len(issues) == 0 {
				t.Fatalf("%+v %v", rows, issues)
			}
		})
	}
}
func TestReuseParity(t *testing.T) {
	for _, row := range []struct {
		s             *Scope
		tree, covered string
		dirty, want   bool
	}{
		{nil, "a", "b", false, false}, {&Scope{Tree: "a", Covered: "b"}, "a", "c", false, true},
		{&Scope{Tree: "a", Covered: "b"}, "c", "b", false, true}, {&Scope{Tree: "a", Covered: "b"}, "c", "c", false, false},
		{&Scope{Tree: "a", Covered: "b"}, "a", "b", true, false},
	} {
		if got := Reuse(row.s, row.tree, row.covered, row.dirty); got != row.want {
			t.Fatalf("%+v %v", row, got)
		}
	}
}

func TestUnstableAndBoundedObservation(t *testing.T) {
	dir, repo := fixture(t)
	run(t, dir, "branch", "feature")
	original := git
	defer func() { git = original }()
	reads := 0
	git = func(ctx context.Context, cwd string, args ...string) (string, error) {
		out, e := original(ctx, cwd, args...)
		if len(args) > 0 && args[0] == "for-each-ref" {
			reads++
			if reads%2 == 0 {
				out += "\nmoving " + run(t, dir, "rev-parse", "HEAD")
			}
		}
		return out, e
	}
	s := Observe(context.Background(), dir, repo, "main", nil)
	if s.Complete || reads != 4 || s.Changes[0].Integration != "unknown" {
		t.Fatalf("%+v reads=%d", s, reads)
	} // O23
	git = original
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	s = Observe(ctx, dir, repo, "main", nil)
	if s.Complete || len(s.Issues) == 0 {
		t.Fatal(s)
	} // O01 timeout
}

func TestCoveredContentReuse(t *testing.T) {
	dir, repo := fixture(t)
	run(t, dir, "checkout", "-b", "feature")
	os.WriteFile(filepath.Join(dir, "covered.txt"), []byte("same"), 0600)
	run(t, dir, "add", "covered.txt")
	sha := commit(t, dir)
	r := record(t, dir, repo, "scoped", sha, "passed")
	r.Scope = &Scope{Tree: r.Tree, Roots: []string{"covered.txt"}, Covered: covered(context.Background(), dir, sha, []string{"covered.txt"})}
	receipt(t, repo, r)
	os.WriteFile(filepath.Join(dir, "other.txt"), []byte("other"), 0600)
	run(t, dir, "add", "other.txt")
	commit(t, dir)
	s := Observe(context.Background(), dir, repo, "main", nil)
	if s.Changes[0].Validation != "scope-reusable" {
		t.Fatal(s.Changes)
	} // O10
	os.WriteFile(filepath.Join(dir, "covered.txt"), []byte("changed"), 0600)
	run(t, dir, "add", "covered.txt")
	commit(t, dir)
	s = Observe(context.Background(), dir, repo, "main", nil)
	if s.Changes[0].Validation != "needs-recheck" {
		t.Fatal(s.Changes)
	} // O11
}

func TestReviewTargetAndNoRepositoryHook(t *testing.T) {
	dir, repo := fixture(t)
	run(t, dir, "checkout", "-b", "feature")
	sha := commit(t, dir)
	marker := filepath.Join(dir, "hook-ran")
	hook := filepath.Join(dir, "monitor.sh")
	os.WriteFile(hook, []byte("#!/bin/sh\ntouch '"+marker+"'\n"), 0700)
	run(t, dir, "config", "core.fsmonitor", hook)
	s := Observe(context.Background(), dir, repo, "main", []Declaration{{ID: "d", Branch: "feature", Revision: sha, Review: "requested", Target: "other"}})
	if s.Changes[0].Review != "other-target" {
		t.Fatal(s)
	}
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Fatal("observer executed repository hook")
	}
}
