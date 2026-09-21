// Package delivery observes Git and producer receipts without executing project scripts.
package delivery

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"time"
)

var oid = regexp.MustCompile(`^(?:[a-f0-9]{40}|[a-f0-9]{64})$`)
var receiptID = regexp.MustCompile(`^[a-f0-9-]{36}$`)

type Scope struct {
	Tree    string   `json:"tree"`
	Covered string   `json:"covered"`
	Roots   []string `json:"roots"`
}
type Receipt struct {
	SchemaVersion int    `json:"schemaVersion"`
	ID            string `json:"id"`
	Kind          string `json:"kind"`
	RepositoryKey string `json:"repositoryKey"`
	SourceRef     string `json:"sourceRef"`
	Source        string `json:"source"`
	Tree          string `json:"tree"`
	TargetBefore  string `json:"targetBefore"`
	TargetAfter   string `json:"targetAfter"`
	StartedAt     string `json:"startedAt"`
	FinishedAt    string `json:"finishedAt"`
	Outcome       string `json:"outcome"`
	Clean         bool   `json:"clean"`
	Scope         *Scope `json:"scope,omitempty"`
}
type Evidence struct {
	ID      string `json:"id"`
	Kind    string `json:"kind"`
	Outcome string `json:"outcome"`
	At      string `json:"at"`
	Source  string `json:"source"`
}
type Declaration struct{ ID, Title, Branch, Revision, Review, Target string }
type Change struct {
	ID           string     `json:"id"`
	Title        string     `json:"title"`
	Branch       string     `json:"branch"`
	Revision     string     `json:"revision"`
	Registered   bool       `json:"registered"`
	Review       string     `json:"review"`
	SourceStatus string     `json:"sourceStatus"`
	Integration  string     `json:"integration"`
	Validation   string     `json:"validation"`
	Publication  string     `json:"publication"`
	Checkout     string     `json:"checkout"`
	Evidence     []Evidence `json:"evidence"`
	Agents       []string   `json:"agents"`
	Worktree     string     `json:"-"`
}
type Snapshot struct {
	SchemaVersion int           `json:"schemaVersion"`
	RepositoryKey string        `json:"repositoryKey"`
	Root          string        `json:"root"`
	ObservedAt    string        `json:"observedAt"`
	Target        string        `json:"target"`
	TargetOID     string        `json:"targetOid"`
	Targets       []string      `json:"targets"`
	Changes       []Change      `json:"changes"`
	Environments  []Environment `json:"environments"`
	Issues        []string      `json:"issues"`
	Complete      bool          `json:"complete"`
}
type cappedBuffer struct {
	bytes.Buffer
	exceeded bool
}

func (b *cappedBuffer) Write(p []byte) (int, error) {
	if b.Len()+len(p) > 2<<20 {
		b.exceeded = true
		return 0, errors.New("Git output limit")
	}
	return b.Buffer.Write(p)
}

var git = runGit

func runGit(ctx context.Context, cwd string, args ...string) (string, error) {
	c := exec.CommandContext(ctx, "git", append([]string{"--no-optional-locks", "-c", "core.fsmonitor=false"}, args...)...)
	c.Dir = cwd
	var out cappedBuffer
	c.Stdout = &out
	err := c.Run()
	return strings.TrimSpace(out.String()), err
}
func rawGit(ctx context.Context, cwd string, args ...string) ([]byte, error) {
	c := exec.CommandContext(ctx, "git", append([]string{"--no-optional-locks", "-c", "core.fsmonitor=false"}, args...)...)
	c.Dir = cwd
	var out cappedBuffer
	c.Stdout = &out
	err := c.Run()
	return out.Bytes(), err
}
func resolve(ctx context.Context, cwd, ref string) string {
	v, e := git(ctx, cwd, "rev-parse", "--verify", ref)
	if e != nil || !oid.MatchString(v) {
		return ""
	}
	return v
}
func ancestor(ctx context.Context, cwd, a, b string) (bool, error) {
	_, err := git(ctx, cwd, "merge-base", "--is-ancestor", a, b)
	if err == nil {
		return true, nil
	}
	var e *exec.ExitError
	if errors.As(err, &e) && e.ExitCode() == 1 {
		return false, nil
	}
	return false, err
}

// ReadReceipts confines bounded reads to the common Git evidence directory. Invalid
// records never become green. Raw repository paths and scope roots stay server-side.
func ReadReceipts(repo string) ([]Receipt, []string) {
	out := []Receipt{}
	issues := []string{}
	root, e := os.OpenRoot(repo)
	if e != nil {
		return out, []string{"Evidence unavailable"}
	}
	defer root.Close()
	info, e := root.Lstat("picode-delivery")
	if os.IsNotExist(e) {
		return out, issues
	}
	if e != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 || (runtime.GOOS != "windows" && info.Mode().Perm()&0077 != 0) {
		return out, []string{"Evidence directory is unsafe"}
	}
	dir, e := root.OpenRoot("picode-delivery")
	if e != nil {
		return out, []string{"Evidence unavailable"}
	}
	defer dir.Close()
	actual, e := dir.Stat(".")
	if e != nil || !os.SameFile(info, actual) {
		return out, []string{"Evidence directory changed"}
	}
	f, e := dir.Open(".")
	if e != nil {
		return out, []string{"Evidence unavailable"}
	}
	defer f.Close()
	entries, e := f.ReadDir(1001)
	if e != nil && e != io.EOF {
		return out, []string{"Evidence unavailable"}
	}
	if len(entries) > 1000 {
		issues = append(issues, "Evidence limit reached; history is incomplete")
		entries = entries[:1000]
	}
	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		st, e := dir.Lstat(entry.Name())
		if e != nil || !st.Mode().IsRegular() || st.Size() > 65536 || (runtime.GOOS != "windows" && st.Mode().Perm()&0077 != 0) {
			issues = append(issues, "Unreadable evidence record")
			continue
		}
		file, e := dir.Open(entry.Name())
		if e != nil {
			issues = append(issues, "Unreadable evidence record")
			continue
		}
		now, e := file.Stat()
		if e != nil || !os.SameFile(st, now) {
			file.Close()
			issues = append(issues, "Evidence record changed")
			continue
		}
		raw, e := io.ReadAll(io.LimitReader(file, 65537))
		file.Close()
		var r Receipt
		if e != nil || len(raw) > 65536 || json.Unmarshal(raw, &r) != nil || !validReceipt(r, repo, entry.Name()) {
			issues = append(issues, "Unrecognized evidence record")
			continue
		}
		if r.Outcome == "started" {
			r.Outcome = "unknown"
		}
		out = append(out, r)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].StartedAt > out[j].StartedAt })
	return out, unique(issues)
}
func validReceipt(r Receipt, repo, name string) bool {
	if r.SchemaVersion != 1 || r.RepositoryKey != repo || !receiptID.MatchString(r.ID) || name != r.ID+".json" || !oid.MatchString(r.Source) || !oid.MatchString(r.Tree) {
		return false
	}
	if r.Kind != "scoped" && r.Kind != "full-ci" && r.Kind != "land" {
		return false
	}
	if r.Outcome != "started" && r.Outcome != "passed" && r.Outcome != "failed" {
		return false
	}
	start, e := time.Parse(time.RFC3339Nano, r.StartedAt)
	if e != nil || start.After(time.Now().Add(time.Minute)) {
		return false
	}
	if r.Outcome != "started" {
		end, e := time.Parse(time.RFC3339Nano, r.FinishedAt)
		if e != nil || end.Before(start) || end.After(time.Now().Add(time.Minute)) {
			return false
		}
	}
	return len(r.SourceRef) <= 512 && (r.Scope == nil || len(r.Scope.Roots) <= 4096)
}

// Reuse mirrors ADR-0124's decideReuse. Covered-only reuse is not full main CI.
func Reuse(s *Scope, tree, covered string, dirty bool) bool {
	return s != nil && !dirty && (s.Tree == tree || (s.Covered != "" && s.Covered == covered))
}
func covered(ctx context.Context, cwd, revision string, roots []string) string {
	if len(roots) == 0 {
		return ""
	}
	for _, p := range roots {
		if p == "" || len(p) > 4096 || filepath.IsAbs(p) || strings.Contains(p, "..") || strings.ContainsAny(p, "\x00\r\n") || strings.HasPrefix(p, ":") {
			return ""
		}
	}
	args := append([]string{"ls-tree", "-r", revision, "--"}, roots...)
	b, e := rawGit(ctx, cwd, args...)
	if e != nil {
		return ""
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}
func unique(in []string) []string {
	out := []string{}
	seen := map[string]bool{}
	for _, s := range in {
		if !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	return out
}

// Observe retries one changing Git snapshot and makes every uncertainty explicit.
func Observe(parent context.Context, cwd, repo, target string, decl []Declaration) Snapshot {
	ctx, cancel := context.WithTimeout(parent, 10*time.Second)
	defer cancel()
	var s Snapshot
	for attempt := 0; attempt < 2; attempt++ {
		s = collect(ctx, cwd, repo, target, decl)
		unstable := false
		for _, issue := range s.Issues {
			if issue == "Project changed during observation" {
				unstable = true
			}
		}
		if !unstable {
			break
		}
	}
	return s
}
func collect(ctx context.Context, cwd, repo, target string, decl []Declaration) Snapshot {
	s := Snapshot{SchemaVersion: 1, RepositoryKey: repo, Root: cwd, ObservedAt: time.Now().UTC().Format(time.RFC3339Nano), Target: target, Targets: []string{}, Changes: []Change{}, Environments: []Environment{}, Issues: []string{}}
	refsText, err := git(ctx, cwd, "for-each-ref", "--format=%(refname:short) %(objectname)", "refs/heads/")
	if err != nil {
		s.Issues = append(s.Issues, "Repository unavailable")
		return s
	}
	refs := map[string]string{}
	for _, line := range strings.Split(refsText, "\n") {
		parts := strings.Fields(line)
		if len(parts) == 2 && oid.MatchString(parts[1]) {
			refs[parts[0]] = parts[1]
			s.Targets = append(s.Targets, parts[0])
		}
	}
	if target == "" {
		if _, ok := refs["main"]; ok {
			target = "main"
			s.Target = target
		}
	}
	s.TargetOID = refs[target]
	if s.TargetOID == "" {
		s.Issues = append(s.Issues, "Choose the branch changes will join")
		return s
	}
	records, issues := ReadReceipts(repo)
	s.Issues = append(s.Issues, issues...)
	worktrees := map[string]string{}
	dirty := map[string]string{}
	wt, err := git(ctx, cwd, "worktree", "list", "--porcelain", "-z")
	if err != nil {
		s.Issues = append(s.Issues, "Checkout inventory unavailable")
	} else {
		path := ""
		for _, v := range strings.Split(wt, "\x00") {
			if strings.HasPrefix(v, "worktree ") {
				path = strings.TrimPrefix(v, "worktree ")
			}
			if strings.HasPrefix(v, "branch refs/heads/") {
				worktrees[strings.TrimPrefix(v, "branch refs/heads/")] = path
			}
		}
	}
	checked := 0
	for _, branch := range s.Targets {
		if path := worktrees[branch]; path != "" {
			dirty[branch] = "unknown"
			if checked >= 32 {
				s.Issues = append(s.Issues, "Some checkout statuses were omitted")
				continue
			}
			checked++
			status, e := git(ctx, path, "status", "--porcelain")
			if e != nil {
				s.Issues = append(s.Issues, "Checkout status unavailable")
			} else if status != "" {
				dirty[branch] = "dirty"
			} else {
				dirty[branch] = "clean"
			}
		}
	}
	seen := map[string]bool{}
	for _, d := range decl {
		c := Change{ID: d.ID, Title: d.Title, Branch: d.Branch, Revision: d.Revision, Registered: true, Review: d.Review}
		if d.Target != "" && d.Target != target && c.Review == "requested" {
			c.Review = "other-target"
		}
		s.Changes = append(s.Changes, c)
		seen[d.Branch] = true
	}
	for _, br := range s.Targets {
		if br != target && !seen[br] {
			s.Changes = append(s.Changes, Change{ID: "ref:" + br, Title: br, Branch: br, Revision: refs[br], Review: "unknown"})
		}
	}
	for _, r := range records {
		if r.Kind == "land" && refs[r.SourceRef] == "" && !seen[r.SourceRef] {
			seen[r.SourceRef] = true
			s.Changes = append(s.Changes, Change{ID: "receipt:" + r.ID, Title: r.SourceRef, Branch: r.SourceRef, Revision: r.Source, Review: "unknown"})
		}
	}
	if len(s.Changes) > 1000 {
		s.Changes = s.Changes[:1000]
		s.Issues = append(s.Issues, "Change limit reached; history is incomplete")
	}
	targetCheck := latest(records, s.TargetOID, "full-ci")
	for i := range s.Changes {
		c := &s.Changes[i]
		c.Agents = []string{}
		c.Evidence = []Evidence{}
		c.SourceStatus = "current"
		c.Integration = "unknown"
		c.Validation = "unknown"
		c.Publication = "unknown"
		c.Checkout = dirty[c.Branch]
		c.Worktree = worktrees[c.Branch]
		if c.Checkout == "" {
			c.Checkout = "not-present"
		}
		if refs[c.Branch] == "" {
			c.SourceStatus = "removed"
		} else if refs[c.Branch] != c.Revision {
			c.SourceStatus = "changed"
		}
		if ctx.Err() != nil {
			s.Issues = append(s.Issues, "Observation timed out")
			continue
		}
		if !oid.MatchString(c.Revision) {
			s.Issues = append(s.Issues, "Change revision unavailable")
			continue
		}
		included, e := ancestor(ctx, cwd, c.Revision, s.TargetOID)
		if e != nil {
			s.Issues = append(s.Issues, "Change history unavailable")
			continue
		}
		if included {
			c.Integration = "integrated"
		} else {
			ff, e := ancestor(ctx, cwd, s.TargetOID, c.Revision)
			if e != nil {
				s.Issues = append(s.Issues, "Change history unavailable")
			} else if ff {
				c.Integration = "not-integrated"
			} else {
				c.Integration = "update-needed"
			}
		}
		var chosen *Receipt
		if included {
			chosen = targetCheck
		} else {
			chosen = latest(records, c.Revision, "")
		}
		if chosen == nil && !included {
			// Covered-only reuse is scoped evidence, never an overall Ready claim.
			tree := resolve(ctx, cwd, c.Revision+"^{tree}")
			for j := range records {
				r := &records[j]
				if r.Kind != "scoped" || r.SourceRef != c.Branch || r.Outcome != "passed" || !r.Clean {
					continue
				}
				hash := ""
				if r.Scope != nil && r.Scope.Tree != tree {
					hash = covered(ctx, cwd, c.Revision, r.Scope.Roots)
				}
				if Reuse(r.Scope, tree, hash, c.Checkout != "clean") {
					chosen = r
					c.Validation = "scope-reusable"
					break
				}
				chosen = r
				c.Validation = "needs-recheck"
				break
			}
		}
		if chosen != nil {
			r := chosen
			c.Evidence = append(c.Evidence, Evidence{ID: r.ID, Kind: r.Kind, Outcome: r.Outcome, At: r.FinishedAt, Source: r.Source})
			if c.Validation == "unknown" {
				if r.Outcome == "failed" && r.Clean {
					c.Validation = "failed"
				} else if r.Outcome == "passed" && r.Clean {
					if r.Kind == "full-ci" {
						c.Validation = "passed"
					} else {
						c.Validation = "scoped-passed"
					}
				}
			}
			if !included && (c.Checkout == "dirty" || c.SourceStatus == "changed" || r.TargetBefore != s.TargetOID) {
				c.Validation = "needs-recheck"
			}
		}
	}
	after, e := git(ctx, cwd, "for-each-ref", "--format=%(refname:short) %(objectname)", "refs/heads/")
	if e != nil {
		s.Issues = append(s.Issues, "Could not verify current branches")
	} else if after != refsText {
		s.Issues = append(s.Issues, "Project changed during observation")
		for i := range s.Changes {
			s.Changes[i].Integration = "unknown"
			s.Changes[i].Validation = "unknown"
		}
	}
	s.Issues = unique(s.Issues)
	s.Complete = len(s.Issues) == 0
	return s
}
func latest(records []Receipt, revision, kind string) *Receipt {
	for i := range records {
		r := &records[i]
		if r.Source == revision && r.Kind != "land" && (kind == "" || r.Kind == kind) {
			return r
		}
	}
	return nil
}
