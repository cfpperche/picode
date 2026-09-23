// Package gitcmd composes the exact git command an action means, and says
// which risk tier it belongs to (ADR-0096).
//
// It executes nothing. The command it returns travels to one of ADR-0078's
// three doors — typed into the user's shell, typed and submitted behind the
// repository interlock, or described to the agent that owns the folder — and
// git runs there, with that user's credentials, hooks and signing. The
// service process never runs git: its environment has no ssh-agent socket,
// no pinentry and no credential helper, so a push would fail here precisely
// where the terminal succeeds.
//
// Composition lives on the server because two clients composing shell strings
// for thirty-odd actions is two places to get quoting wrong. The browser
// sends an action and its arguments; what it shows as a preview is what this
// package returned for the same input.
package gitcmd

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

// Tier is the risk class of an action. It decides which doors an action may
// take and what gate stands in front of it.
//
//	A  additive, local, recoverable from the reflog
//	B  moves HEAD or history, still recoverable
//	C  publishes, or destroys work
//
// The gate is the caller's to apply, with one exception this package does
// enforce for its callers to check: a tier C action submitted by PiCode
// itself (the "run" door) needs an explicit confirmation, because there the
// human never presses Enter.
type Tier string

const (
	TierA Tier = "A"
	TierB Tier = "B"
	TierC Tier = "C"
)

// Args are the pieces an action may need. Every field that reaches a command
// line is validated; nothing else is interpolated.
type Args struct {
	// Target is what the reader pointed at: a ref name or a full object name.
	Target string
	// Branch is the checkout's current branch, and Upstream its upstream —
	// both facts about where the command will run, not about the target.
	Branch   string
	Upstream string
	// Remote defaults to origin when empty.
	Remote string
	// Name is a *new* name the action creates: a branch, a tag, a worktree
	// directory.
	Name string
	// Message is a commit subject. One line, quoted for a POSIX shell.
	Message string
	// RepoRoot is the repository's top-level folder — the parent of its
	// common dir. A worktree command names its checkout absolutely under
	// RepoRoot/.worktrees/, so the same action read from a sibling worktree
	// does not nest a checkout inside that worktree (the command would
	// otherwise be relative to wherever the terminal sits). Empty means the
	// relative form, for callers that have no repository at hand.
	RepoRoot string
	// Remotes are the repository's configured remote names. A remote-branch
	// action finds its remote in the target's own prefix — origin/feat-x
	// belongs to origin, upstream/feat-x to upstream — rather than assuming
	// origin, which turned a second remote's branches into "not a branch".
	Remotes []string
}

// Errors callers translate into a 400. They name the field, so the browser
// can point at the input that is wrong.
var (
	ErrUnknownAction = errors.New("unknown action")
	ErrTarget        = errors.New("that target is not a branch, tag or commit name")
	ErrName          = errors.New("that name is not a valid branch, tag or folder name")
	ErrMessage       = errors.New("a commit message is one line, at most 200 characters")
	ErrNoBranch      = errors.New("this action needs a branch, and the checkout has none")
	ErrRemote        = errors.New("that remote name is not valid")
	ErrRemoteBranch  = errors.New("that is not a branch of a known remote")
	ErrSlug          = errors.New("a worktree folder is a single name, without slashes")
)

// refPattern is what git allows and a shell reads literally. git already
// forbids spaces, ~ ^ : ? * [ and backslash in a refname; this is stricter
// still, so nothing that reaches argv needs quoting or an escape.
var refPattern = regexp.MustCompile(`^[A-Za-z0-9._/@+-]+$`)

// namePattern is refPattern minus the leading dot and slash cases that make
// a poor new branch or directory name.
var namePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._/@+-]*$`)

const messageMax = 200

// validRef says whether s can go on a git command line as it stands. A
// leading dash would be read as a flag, "@{" is a revision expression, and
// ".." names a range — none of them is a target the graph points at.
func validRef(s string) bool {
	if s == "" || len(s) > 255 || !refPattern.MatchString(s) {
		return false
	}
	if strings.HasPrefix(s, "-") || strings.HasPrefix(s, ".") || strings.HasSuffix(s, ".lock") {
		return false
	}
	if strings.Contains(s, "..") || strings.Contains(s, "@{") || strings.HasSuffix(s, "/") {
		return false
	}
	return true
}

func validName(s string) bool {
	return validRef(s) && namePattern.MatchString(s)
}

// validSlug is a worktree folder: one path segment under .worktrees/, so the
// command can never name a directory outside it, nested or not.
func validSlug(s string) bool {
	return validName(s) && !strings.Contains(s, "/")
}

// slug reads the worktree folder name.
func slug(a Args) (string, error) {
	if !validSlug(a.Name) {
		return "", ErrSlug
	}
	return a.Name, nil
}

// worktreePath is where a worktree action points: absolute under the
// repository root when the caller knows it, relative to the terminal's
// folder when it does not. Quoted either way — a root may hold a space.
func worktreePath(a Args, name string) string {
	if a.RepoRoot == "" {
		return ".worktrees/" + name
	}
	return quote(strings.TrimRight(a.RepoRoot, "/") + "/.worktrees/" + name)
}

// remoteBranch splits a remote-tracking name into the remote it belongs to
// and the branch on that remote, using the caller's remote list so that a
// remote whose name contains a slash still resolves. The longest match wins.
func remoteBranch(a Args) (remote, branch string, err error) {
	t, err := target(a)
	if err != nil {
		return "", "", err
	}
	candidates := a.Remotes
	if len(candidates) == 0 {
		candidates = []string{remoteOf(a)}
	}
	for _, r := range candidates {
		if !validRef(r) {
			continue
		}
		if rest := strings.TrimPrefix(t, r+"/"); rest != t && rest != "" && len(r) > len(remote) {
			remote, branch = r, rest
		}
	}
	if remote == "" {
		return "", "", ErrRemoteBranch
	}
	return remote, branch, nil
}

// quote wraps a string for a POSIX shell: single quotes, with any single
// quote inside closed, escaped and reopened. Nothing else is interpreted.
func quote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// validMessage: one line, short enough to read in a prompt, no control
// characters. Quoting handles the rest.
func validMessage(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" || len(s) > messageMax {
		return false
	}
	for _, r := range s {
		if r < 0x20 || r == 0x7f {
			return false
		}
	}
	return true
}

type spec struct {
	tier Tier
	// needs names the Args fields this action cannot compose without, so a
	// form knows which inputs to show and a caller knows what to collect.
	needs []string
	// verb is the plain-words form for a toast and for the agent prompt:
	// "merge feat/x into main".
	verb func(a Args) string
	// build returns the command, or an error naming the bad field.
	build func(a Args) (string, error)
}

func remoteOf(a Args) string {
	if a.Remote == "" {
		return "origin"
	}
	return a.Remote
}

// pushCommand is the one command that changes shape with the checkout: a
// branch with no upstream publishes itself and sets one.
func pushCommand(a Args, force bool) (string, error) {
	flags := ""
	if force {
		// --force-with-lease, never a plain --force: the lease refuses when
		// the remote moved since the last fetch. It is void if anything
		// fetches behind the user's back, which is one reason PiCode adds no
		// background fetch on a timer.
		flags = " --force-with-lease"
	}
	if a.Upstream != "" {
		return "git push" + flags, nil
	}
	if a.Branch == "" {
		return "", ErrNoBranch
	}
	if !validRef(a.Branch) || !validRef(remoteOf(a)) {
		return "", ErrRemote
	}
	return fmt.Sprintf("git push%s -u %s %s", flags, remoteOf(a), a.Branch), nil
}

// target reads the pointed-at ref, refusing anything that would not survive
// a command line as itself.
func target(a Args) (string, error) {
	if !validRef(a.Target) {
		return "", ErrTarget
	}
	return a.Target, nil
}

func name(a Args) (string, error) {
	if !validName(a.Name) {
		return "", ErrName
	}
	return a.Name, nil
}

// one composes a command from a format and already-validated pieces.
func one(format string, parts ...any) (string, error) {
	return fmt.Sprintf(format, parts...), nil
}

var specs = map[string]spec{
	// --- Tier A: additive, local, recoverable ---------------------------
	"fetch": {TierA, nil, func(Args) string { return "fetch" }, func(Args) (string, error) {
		return "git fetch --prune", nil
	}},
	"fetch-into-local": {TierA, []string{"target", "name"}, func(a Args) string { return "fetch " + a.Target + " into " + a.Name }, func(a Args) (string, error) {
		n, err := name(a)
		if err != nil {
			return "", err
		}
		// origin/feat-x -> origin, feat-x: the shape `git fetch <remote>
		// <src>:<dst>` wants.
		r, src, err := remoteBranch(a)
		if err != nil {
			return "", err
		}
		return one("git fetch %s %s:%s", r, src, n)
	}},
	"checkout": {TierA, []string{"target"}, func(a Args) string { return "switch to " + a.Target }, func(a Args) (string, error) {
		t, err := target(a)
		if err != nil {
			return "", err
		}
		return one("git switch %s", t)
	}},
	"checkout-detach": {TierA, []string{"target"}, func(a Args) string { return "check out " + short(a.Target) + " detached" }, func(a Args) (string, error) {
		t, err := target(a)
		if err != nil {
			return "", err
		}
		return one("git switch --detach %s", t)
	}},
	"checkout-remote": {TierA, []string{"target", "name"}, func(a Args) string { return "check out " + a.Target + " as " + a.Name }, func(a Args) (string, error) {
		t, err := target(a)
		if err != nil {
			return "", err
		}
		n, err := name(a)
		if err != nil {
			return "", err
		}
		return one("git switch -c %s --track %s", n, t)
	}},
	"create-branch": {TierA, []string{"target", "name"}, func(a Args) string { return "create branch " + a.Name }, func(a Args) (string, error) {
		n, err := name(a)
		if err != nil {
			return "", err
		}
		t, err := target(a)
		if err != nil {
			return "", err
		}
		return one("git switch -c %s %s", n, t)
	}},
	"create-tag": {TierA, []string{"target", "name"}, func(a Args) string { return "tag " + short(a.Target) + " as " + a.Name }, func(a Args) (string, error) {
		n, err := name(a)
		if err != nil {
			return "", err
		}
		t, err := target(a)
		if err != nil {
			return "", err
		}
		return one("git tag %s %s", n, t)
	}},
	"create-worktree": {TierA, []string{"target", "name"}, func(a Args) string { return "create a worktree for " + short(a.Target) }, func(a Args) (string, error) {
		n, err := slug(a)
		if err != nil {
			return "", err
		}
		t, err := target(a)
		if err != nil {
			return "", err
		}
		// The repository's own convention (make worktree): a sibling
		// checkout under <root>/.worktrees/, named for the work — a sibling
		// of every other checkout, never nested inside the one the terminal
		// happens to sit in.
		return one("git worktree add %s %s", worktreePath(a, n), t)
	}},
	// create-worktree-branch is the same sibling checkout on a new branch
	// named for it (Fork agent…): a fork's work gets a branch of its own
	// from the commit the source stands on, never a detached checkout.
	"create-worktree-branch": {TierA, []string{"target", "name"}, func(a Args) string { return "create a worktree on a new branch from " + short(a.Target) }, func(a Args) (string, error) {
		n, err := slug(a)
		if err != nil {
			return "", err
		}
		t, err := target(a)
		if err != nil {
			return "", err
		}
		return one("git worktree add -b %s %s %s", n, worktreePath(a, n), t)
	}},
	"prune-worktrees": {TierA, nil, func(Args) string { return "prune stale worktrees" }, func(Args) (string, error) {
		return "git worktree prune", nil
	}},
	// restore-branch exists for undo (ADR-0096 phase 4): it puts a deleted
	// branch back where it was, without checking it out — which is what
	// distinguishes it from create-branch.
	"restore-branch": {TierA, []string{"target", "name"}, func(a Args) string { return "put " + a.Name + " back at " + short(a.Target) }, func(a Args) (string, error) {
		n, err := name(a)
		if err != nil {
			return "", err
		}
		t, err := target(a)
		if err != nil {
			return "", err
		}
		return one("git branch %s %s", n, t)
	}},

	// --- Tier B: moves HEAD or history, still recoverable ---------------
	"pull": {TierB, nil, func(Args) string { return "pull" }, func(Args) (string, error) {
		return "git pull --ff-only", nil
	}},
	"pull-remote": {TierB, []string{"target"}, func(a Args) string { return "pull " + a.Target }, func(a Args) (string, error) {
		r, src, err := remoteBranch(a)
		if err != nil {
			return "", err
		}
		return one("git pull --ff-only %s %s", r, src)
	}},
	"merge": {TierB, []string{"target"}, func(a Args) string { return "merge " + a.Target }, func(a Args) (string, error) {
		t, err := target(a)
		if err != nil {
			return "", err
		}
		return one("git merge --no-ff %s", t)
	}},
	"rebase": {TierB, []string{"target"}, func(a Args) string { return "rebase onto " + a.Target }, func(a Args) (string, error) {
		t, err := target(a)
		if err != nil {
			return "", err
		}
		return one("git rebase %s", t)
	}},
	"cherry-pick": {TierB, []string{"target"}, func(a Args) string { return "cherry-pick " + short(a.Target) }, func(a Args) (string, error) {
		t, err := target(a)
		if err != nil {
			return "", err
		}
		return one("git cherry-pick %s", t)
	}},
	"revert": {TierB, []string{"target"}, func(a Args) string { return "revert " + short(a.Target) }, func(a Args) (string, error) {
		t, err := target(a)
		if err != nil {
			return "", err
		}
		return one("git revert --no-edit %s", t)
	}},
	"reset-soft": {TierB, []string{"target"}, func(a Args) string { return "reset to " + short(a.Target) + ", keeping the changes staged" }, func(a Args) (string, error) {
		t, err := target(a)
		if err != nil {
			return "", err
		}
		return one("git reset --soft %s", t)
	}},
	// reset-keep exists for undo (ADR-0096 phase 4): it moves the branch back
	// and keeps local changes, and git refuses outright if a change would be
	// lost — which is why it, and not --hard, is the inverse of a merge or a
	// rebase that ran over a dirty tree.
	"reset-keep": {TierB, []string{"target"}, func(a Args) string { return "move back to " + short(a.Target) + ", keeping local changes" }, func(a Args) (string, error) {
		t, err := target(a)
		if err != nil {
			return "", err
		}
		return one("git reset --keep %s", t)
	}},
	"reset-mixed": {TierB, []string{"target"}, func(a Args) string { return "reset to " + short(a.Target) + ", keeping the changes" }, func(a Args) (string, error) {
		t, err := target(a)
		if err != nil {
			return "", err
		}
		return one("git reset --mixed %s", t)
	}},
	"rename-branch": {TierB, []string{"target", "name"}, func(a Args) string { return "rename " + a.Target + " to " + a.Name }, func(a Args) (string, error) {
		t, err := target(a)
		if err != nil {
			return "", err
		}
		n, err := name(a)
		if err != nil {
			return "", err
		}
		return one("git branch -m %s %s", t, n)
	}},
	"delete-branch": {TierB, []string{"target"}, func(a Args) string { return "delete the merged branch " + a.Target }, func(a Args) (string, error) {
		t, err := target(a)
		if err != nil {
			return "", err
		}
		// -d, not -D: git refuses a branch that is not merged, which is the
		// whole reason this one is tier B and its force twin is tier C.
		return one("git branch -d %s", t)
	}},
	"commit": {TierB, []string{"message"}, func(Args) string { return "commit" }, func(a Args) (string, error) {
		if !validMessage(a.Message) {
			return "", ErrMessage
		}
		return one("git add -A && git commit -m %s", quote(strings.TrimSpace(a.Message)))
	}},
	"pr": {TierB, nil, func(Args) string { return "open a pull request" }, func(Args) (string, error) {
		return "gh pr create --fill", nil
	}},

	// --- Tier C: publishes, or destroys work -----------------------------
	"push": {TierC, nil, func(Args) string { return "push" }, func(a Args) (string, error) {
		return pushCommand(a, false)
	}},
	"push-force": {TierC, nil, func(Args) string { return "force-push with lease" }, func(a Args) (string, error) {
		return pushCommand(a, true)
	}},
	"commit-push": {TierC, []string{"message"}, func(Args) string { return "commit and push" }, func(a Args) (string, error) {
		if !validMessage(a.Message) {
			return "", ErrMessage
		}
		push, err := pushCommand(a, false)
		if err != nil {
			return "", err
		}
		return one("git add -A && git commit -m %s && %s", quote(strings.TrimSpace(a.Message)), push)
	}},
	"push-tag": {TierC, []string{"target"}, func(a Args) string { return "push the tag " + a.Target }, func(a Args) (string, error) {
		t, err := target(a)
		if err != nil {
			return "", err
		}
		r := remoteOf(a)
		if !validRef(r) {
			return "", ErrRemote
		}
		return one("git push %s tag %s", r, t)
	}},
	"delete-branch-force": {TierC, []string{"target"}, func(a Args) string { return "delete the unmerged branch " + a.Target }, func(a Args) (string, error) {
		t, err := target(a)
		if err != nil {
			return "", err
		}
		return one("git branch -D %s", t)
	}},
	"delete-remote-branch": {TierC, []string{"target"}, func(a Args) string { return "delete " + a.Target + " on the remote" }, func(a Args) (string, error) {
		r, src, err := remoteBranch(a)
		if err != nil {
			return "", err
		}
		return one("git push %s --delete %s", r, src)
	}},
	"delete-tag": {TierC, []string{"target"}, func(a Args) string { return "delete the tag " + a.Target }, func(a Args) (string, error) {
		t, err := target(a)
		if err != nil {
			return "", err
		}
		return one("git tag -d %s", t)
	}},
	"reset-hard": {TierC, []string{"target"}, func(a Args) string { return "reset to " + short(a.Target) + ", discarding the changes" }, func(a Args) (string, error) {
		t, err := target(a)
		if err != nil {
			return "", err
		}
		return one("git reset --hard %s", t)
	}},
	"discard": {TierC, nil, func(Args) string { return "discard every uncommitted change" }, func(Args) (string, error) {
		return "git restore --source=HEAD --staged --worktree -- .", nil
	}},
	"clean": {TierC, nil, func(Args) string { return "delete every untracked file" }, func(Args) (string, error) {
		return "git clean -fd", nil
	}},
	"worktree-remove": {TierC, []string{"name"}, func(a Args) string { return "remove the worktree " + a.Name }, func(a Args) (string, error) {
		n, err := slug(a)
		if err != nil {
			return "", err
		}
		// git refuses a checkout with modified or untracked files, which is
		// what makes this the non-force twin.
		return one("git worktree remove %s", worktreePath(a, n))
	}},
	"worktree-remove-force": {TierC, []string{"name"}, func(a Args) string { return "remove the worktree " + a.Name + " and its uncommitted work" }, func(a Args) (string, error) {
		n, err := slug(a)
		if err != nil {
			return "", err
		}
		return one("git worktree remove --force %s", worktreePath(a, n))
	}},
}

func short(s string) string {
	if len(s) >= 40 && isHex(s) {
		return s[:7]
	}
	return s
}

func isHex(s string) bool {
	for _, r := range s {
		if !((r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F')) {
			return false
		}
	}
	return true
}

// Compose returns the exact command an action means, and its tier.
func Compose(action string, a Args) (string, Tier, error) {
	s, ok := specs[action]
	if !ok {
		return "", "", ErrUnknownAction
	}
	cmd, err := s.build(a)
	if err != nil {
		return "", s.tier, err
	}
	return cmd, s.tier, nil
}

// TierOf answers without composing, for a caller that only needs the gate.
func TierOf(action string) (Tier, bool) {
	s, ok := specs[action]
	if !ok {
		return "", false
	}
	return s.tier, true
}

// Verb is the action in plain words, for a toast and for the sentence an
// agent is asked. It never contains a command.
func Verb(action string, a Args) string {
	s, ok := specs[action]
	if !ok {
		return ""
	}
	return s.verb(a)
}

// Info is one action's contract with a caller: its risk tier and the fields
// it cannot compose without. The browser reads the catalog rather than
// carrying its own copy of the tiers — a client that believed a tier C action
// were tier B would skip the confirmation that tier exists for.
type Info struct {
	ID    string   `json:"id"`
	Tier  Tier     `json:"tier"`
	Needs []string `json:"needs"`
}

// Catalog is every action, sorted by id.
func Catalog() []Info {
	out := make([]Info, 0, len(specs))
	for _, id := range Actions() {
		s := specs[id]
		needs := s.needs
		if needs == nil {
			needs = []string{}
		}
		out = append(out, Info{ID: id, Tier: s.tier, Needs: needs})
	}
	return out
}

// Actions lists every action this package knows, sorted, for tests and for
// the contract the browser is checked against.
func Actions() []string {
	out := make([]string, 0, len(specs))
	for id := range specs {
		out = append(out, id)
	}
	sortStrings(out)
	return out
}

func sortStrings(s []string) {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j] < s[j-1]; j-- {
			s[j], s[j-1] = s[j-1], s[j]
		}
	}
}

// --- What an agent is asked (ADR-0078 stage 3, ADR-0096) -------------------

// Prompt is the sentence an agent receives instead of a command: the folder,
// the action and the rules that matter, in the plain words the human reads in
// the form's preview. Where a command would be blind, it asks for judgment —
// an agent that hits a conflict should stop and say why, not improvise.
//
// It contains no command. The agent runs git in its own turn, under its own
// rules, and writing the command for it would only hide that.
func Prompt(action string, a Args, root string) string {
	if _, ok := specs[action]; !ok {
		return ""
	}
	where := "In this folder"
	if root != "" {
		where = "In " + root
	}
	verb := Verb(action, a)
	switch action {
	case "fetch":
		return where + ", fetch with --prune and tell me how the branch stands against its upstream."
	case "pull", "pull-remote":
		return where + ", " + verb + " by fast-forward only. If it cannot fast-forward, stop and tell me why instead of merging or rebasing."
	case "merge", "rebase", "cherry-pick":
		return where + ", " + verb + ". If it conflicts, stop and tell me which files — do not resolve it your way without asking."
	case "revert":
		return where + ", " + verb + " without editing the message, and tell me what it undid."
	case "commit":
		return where + ", commit the current changes" + messageClause(a) + " Do not push."
	case "commit-push":
		return where + ", commit the current changes" + messageClause(a) + " Then push, never with force."
	case "push":
		return where + ", push this branch, setting its upstream if it has none. Never force-push; if the push is rejected, tell me why."
	case "push-force":
		return where + ", push this branch with --force-with-lease — never a plain --force. If the lease refuses, stop and tell me what moved on the remote."
	case "reset-hard", "discard", "clean", "delete-branch-force", "delete-remote-branch", "worktree-remove-force":
		return where + ", " + verb + ". This destroys work that is not committed or not merged, so tell me exactly what will be lost before you do it."
	case "pr":
		return where + ", create a pull request with gh, writing the title and body from the commits, and give me its URL."
	default:
		return where + ", " + verb + "."
	}
}

func messageClause(a Args) string {
	if m := strings.TrimSpace(a.Message); m != "" {
		return ` with this message: "` + m + `".`
	}
	return " with a message you write from the changes."
}

// WorktreeCreate reports whether text is exactly the command
// create-worktree-branch composes — `git worktree add -b <slug>
// <root>/.worktrees/<slug> <ref>`, with nothing before, after or between —
// and returns the repository root it names ("" for the relative form). The
// run door lets this one command past the busy-repository interlock
// (ADR-0202): it makes a new folder and a new branch and touches no file
// another agent is working on. A root holding a quote character is never
// recognized, so an escaped quote can never hide a second command.
func WorktreeCreate(text string) (root string, ok bool) {
	rest, found := strings.CutPrefix(text, "git worktree add -b ")
	if !found {
		return "", false
	}
	slugPart, rest, found := strings.Cut(rest, " ")
	if !found || !validSlug(slugPart) {
		return "", false
	}
	var path string
	if strings.HasPrefix(rest, "'") {
		end := strings.Index(rest[1:], "'")
		if end < 0 {
			return "", false
		}
		path, rest = rest[1:1+end], rest[2+end:]
	} else {
		path, rest, found = strings.Cut(rest, " ")
		if !found {
			return "", false
		}
		rest = " " + rest
	}
	ref, found := strings.CutPrefix(rest, " ")
	if !found || !validRef(ref) || strings.ContainsAny(ref, " \t") {
		return "", false
	}
	if strings.ContainsAny(path, "\x00\r\n") {
		return "", false
	}
	suffix := ".worktrees/" + slugPart
	switch {
	case path == suffix:
		return "", true
	case strings.HasSuffix(path, "/"+suffix):
		root = strings.TrimSuffix(path, "/"+suffix)
		if root == "" || !strings.HasPrefix(root, "/") {
			return "", false
		}
		return root, true
	}
	return "", false
}
