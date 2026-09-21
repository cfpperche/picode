package delivery

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"runtime"
	"sort"
	"strings"
	"time"
)

// D2 publication observation (ADR-0170). A deployment receipt is producer
// evidence, not a command and not an authorization: it lives under the PiCode
// data directory, and an invalid record never becomes a green fact.
//
// The running revision is the environment's own identity, so the two are
// compared directly: a change is published when the artifact that answers
// contains its exact revision. Semver, a shortened hash, a dirty build or a
// revision this repository does not have can never claim publication.

// Deployment is one deployment attempt as the deploying process recorded it:
// <data>/var/delivery/<uuid>.json (dir 0700, file 0600).
type Deployment struct {
	SchemaVersion     int    `json:"schemaVersion"`
	ID                string `json:"id"`
	Kind              string `json:"kind"`
	RepositoryKey     string `json:"repositoryKey"`
	Actor             string `json:"actor,omitempty"`
	StartedAt         string `json:"startedAt"`
	FinishedAt        string `json:"finishedAt,omitempty"`
	Outcome           string `json:"outcome"`
	RequestedRevision string `json:"requestedRevision"`
	BuiltRevision     string `json:"builtRevision"`
	RevisionBefore    string `json:"revisionBefore"`
	ObservedRevision  string `json:"observedRevision"`
	BootBefore        string `json:"bootBefore,omitempty"`
	BootAfter         string `json:"bootAfter,omitempty"`
	Clean             bool   `json:"clean"`
	Error             string `json:"error,omitempty"`
}

// Running is the observed instance's identity at one instant. It is sampled
// before and after a read, because a daemon that restarted mid-read cannot
// describe one environment (O19).
type Running struct {
	Display    string
	Revision   string
	Boot       string
	Responding bool
}

// Binding is the store's local-observer configuration for one workspace,
// already resolved to the repository key it was recorded against.
type Binding struct {
	WorkspaceID string
	Repo        string
	Observer    string
}

// Attempt is the newest recorded deployment of this repository.
type Attempt struct {
	ID                string `json:"id"`
	Outcome           string `json:"outcome"`
	At                string `json:"at"`
	RequestedRevision string `json:"requestedRevision"`
	BuiltRevision     string `json:"builtRevision"`
	RevisionBefore    string `json:"revisionBefore"`
	ObservedRevision  string `json:"observedRevision"`
	Error             string `json:"error,omitempty"`
}

// Published is the set of observed changes the running artifact does not
// contain. Changes are listed as change IDs; two changes at one source
// revision are one commit and are counted once (O26).
type Published struct {
	Count   int      `json:"count"`
	Changes []string `json:"changes"`
}

// BusyObservation reports the deploy guard's own answer. Unknown coverage is
// never turned into a safety claim (O25).
type BusyObservation struct {
	Status string `json:"status"`
	Owners int    `json:"owners"`
}

// Environment is one observed deployment target. The pilot has exactly one
// kind, picode-self: the instance answering this read.
type Environment struct {
	ID             string          `json:"id"`
	Kind           string          `json:"kind"`
	Status         string          `json:"status"`
	ReasonCode     string          `json:"reasonCode,omitempty"`
	RepositoryKey  string          `json:"repositoryKey,omitempty"`
	DisplayVersion string          `json:"displayVersion,omitempty"`
	Revision       string          `json:"revision,omitempty"`
	Boot           string          `json:"boot,omitempty"`
	Responding     *bool           `json:"responding"`
	Artifact       string          `json:"artifact"`
	ObservedAt     string          `json:"observedAt"`
	Unpublished    Published       `json:"unpublished"`
	LastAttempt    *Attempt        `json:"lastAttempt"`
	Busy           BusyObservation `json:"busy"`
	Issues         []string        `json:"issues"`
}

// EnvironmentInput is everything the environment read needs, resolved by the
// caller from the same snapshot as the changes.
type EnvironmentInput struct {
	DataDir   string
	Cwd       string
	Repo      string
	TargetOID string
	Binding   *Binding
	Conflict  bool
	Busy      *int
	Sample    func() Running
}

// ReadDeployments confines a bounded read to <data>/var/delivery. Unsafe or
// unreadable records are reported as issues, never silently dropped, and a
// record whose outcome never finished is unknown rather than running or
// failed (O21).
func ReadDeployments(dataDir string) ([]Deployment, []string) {
	out := []Deployment{}
	issues := []string{}
	if dataDir == "" {
		return out, []string{"Deployment evidence unavailable"}
	}
	root, err := os.OpenRoot(dataDir)
	if err != nil {
		return out, []string{"Deployment evidence unavailable"}
	}
	defer root.Close()
	dir, issue := evidenceDir(root, "var", "delivery")
	if issue != "" {
		return out, []string{issue}
	}
	if dir == nil {
		return out, issues
	}
	defer dir.Close()
	f, err := dir.Open(".")
	if err != nil {
		return out, []string{"Deployment evidence unavailable"}
	}
	defer f.Close()
	entries, err := f.ReadDir(1001)
	if err != nil && err != io.EOF {
		return out, []string{"Deployment evidence unavailable"}
	}
	if len(entries) > 1000 {
		issues = append(issues, "Deployment evidence limit reached; history is incomplete")
		entries = entries[:1000]
	}
	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		st, err := dir.Lstat(entry.Name())
		if err != nil || !st.Mode().IsRegular() || st.Size() > 65536 || (runtime.GOOS != "windows" && st.Mode().Perm()&0077 != 0) {
			issues = append(issues, "Unreadable deployment record")
			continue
		}
		file, err := dir.Open(entry.Name())
		if err != nil {
			issues = append(issues, "Unreadable deployment record")
			continue
		}
		now, err := file.Stat()
		if err != nil || !os.SameFile(st, now) {
			file.Close()
			issues = append(issues, "Deployment record changed")
			continue
		}
		raw, err := io.ReadAll(io.LimitReader(file, 65537))
		file.Close()
		var d Deployment
		if err != nil || len(raw) > 65536 || json.Unmarshal(raw, &d) != nil || !validDeployment(d, entry.Name()) {
			issues = append(issues, "Unrecognized deployment record")
			continue
		}
		if d.Outcome == "started" {
			d.Outcome = "unknown"
		}
		out = append(out, d)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].StartedAt != out[j].StartedAt {
			return out[i].StartedAt > out[j].StartedAt
		}
		return out[i].ID < out[j].ID
	})
	return out, unique(issues)
}

// evidenceDir walks a managed evidence path without following a symlink at
// any level, and requires owner-only permissions on unix. A missing directory
// is (nil, "") — no evidence, not an error.
func evidenceDir(root *os.Root, parts ...string) (*os.Root, string) {
	cur := root
	for _, part := range parts {
		info, err := cur.Lstat(part)
		if os.IsNotExist(err) {
			return nil, ""
		}
		if err != nil {
			return nil, "Deployment evidence unavailable"
		}
		if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 || (runtime.GOOS != "windows" && info.Mode().Perm()&0077 != 0) {
			return nil, "Deployment evidence directory is unsafe"
		}
		next, err := cur.OpenRoot(part)
		if err != nil {
			return nil, "Deployment evidence unavailable"
		}
		actual, err := next.Stat(".")
		if err != nil || !os.SameFile(info, actual) {
			next.Close()
			return nil, "Deployment evidence directory changed"
		}
		cur = next
	}
	return cur, ""
}

func validDeployment(d Deployment, name string) bool {
	if d.SchemaVersion != 1 || d.Kind != "deploy" || !receiptID.MatchString(d.ID) || name != d.ID+".json" {
		return false
	}
	if d.RepositoryKey == "" || len(d.RepositoryKey) > 4096 || hasControl(d.RepositoryKey) {
		return false
	}
	if d.Outcome != "started" && d.Outcome != "passed" && d.Outcome != "failed" {
		return false
	}
	start, err := time.Parse(time.RFC3339Nano, d.StartedAt)
	if err != nil || start.After(time.Now().Add(time.Minute)) {
		return false
	}
	if d.Outcome != "started" {
		end, err := time.Parse(time.RFC3339Nano, d.FinishedAt)
		if err != nil || end.Before(start) || end.After(time.Now().Add(time.Minute)) {
			return false
		}
	}
	for _, rev := range []string{d.RequestedRevision, d.BuiltRevision, d.RevisionBefore, d.ObservedRevision} {
		if rev != "" && !oid.MatchString(rev) {
			return false
		}
	}
	if len(d.Error) > 512 || hasControl(d.Error) || len(d.Actor) > 128 || hasControl(d.Actor) {
		return false
	}
	return len(d.BootBefore) <= 128 && len(d.BootAfter) <= 128
}

func hasControl(s string) bool {
	for _, r := range s {
		if r < 0x20 || r == 0x7f {
			return true
		}
	}
	return false
}

// ObserveEnvironment composes the environment facts and marks every change
// with its publication state. It never invents a value: a fact it could not
// establish is unknown with a reason code, and the caller renders the copy.
func ObserveEnvironment(ctx context.Context, in EnvironmentInput, changes []Change) (Environment, []Change) {
	env := Environment{ID: "picode-self", Kind: "picode-self", Status: "unknown", Artifact: "unknown",
		ObservedAt: time.Now().UTC().Format(time.RFC3339Nano), Unpublished: Published{Changes: []string{}},
		Busy: BusyObservation{Status: "unknown"}, Issues: []string{}}
	if in.Busy != nil {
		env.Busy = BusyObservation{Status: "known", Owners: *in.Busy}
	}
	if in.Binding == nil {
		env.Status, env.ReasonCode = "unconfigured", "no-observer"
		return env, markPublication(changes, "unknown")
	}
	env.RepositoryKey = in.Binding.Repo
	// A project bound to *something* always carries the instance's own
	// identity, even when the connection turns out to be unusable: which
	// version is running is a fact about this machine, not about the binding,
	// and it is what tells a reader whether the mismatch matters.
	first := in.Sample()
	env.Revision, env.Boot, env.DisplayVersion = first.Revision, first.Boot, first.Display
	responding := first.Responding
	env.Responding = &responding
	if in.Conflict {
		env.Status, env.ReasonCode = "conflict", "binding-conflict"
		return env, markPublication(changes, "unknown")
	}
	if in.Binding.Observer != "picode-self" {
		env.Status, env.ReasonCode = "unknown", "unsupported-observer"
		return env, markPublication(changes, "unknown")
	}
	if in.Binding.Repo != in.Repo {
		env.Status, env.ReasonCode = "unknown", "repository-mismatch"
		return env, markPublication(changes, "unknown")
	}
	records, issues := ReadDeployments(in.DataDir)
	env.Issues = append(env.Issues, issues...)
	after := in.Sample()
	if first.Boot != after.Boot || first.Revision != after.Revision {
		// One retry: a restart during the read is real, and the retry usually
		// lands inside one boot. A second crossing stays unknown (O19).
		first = in.Sample()
		records, issues = ReadDeployments(in.DataDir)
		env.Issues = append(env.Issues, issues...)
		after = in.Sample()
	}
	env.Revision, env.Boot, env.DisplayVersion = after.Revision, after.Boot, after.Display
	responding = after.Responding
	env.Responding = &responding
	if first.Boot != after.Boot || first.Revision != after.Revision {
		env.Status, env.ReasonCode = "unknown", "boot-changed"
		return env, markPublication(changes, "unknown")
	}
	env.LastAttempt = latestAttempt(records, in.Repo)
	env.Artifact = artifactState(records, after.Revision)
	if env.Artifact == "unknown" && after.Revision != "" {
		// No receipt for this artifact: the running revision is evidence of
		// itself only after it maps to this repository's history.
		env.Issues = append(env.Issues, "No deployment record matches the running revision")
	}
	if after.Revision == "" {
		env.Status, env.ReasonCode = "unknown", "revision-unknown"
		return env, markPublication(changes, "unknown")
	}
	if resolve(ctx, in.Cwd, after.Revision+"^{commit}") == "" {
		env.Status, env.ReasonCode = "unknown", "revision-unmapped"
		return env, markPublication(changes, "unknown")
	}
	if env.Artifact == "dirty" {
		env.Status, env.ReasonCode = "unknown", "artifact-dirty"
		return env, markPublication(changes, "unknown")
	}
	if in.TargetOID == "" {
		// Without a target there is nothing to be published into; the copy is
		// the integration lane's own "Choose the branch changes will join."
		env.Status, env.ReasonCode = "unknown", "target-required"
		return env, markPublication(changes, "unknown")
	}
	inRunning, err := ancestor(ctx, in.Cwd, after.Revision, in.TargetOID)
	if err != nil {
		env.Status, env.ReasonCode = "unknown", "history-unavailable"
		return env, markPublication(changes, "unknown")
	}
	targetInRunning, err := ancestor(ctx, in.Cwd, in.TargetOID, after.Revision)
	if err != nil {
		env.Status, env.ReasonCode = "unknown", "history-unavailable"
		return env, markPublication(changes, "unknown")
	}
	if !inRunning && !targetInRunning {
		// Neither contains the other: the artifact is not a revision of this
		// target's line, so no membership claim is safe (O17).
		env.Status, env.ReasonCode = "unknown", "divergent-history"
		return env, markPublication(changes, "unknown")
	}
	env.Status = "known"
	env.ReasonCode = ""
	seen := map[string]bool{}
	for i := range changes {
		c := &changes[i]
		if c.Integration != "integrated" {
			c.Publication = "unknown"
			continue
		}
		published, err := ancestor(ctx, in.Cwd, c.Revision, after.Revision)
		if err != nil {
			c.Publication = "unknown"
			env.Status, env.ReasonCode = "unknown", "history-unavailable"
			continue
		}
		if published {
			c.Publication = "published"
			continue
		}
		c.Publication = "not-published"
		if !seen[c.Revision] {
			seen[c.Revision] = true
			env.Unpublished.Changes = append(env.Unpublished.Changes, c.ID)
		}
	}
	env.Unpublished.Count = len(env.Unpublished.Changes)
	if ctx.Err() != nil {
		env.Status, env.ReasonCode = "unknown", "observation-timeout"
	}
	if env.Status == "unknown" {
		// A failed membership read invalidates the set it was building.
		for i := range changes {
			changes[i].Publication = "unknown"
		}
		env.Unpublished = Published{Changes: []string{}}
	}
	return env, changes
}

func markPublication(changes []Change, state string) []Change {
	for i := range changes {
		changes[i].Publication = state
	}
	return changes
}

// artifactState is clean only when a passing record for this exact revision
// says the deploying checkout was clean. Everything else is unknown.
func artifactState(records []Deployment, revision string) string {
	if revision == "" {
		return "unknown"
	}
	for i := range records {
		r := &records[i]
		if r.BuiltRevision != revision || r.ObservedRevision != revision {
			continue
		}
		if r.Outcome == "passed" {
			if r.Clean {
				return "clean"
			}
			return "dirty"
		}
	}
	return "unknown"
}

func latestAttempt(records []Deployment, repo string) *Attempt {
	for i := range records {
		r := &records[i]
		if r.RepositoryKey != repo {
			continue
		}
		return &Attempt{ID: r.ID, Outcome: r.Outcome, At: r.StartedAt, RequestedRevision: r.RequestedRevision,
			BuiltRevision: r.BuiltRevision, RevisionBefore: r.RevisionBefore, ObservedRevision: r.ObservedRevision, Error: r.Error}
	}
	return nil
}
