package delivery

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// envFixture is the pilot's shape in miniature: main contains a feature commit
// on top of the initial one, so a running artifact at the initial revision is
// *behind* the target and the feature is integrated without being published.
func envFixture(t *testing.T) (dir, repo, running string, feature string) {
	t.Helper()
	dir, repo = fixture(t)
	initial := run(t, dir, "rev-parse", "main")
	run(t, dir, "checkout", "-b", "feature")
	feature = commit(t, dir)
	run(t, dir, "checkout", "main")
	run(t, dir, "merge", "--ff-only", "feature")
	return dir, repo, initial, feature
}

func deploymentReceipt(t *testing.T, dataDir string, d Deployment) {
	t.Helper()
	dir := filepath.Join(dataDir, "var", "delivery")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(d)
	if err := os.WriteFile(filepath.Join(dir, d.ID+".json"), raw, 0o600); err != nil {
		t.Fatal(err)
	}
}

func deployRecord(id, repo, outcome string, at time.Time) Deployment {
	d := Deployment{SchemaVersion: 1, ID: id, Kind: "deploy", RepositoryKey: repo, StartedAt: at.UTC().Format(time.RFC3339Nano),
		Outcome: outcome, RequestedRevision: strings.Repeat("a", 40), BuiltRevision: strings.Repeat("b", 40), Clean: true}
	if outcome != "started" {
		d.FinishedAt = at.Add(20 * time.Second).UTC().Format(time.RFC3339Nano)
	}
	return d
}

// id builds a 36-character UUID-shaped id: the receipt parser accepts
// nothing else, and a short one is a silent "unrecognized record".
func id(seed string) string { return fmt.Sprintf("%08s-1111-4111-8111-111111111111", seed) }

// O16–O21 and O25/O26 as one table: every row names the environment fact set
// it must produce, including the ones that must stay unknown.
func TestEnvironmentDecisionTable(t *testing.T) {
	ctx := context.Background()
	dir, repo, running, feature := envFixture(t)
	data := t.TempDir()
	binding := &Binding{WorkspaceID: "w1", Repo: repo, Observer: "picode-self"}
	change := Change{ID: "chg", Branch: "feature", Revision: feature, Integration: "integrated"}
	fixed := func() Running {
		return Running{Display: "0.4.0+abc1234", Revision: running, Boot: "boot-1", Responding: true}
	}

	cases := []struct {
		name       string
		in         EnvironmentInput
		wantStatus string
		wantReason string
		wantCount  int
		wantPub    string
	}{
		{"O16 no binding", EnvironmentInput{Cwd: dir, Repo: repo, TargetOID: run(t, dir, "rev-parse", "main"), Sample: fixed}, "unconfigured", "no-observer", 0, "unknown"},
		{"conflicting bindings", EnvironmentInput{Cwd: dir, Repo: repo, Binding: binding, Conflict: true, TargetOID: run(t, dir, "rev-parse", "main"), Sample: fixed}, "conflict", "binding-conflict", 0, "unknown"},
		{"moved folder", EnvironmentInput{Cwd: dir, Repo: repo, Binding: &Binding{Repo: "/elsewhere/.git", Observer: "picode-self"}, TargetOID: run(t, dir, "rev-parse", "main"), Sample: fixed}, "unknown", "repository-mismatch", 0, "unknown"},
		{"unsupported observer", EnvironmentInput{Cwd: dir, Repo: repo, Binding: &Binding{Repo: repo, Observer: "somewhere-else"}, TargetOID: run(t, dir, "rev-parse", "main"), Sample: fixed}, "unknown", "unsupported-observer", 0, "unknown"},
		{"O17 revision unknown", EnvironmentInput{Cwd: dir, Repo: repo, Binding: binding, TargetOID: run(t, dir, "rev-parse", "main"), Sample: func() Running { return Running{Display: "0.4.0", Boot: "b", Responding: true} }}, "unknown", "revision-unknown", 0, "unknown"},
		{"O17 revision not in this repository", EnvironmentInput{Cwd: dir, Repo: repo, Binding: binding, TargetOID: run(t, dir, "rev-parse", "main"), Sample: func() Running {
			return Running{Display: "0.4.0+deadbee", Revision: strings.Repeat("d", 40), Boot: "b", Responding: true}
		}}, "unknown", "revision-unmapped", 0, "unknown"},
		{"O19 boot changed twice", EnvironmentInput{Cwd: dir, Repo: repo, Binding: binding, TargetOID: run(t, dir, "rev-parse", "main"), Sample: flakySample(running)}, "unknown", "boot-changed", 0, "unknown"},
		{"no target", EnvironmentInput{Cwd: dir, Repo: repo, Binding: binding, Sample: fixed}, "unknown", "target-required", 0, "unknown"},
		{"divergent history", EnvironmentInput{Cwd: dir, Repo: repo, Binding: binding, TargetOID: run(t, dir, "rev-parse", "main"), Sample: orphanSample(t, dir)}, "unknown", "divergent-history", 0, "unknown"},
		{"O18 target ahead", EnvironmentInput{Cwd: dir, Repo: repo, Binding: binding, TargetOID: run(t, dir, "rev-parse", "main"), Sample: fixed}, "known", "", 1, "not-published"},
		{"O20 failed attempt keeps the live revision", EnvironmentInput{Cwd: dir, Repo: repo, Binding: binding, TargetOID: run(t, dir, "rev-parse", "main"), Sample: fixed, DataDir: data}, "known", "", 1, "not-published"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			changes := []Change{change}
			env, marked := ObserveEnvironment(ctx, c.in, changes)
			if env.Status != c.wantStatus || env.ReasonCode != c.wantReason {
				t.Fatalf("status/reason = %q/%q, want %q/%q", env.Status, env.ReasonCode, c.wantStatus, c.wantReason)
			}
			if env.Unpublished.Count != c.wantCount {
				t.Fatalf("unpublished = %d (%v), want %d", env.Unpublished.Count, env.Unpublished.Changes, c.wantCount)
			}
			if marked[0].Publication != c.wantPub {
				t.Fatalf("publication = %q, want %q", marked[0].Publication, c.wantPub)
			}
			if env.Busy.Status != "unknown" {
				t.Fatalf("busy without a guard read = %+v, want unknown (O25)", env.Busy)
			}
		})
	}
}

func flakySample(revision string) func() Running {
	calls := 0
	return func() Running {
		calls++
		return Running{Display: "0.4.0+abc1234", Revision: revision, Boot: fmt.Sprintf("boot-%d", calls), Responding: true}
	}
}

// orphanSample is a revision this repository has, but on no line to the
// target: neither contains the other.
func orphanSample(t *testing.T, dir string) func() Running {
	t.Helper()
	run(t, dir, "checkout", "--orphan", "orphan")
	sha := commit(t, dir)
	run(t, dir, "checkout", "main")
	return func() Running {
		return Running{Display: "0.4.0+orphan", Revision: sha, Boot: "boot-1", Responding: true}
	}
}

// O26: one commit delivered by two changes is one publication entry.
func TestUnpublishedDeduplicatesOneCommit(t *testing.T) {
	dir, repo, running, feature := envFixture(t)
	changes := []Change{
		{ID: "a", Branch: "feature", Revision: feature, Integration: "integrated"},
		{ID: "b", Branch: "feature-alias", Revision: feature, Integration: "integrated"},
	}
	env, marked := ObserveEnvironment(context.Background(), EnvironmentInput{
		Cwd: dir, Repo: repo, Binding: &Binding{Repo: repo, Observer: "picode-self"},
		TargetOID: run(t, dir, "rev-parse", "main"),
		Sample:    func() Running { return Running{Revision: running, Boot: "b", Responding: true} },
	}, changes)
	if env.Status != "known" || env.Unpublished.Count != 1 || env.Unpublished.Changes[0] != "a" {
		t.Fatalf("env = %+v", env)
	}
	for _, c := range marked {
		if c.Publication != "not-published" {
			t.Fatalf("%s publication = %q", c.ID, c.Publication)
		}
	}
}

func TestPublishedWhenTheRunningRevisionContainsTheChange(t *testing.T) {
	dir, repo, _, feature := envFixture(t)
	env, marked := ObserveEnvironment(context.Background(), EnvironmentInput{
		Cwd: dir, Repo: repo, Binding: &Binding{Repo: repo, Observer: "picode-self"},
		TargetOID: run(t, dir, "rev-parse", "main"),
		Sample:    func() Running { return Running{Revision: feature, Boot: "b", Responding: true} },
	}, []Change{{ID: "a", Branch: "feature", Revision: feature, Integration: "integrated"}})
	if env.Status != "known" || env.Unpublished.Count != 0 || marked[0].Publication != "published" {
		t.Fatalf("env = %+v marked = %+v", env, marked)
	}
}

func TestArtifactStateNeedsAPassingRecordForTheRunningRevision(t *testing.T) {
	rev := strings.Repeat("c", 40)
	rows := []struct {
		name     string
		records  []Deployment
		revision string
		want     string
	}{
		{"no receipt", nil, rev, "unknown"},
		{"no revision", []Deployment{deployRecord(id("1"), "/r", "passed", time.Now())}, "", "unknown"},
		{"passed clean", []Deployment{{SchemaVersion: 1, ID: id("1"), Kind: "deploy", RepositoryKey: "/r", StartedAt: time.Now().UTC().Format(time.RFC3339Nano), Outcome: "passed", BuiltRevision: rev, ObservedRevision: rev, Clean: true}}, rev, "clean"},
		{"passed dirty", []Deployment{{SchemaVersion: 1, ID: id("1"), Kind: "deploy", RepositoryKey: "/r", StartedAt: time.Now().UTC().Format(time.RFC3339Nano), Outcome: "passed", BuiltRevision: rev, ObservedRevision: rev}}, rev, "dirty"},
		{"failed only", []Deployment{{SchemaVersion: 1, ID: id("1"), Kind: "deploy", RepositoryKey: "/r", StartedAt: time.Now().UTC().Format(time.RFC3339Nano), Outcome: "failed", BuiltRevision: rev, ObservedRevision: rev, Clean: true}}, rev, "unknown"},
	}
	for _, r := range rows {
		if got := artifactState(r.records, r.revision); got != r.want {
			t.Errorf("%s: artifactState = %q, want %q", r.name, got, r.want)
		}
	}
}

// A deployment directory is data: unsafe, foreign or malformed records are
// issues, and a record that never finished is unknown (ADR-0170).
func TestReadDeploymentsConfinement(t *testing.T) {
	data := t.TempDir()
	dir := filepath.Join(data, "var", "delivery")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	good := deployRecord(id("aaaa"), "/repo/.git", "passed", time.Now().Add(-time.Minute))
	write := func(name, body string, mode os.FileMode) {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), mode); err != nil {
			t.Fatal(err)
		}
	}
	raw, _ := json.Marshal(good)
	write(good.ID+".json", string(raw), 0o600)
	started := deployRecord(id("bbbb"), "/repo/.git", "started", time.Now().Add(-2*time.Minute))
	rawStarted, _ := json.Marshal(started)
	write(started.ID+".json", string(rawStarted), 0o600)
	write("notes.txt", "x", 0o600)

	records, issues := ReadDeployments(data)
	if len(records) != 2 || len(issues) != 0 {
		t.Fatalf("records = %d issues = %v, want the two valid records and no issue", len(records), issues)
	}
	for _, r := range records {
		if r.ID == id("bbbb") && r.Outcome != "unknown" {
			t.Fatalf("unfinished record observed as %q, want unknown", r.Outcome)
		}
	}
	if records[0].ID != good.ID {
		t.Fatalf("records must be newest first, got %s then %s", records[0].ID, records[1].ID)
	}
	if out, _ := ReadDeployments(""); len(out) != 0 {
		t.Fatalf("no data dir must read nothing, got %v", out)
	}
	if _, issues := ReadDeployments(t.TempDir()); len(issues) != 0 {
		t.Fatalf("a data dir without receipts is not an error: %v", issues)
	}

	// Each rejected shape gets its own directory: the issue text is
	// deliberately one sentence, so uniqueness inside one directory would
	// hide a second bad record.
	bad := []struct {
		name string
		body string
		mode os.FileMode
	}{
		{"malformed JSON", "{not json", 0o600},
		{"unknown schema", `{"schemaVersion":2,"id":"` + id("cccc") + `","kind":"deploy","repositoryKey":"/r","startedAt":"2026-01-01T00:00:00Z","outcome":"passed","finishedAt":"2026-01-01T00:00:10Z"}`, 0o600},
		{"name does not match the id", string(raw), 0o600},
		{"not a deploy", `{"schemaVersion":1,"id":"` + id("dddd") + `","kind":"land","repositoryKey":"/r","startedAt":"2026-01-01T00:00:00Z","outcome":"passed","finishedAt":"2026-01-01T00:00:10Z"}`, 0o600},
		{"implausible start time", `{"schemaVersion":1,"id":"` + id("eeee") + `","kind":"deploy","repositoryKey":"/r","startedAt":"2099-01-01T00:00:00Z","outcome":"passed","finishedAt":"2099-01-01T00:00:10Z"}`, 0o600},
		{"abbreviated revision", `{"schemaVersion":1,"id":"` + id("ffff") + `","kind":"deploy","repositoryKey":"/r","startedAt":"2026-01-01T00:00:00Z","outcome":"passed","finishedAt":"2026-01-01T00:00:10Z","builtRevision":"abc1234"}`, 0o600},
	}
	for _, c := range bad {
		t.Run(c.name, func(t *testing.T) {
			one := t.TempDir()
			d := filepath.Join(one, "var", "delivery")
			if err := os.MkdirAll(d, 0o700); err != nil {
				t.Fatal(err)
			}
			name := id("9999") + ".json"
			if c.name == "name does not match the id" {
				name = id("8888") + ".json"
			}
			if err := os.WriteFile(filepath.Join(d, name), []byte(c.body), c.mode); err != nil {
				t.Fatal(err)
			}
			got, issues := ReadDeployments(one)
			if len(got) != 0 || len(issues) == 0 {
				t.Fatalf("records = %v issues = %v, want the record rejected as an issue", got, issues)
			}
		})
	}

	if runtime.GOOS != "windows" {
		if err := os.Chmod(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		if _, issues := ReadDeployments(data); len(issues) == 0 {
			t.Fatal("a world-readable evidence directory must be reported")
		}
	}
}

func TestReadDeploymentsIgnoresASymlinkedDirectory(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink permissions differ on Windows")
	}
	real := t.TempDir()
	data := t.TempDir()
	if err := os.MkdirAll(filepath.Join(real, "delivery"), 0o700); err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(deployRecord(id("ffff"), "/r", "passed", time.Now()))
	if err := os.WriteFile(filepath.Join(real, "delivery", id("ffff")+".json"), raw, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(data, "var"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(real, "delivery"), filepath.Join(data, "var", "delivery")); err != nil {
		t.Fatal(err)
	}
	records, issues := ReadDeployments(data)
	if len(records) != 0 || len(issues) == 0 {
		t.Fatalf("records = %v issues = %v, want nothing read through the symlink", records, issues)
	}
}

func TestLatestAttemptIsTheNewestForThisRepository(t *testing.T) {
	now := time.Now()
	records := []Deployment{
		deployRecord(id("1111"), "/other/.git", "passed", now),
		deployRecord(id("2222"), "/mine/.git", "failed", now.Add(-time.Minute)),
		deployRecord(id("3333"), "/mine/.git", "passed", now.Add(-2*time.Minute)),
	}
	a := latestAttempt(records, "/mine/.git")
	if a == nil || a.ID != id("2222") || a.Outcome != "failed" {
		t.Fatalf("latest attempt = %+v", a)
	}
	if latestAttempt(records, "/absent/.git") != nil {
		t.Fatal("a repository with no record has no attempt")
	}
}

func TestBusyObservationNeverClaimsSafety(t *testing.T) {
	dir, repo, running, _ := envFixture(t)
	base := EnvironmentInput{Cwd: dir, Repo: repo, Binding: &Binding{Repo: repo, Observer: "picode-self"},
		TargetOID: run(t, dir, "rev-parse", "main"), Sample: func() Running { return Running{Revision: running, Boot: "b", Responding: true} }}
	env, _ := ObserveEnvironment(context.Background(), base, nil)
	if env.Busy.Status != "unknown" {
		t.Fatalf("missing guard read must be unknown, got %+v", env.Busy)
	}
	owners := 0
	base.Busy = &owners
	env, _ = ObserveEnvironment(context.Background(), base, nil)
	if env.Busy.Status != "known" || env.Busy.Owners != 0 {
		t.Fatalf("a complete guard read is a known observation, got %+v", env.Busy)
	}
	owners = 3
	env, _ = ObserveEnvironment(context.Background(), base, nil)
	if env.Busy.Owners != 3 {
		t.Fatalf("owners = %d, want 3", env.Busy.Owners)
	}
}

func TestRevisionIdentitySurvivesAStableBoot(t *testing.T) {
	dir, repo, running, _ := envFixture(t)
	env, _ := ObserveEnvironment(context.Background(), EnvironmentInput{
		Cwd: dir, Repo: repo, Binding: &Binding{Repo: repo, Observer: "picode-self"},
		TargetOID: run(t, dir, "rev-parse", "main"),
		Sample: func() Running {
			return Running{Display: "0.4.0+abc1234", Revision: running, Boot: "boot-1", Responding: true}
		},
	}, nil)
	if env.Revision != running || env.Boot != "boot-1" || env.DisplayVersion != "0.4.0+abc1234" {
		t.Fatalf("identity = %+v", env)
	}
	if env.Responding == nil || !*env.Responding {
		t.Fatalf("responding = %v, want true", env.Responding)
	}
}
