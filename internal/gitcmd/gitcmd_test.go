package gitcmd

import (
	"errors"
	"strings"
	"testing"
)

const hash = "0123456789abcdef0123456789abcdef01234567"

func TestComposeEveryAction(t *testing.T) {
	cases := []struct {
		action string
		args   Args
		want   string
		tier   Tier
	}{
		// Tier A
		{"fetch", Args{}, "git fetch --prune", TierA},
		{"fetch-into-local", Args{Target: "origin/feat-x", Name: "feat-x"}, "git fetch origin feat-x:feat-x", TierA},
		{"checkout", Args{Target: "feat/x"}, "git switch feat/x", TierA},
		{"checkout-detach", Args{Target: hash}, "git switch --detach " + hash, TierA},
		{"checkout-remote", Args{Target: "origin/feat-x", Name: "feat-x"}, "git switch -c feat-x --track origin/feat-x", TierA},
		{"create-branch", Args{Target: hash, Name: "feat/new"}, "git switch -c feat/new " + hash, TierA},
		{"create-tag", Args{Target: hash, Name: "v1.2.0"}, "git tag v1.2.0 " + hash, TierA},
		{"create-worktree", Args{Target: "feat/x", Name: "x"}, "git worktree add .worktrees/x feat/x", TierA},
		{"prune-worktrees", Args{}, "git worktree prune", TierA},
		{"restore-branch", Args{Target: hash, Name: "feat/x"}, "git branch feat/x " + hash, TierA},
		// Tier B
		{"pull", Args{}, "git pull --ff-only", TierB},
		{"pull-remote", Args{Target: "origin/main"}, "git pull --ff-only origin main", TierB},
		{"merge", Args{Target: "feat/x"}, "git merge --no-ff feat/x", TierB},
		{"rebase", Args{Target: "main"}, "git rebase main", TierB},
		{"cherry-pick", Args{Target: hash}, "git cherry-pick " + hash, TierB},
		{"revert", Args{Target: hash}, "git revert --no-edit " + hash, TierB},
		{"reset-soft", Args{Target: hash}, "git reset --soft " + hash, TierB},
		{"reset-mixed", Args{Target: hash}, "git reset --mixed " + hash, TierB},
		{"rename-branch", Args{Target: "old", Name: "new"}, "git branch -m old new", TierB},
		{"delete-branch", Args{Target: "feat/done"}, "git branch -d feat/done", TierB},
		{"commit", Args{Message: "fix the thing"}, "git add -A && git commit -m 'fix the thing'", TierB},
		{"pr", Args{}, "gh pr create --fill", TierB},
		// Tier C
		{"push", Args{Branch: "main", Upstream: "origin/main"}, "git push", TierC},
		{"push", Args{Branch: "feat/x"}, "git push -u origin feat/x", TierC},
		{"push-force", Args{Branch: "main", Upstream: "origin/main"}, "git push --force-with-lease", TierC},
		{"push-force", Args{Branch: "feat/x"}, "git push --force-with-lease -u origin feat/x", TierC},
		{"commit-push", Args{Message: "ship", Branch: "main", Upstream: "origin/main"}, "git add -A && git commit -m 'ship' && git push", TierC},
		{"push-tag", Args{Target: "v1.0.0"}, "git push origin tag v1.0.0", TierC},
		{"delete-branch-force", Args{Target: "feat/x"}, "git branch -D feat/x", TierC},
		{"delete-remote-branch", Args{Target: "origin/feat-x"}, "git push origin --delete feat-x", TierC},
		{"delete-tag", Args{Target: "v0.9"}, "git tag -d v0.9", TierC},
		{"reset-hard", Args{Target: hash}, "git reset --hard " + hash, TierC},
		{"discard", Args{}, "git restore --source=HEAD --staged --worktree -- .", TierC},
		{"clean", Args{}, "git clean -fd", TierC},
		{"worktree-remove", Args{Name: "x"}, "git worktree remove .worktrees/x", TierC},
		{"worktree-remove-force", Args{Name: "x"}, "git worktree remove --force .worktrees/x", TierC},
	}
	seen := map[string]bool{}
	for _, c := range cases {
		got, tier, err := Compose(c.action, c.args)
		if err != nil {
			t.Errorf("%s: %v", c.action, err)
			continue
		}
		if got != c.want {
			t.Errorf("%s = %q; want %q", c.action, got, c.want)
		}
		if tier != c.tier {
			t.Errorf("%s tier = %q; want %q", c.action, tier, c.tier)
		}
		seen[c.action] = true
	}
	// A new action without a case here is an action nobody proved composes.
	for _, action := range Actions() {
		if !seen[action] {
			t.Errorf("action %q has no composition case", action)
		}
	}
}

// Never a plain --force. The lease is the whole reason a force push is
// offered at all (ADR-0096).
func TestForcePushAlwaysCarriesTheLease(t *testing.T) {
	for _, a := range []Args{{Branch: "main", Upstream: "origin/main"}, {Branch: "feat/x"}} {
		got, _, err := Compose("push-force", a)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(got, "--force-with-lease") {
			t.Errorf("push-force = %q; want --force-with-lease", got)
		}
		if strings.Contains(got, "--force ") || strings.HasSuffix(got, "--force") {
			t.Errorf("push-force = %q; a plain --force must never be composed", got)
		}
	}
}

// Everything that reaches argv is validated, so nothing needs quoting — and
// anything that would need it is refused instead.
func TestRefusesTargetsThatWouldNotSurviveACommandLine(t *testing.T) {
	bad := []string{
		"", "-f", "--force", "..", "main..other", "HEAD@{1}", "feat/x ; rm -rf /",
		"feat/$(whoami)", "feat/`id`", "feat/x\nmain", ".hidden", "feat/x/", "x.lock",
		"a'b", `a"b`, "a|b", "a&b", "a>b", strings.Repeat("x", 256),
	}
	for _, target := range bad {
		for _, action := range []string{"merge", "rebase", "cherry-pick", "checkout", "reset-hard", "delete-tag"} {
			if _, _, err := Compose(action, Args{Target: target}); !errors.Is(err, ErrTarget) {
				t.Errorf("%s with target %q: err = %v; want ErrTarget", action, target, err)
			}
		}
	}
}

func TestRefusesNamesThatWouldNotSurviveACommandLine(t *testing.T) {
	bad := []string{"", "-x", ".hidden", "a b", "a;b", "a$b", "../escape", "x.lock"}
	for _, n := range bad {
		if _, _, err := Compose("create-branch", Args{Target: hash, Name: n}); !errors.Is(err, ErrName) {
			t.Errorf("create-branch with name %q: err = %v; want ErrName", n, err)
		}
		if _, _, err := Compose("worktree-remove", Args{Name: n}); !errors.Is(err, ErrName) {
			t.Errorf("worktree-remove with name %q: err = %v; want ErrName", n, err)
		}
	}
}

// A worktree name is a single path segment under .worktrees/, so a traversal
// cannot address a directory outside it.
func TestWorktreeNameCannotEscape(t *testing.T) {
	for _, n := range []string{"../../etc", "..", "./x", "/abs"} {
		if _, _, err := Compose("create-worktree", Args{Target: "main", Name: n}); err == nil {
			t.Errorf("create-worktree accepted the escaping name %q", n)
		}
	}
}

func TestCommitMessageQuoting(t *testing.T) {
	got, _, err := Compose("commit", Args{Message: "it's a fix, \"quoted\" and $shell-ish `too`"})
	if err != nil {
		t.Fatal(err)
	}
	want := `git add -A && git commit -m 'it'\''s a fix, "quoted" and $shell-ish ` + "`too`'"
	if got != want {
		t.Errorf("commit =\n %q\nwant\n %q", got, want)
	}
}

func TestRefusesBadCommitMessages(t *testing.T) {
	for _, m := range []string{"", "   ", "one\ntwo", "bell\x07", strings.Repeat("x", 201)} {
		if _, _, err := Compose("commit", Args{Message: m}); !errors.Is(err, ErrMessage) {
			t.Errorf("commit with message %q: err = %v; want ErrMessage", m, err)
		}
	}
}

// A push with no upstream needs the branch to publish; a detached HEAD has
// none, and the composer says so rather than guessing.
func TestPushWithoutABranchOrUpstreamIsRefused(t *testing.T) {
	if _, _, err := Compose("push", Args{}); !errors.Is(err, ErrNoBranch) {
		t.Errorf("push on a detached HEAD: err = %v; want ErrNoBranch", err)
	}
}

// A remote-tracking name that does not start with its remote is not one, and
// silently pushing to a branch of that name would be a different act.
func TestRemoteActionsRequireARemoteTrackingName(t *testing.T) {
	for _, action := range []string{"fetch-into-local", "pull-remote", "delete-remote-branch"} {
		if _, _, err := Compose(action, Args{Target: "feat-x", Name: "feat-x"}); !errors.Is(err, ErrTarget) {
			t.Errorf("%s with a bare branch name: err = %v; want ErrTarget", action, err)
		}
	}
	if _, _, err := Compose("delete-remote-branch", Args{Target: "upstream/feat-x", Remote: "upstream"}); err != nil {
		t.Errorf("a non-origin remote must work: %v", err)
	}
}

func TestUnknownActionIsRefused(t *testing.T) {
	if _, _, err := Compose("rm-rf", Args{}); !errors.Is(err, ErrUnknownAction) {
		t.Errorf("err = %v; want ErrUnknownAction", err)
	}
	if _, ok := TierOf("rm-rf"); ok {
		t.Error("TierOf claimed to know an action that does not exist")
	}
}

// Nothing this package refuses to compose may exist as a tier either: a
// caller that gates on TierOf must not find a gate for a command it cannot
// build.
func TestEveryActionHasATierAndAVerb(t *testing.T) {
	for _, action := range Actions() {
		tier, ok := TierOf(action)
		if !ok || (tier != TierA && tier != TierB && tier != TierC) {
			t.Errorf("%s: tier = %q, ok = %v", action, tier, ok)
		}
		if v := Verb(action, Args{Target: "feat/x", Name: "x"}); v == "" {
			t.Errorf("%s has no plain-words verb", action)
		}
	}
	if Verb("rm-rf", Args{}) != "" {
		t.Error("Verb answered for an unknown action")
	}
}

// The verb is what a toast and an agent prompt carry; a command in it would
// be a second, unvalidated way to say the same thing.
func TestVerbNeverCarriesACommand(t *testing.T) {
	for _, action := range Actions() {
		v := Verb(action, Args{Target: hash, Name: "x", Branch: "main"})
		if strings.Contains(v, "git ") || strings.Contains(v, "--") {
			t.Errorf("%s verb reads like a command: %q", action, v)
		}
	}
}

// A full object name is unreadable in a sentence; the verb shortens it the
// way the graph's rows do.
func TestVerbShortensAHash(t *testing.T) {
	if got := Verb("cherry-pick", Args{Target: hash}); got != "cherry-pick 0123456" {
		t.Errorf("verb = %q; want the short hash", got)
	}
	if got := Verb("merge", Args{Target: "feat/x"}); got != "merge feat/x" {
		t.Errorf("verb = %q; a branch name stays whole", got)
	}
}

func TestPromptNeverCarriesACommand(t *testing.T) {
	for _, action := range Actions() {
		p := Prompt(action, Args{Target: "feat/x", Name: "x", Branch: "main", Message: "m"}, "/repo")
		if p == "" {
			t.Errorf("%s has no prompt", action)
			continue
		}
		if strings.Contains(p, "git ") {
			t.Errorf("%s prompt reads like a command: %q", action, p)
		}
		if !strings.HasPrefix(p, "In /repo,") {
			t.Errorf("%s prompt does not name the folder: %q", action, p)
		}
	}
	if Prompt("rm-rf", Args{}, "/repo") != "" {
		t.Error("Prompt answered for an unknown action")
	}
}

// Every action that can lose work has to say so before an agent acts.
func TestDestructivePromptsWarnAboutLoss(t *testing.T) {
	for _, action := range []string{"reset-hard", "discard", "clean", "delete-branch-force", "delete-remote-branch", "worktree-remove-force"} {
		p := Prompt(action, Args{Target: "origin/feat-x", Name: "x"}, "/repo")
		if !strings.Contains(p, "will be lost") {
			t.Errorf("%s prompt does not warn about losing work: %q", action, p)
		}
	}
}

// The rules ADR-0078 wrote into the six prompts survive here.
func TestPromptRules(t *testing.T) {
	if p := Prompt("pull", Args{}, "/repo"); !strings.Contains(p, "fast-forward only") || !strings.Contains(p, "stop and tell me why") {
		t.Errorf("pull prompt lost its rule: %q", p)
	}
	if p := Prompt("push", Args{Branch: "x"}, "/repo"); !strings.Contains(p, "Never force-push") {
		t.Errorf("push prompt lost its rule: %q", p)
	}
	if p := Prompt("push-force", Args{Branch: "x"}, "/repo"); !strings.Contains(p, "--force-with-lease") || !strings.Contains(p, "never a plain") {
		t.Errorf("push-force prompt must name the lease: %q", p)
	}
	if p := Prompt("merge", Args{Target: "feat/x"}, "/repo"); !strings.Contains(p, "stop and tell me which files") {
		t.Errorf("merge prompt lost its conflict rule: %q", p)
	}
	// A commit with no message asks the agent to write one (ADR-0078).
	if p := Prompt("commit", Args{}, "/repo"); !strings.Contains(p, "a message you write from the changes") {
		t.Errorf("commit without a message: %q", p)
	}
	if p := Prompt("commit", Args{Message: "fix it"}, "/repo"); !strings.Contains(p, `"fix it"`) {
		t.Errorf("commit with a message: %q", p)
	}
}

// The catalog is the contract the browser reads instead of carrying its own
// copy of the tiers. Every action appears once, with a tier and a needs list
// that matches what Compose actually refuses without.
func TestCatalogMatchesWhatComposeRequires(t *testing.T) {
	cat := Catalog()
	if len(cat) != len(Actions()) {
		t.Fatalf("catalog has %d entries for %d actions", len(cat), len(Actions()))
	}
	for _, info := range cat {
		if info.Tier != TierA && info.Tier != TierB && info.Tier != TierC {
			t.Errorf("%s: tier %q", info.ID, info.Tier)
		}
		if info.Needs == nil {
			t.Errorf("%s: needs is null, which JSON-encodes differently from an empty list", info.ID)
		}
		// Composing with every declared field empty must fail exactly when
		// the action declares it needs something.
		_, _, err := Compose(info.ID, Args{Branch: "main", Upstream: "origin/main"})
		if len(info.Needs) > 0 && err == nil {
			t.Errorf("%s declares %v but composes with nothing", info.ID, info.Needs)
		}
		if len(info.Needs) == 0 && err != nil {
			t.Errorf("%s declares no needs but composing failed: %v", info.ID, err)
		}
	}
}

// A caller that fills exactly the declared fields must get a command.
func TestFillingTheDeclaredNeedsIsEnough(t *testing.T) {
	for _, info := range Catalog() {
		a := Args{Branch: "main", Upstream: "origin/main"}
		for _, need := range info.Needs {
			switch need {
			case "target":
				a.Target = "origin/feat-x"
			case "name":
				a.Name = "feat-x"
			case "message":
				a.Message = "a message"
			default:
				t.Fatalf("%s declares an unknown need %q", info.ID, need)
			}
		}
		if _, _, err := Compose(info.ID, a); err != nil {
			t.Errorf("%s with its declared needs filled: %v", info.ID, err)
		}
	}
}
