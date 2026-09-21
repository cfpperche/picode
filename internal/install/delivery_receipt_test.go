//go:build unix

package install

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/cfpperche/picode/internal/gitgraph"
	"github.com/cfpperche/picode/internal/version"
)

// The documented shape, sorted: the delivery lane's reader validates the field
// set and rejects unknown or missing fields, so this list is the contract
// (docs/plans/delivery-publication.md § Deployment receipt). A `started`
// attempt is the same record without finishedAt.
var (
	receiptStartedFields  = []string{"actor", "bootAfter", "bootBefore", "builtRevision", "clean", "error", "id", "kind", "observedRevision", "outcome", "repositoryKey", "requestedRevision", "revisionBefore", "schemaVersion", "startedAt"}
	receiptFinishedFields = []string{"actor", "bootAfter", "bootBefore", "builtRevision", "clean", "error", "finishedAt", "id", "kind", "observedRevision", "outcome", "repositoryKey", "requestedRevision", "revisionBefore", "schemaVersion", "startedAt"}
)

// gitRepo makes a real repository: the receipt's key, requested revision and
// cleanliness all come from the process's cwd, and a faked git would only
// prove the fake — internal/gitgraph already owns what a repository key is.
func gitRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	for _, args := range [][]string{
		{"init", "-q"},
		{"-c", "user.email=test@example.com", "-c", "user.name=test", "commit", "-q", "--allow-empty", "-m", "start"},
	} {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v (%s)", args, err, out)
		}
	}
	return dir
}

// isRestart reports whether a faked Run call is the one that restarts the unit
// — Run("systemctl", "--user", "restart", UnitName).
func isRestart(args []string) bool { return slices.Contains(args, "restart") }

func headRevision(t *testing.T, dir string) string {
	t.Helper()
	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	return strings.TrimSpace(string(out))
}

// readRecord reads an attempt the way a reader does: the contract is the JSON,
// not the struct that wrote it.
func readRecord(t *testing.T, dir, name string) map[string]any {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(dir, name))
	if err != nil {
		t.Fatal(err)
	}
	var rec map[string]any
	if err := json.Unmarshal(raw, &rec); err != nil {
		t.Fatalf("%s: %v", name, err)
	}
	return rec
}

// receiptFiles lists the receipts a data dir holds, "" when the directory is
// not one at all (a symlinked or file-shaped var/ never becomes a directory).
func receiptFiles(dataDir string) []string {
	ents, err := os.ReadDir(receiptDir(dataDir))
	if err != nil {
		return nil
	}
	names := make([]string, 0, len(ents))
	for _, e := range ents {
		names = append(names, e.Name())
	}
	sort.Strings(names)
	return names
}

func assertFields(t *testing.T, rec map[string]any, want []string) {
	t.Helper()
	got := make([]string, 0, len(rec))
	for k := range rec {
		got = append(got, k)
	}
	sort.Strings(got)
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("fields = %v, want %v", got, want)
	}
}

func writeDiscovery(t *testing.T, dataDir, revision, boot string) {
	t.Helper()
	body := []byte(`{"url":"https://localhost:8445","revision":"` + revision + `","boot":"` + boot + `"}`)
	if err := os.WriteFile(filepath.Join(dataDir, "server.json"), body, 0o600); err != nil {
		t.Fatal(err)
	}
}

// deployHome is a home something was installed into: the unit, the installed
// binary, and a data dir whose mode the caller picks.
func deployHome(t *testing.T, dataMode fs.FileMode) (string, Paths, string) {
	t.Helper()
	home := t.TempDir()
	p := ForHome(home)
	for _, d := range []string{filepath.Dir(p.Unit), filepath.Dir(p.Bin), p.Data} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(p.Unit, []byte("[Unit]\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p.Bin, []byte("OLD BINARY"), 0o755); err != nil {
		t.Fatal(err)
	}
	src := filepath.Join(t.TempDir(), "picode-new")
	if err := os.WriteFile(src, []byte("NEW BINARY"), 0o755); err != nil {
		t.Fatal(err)
	}
	// A running daemon from before the identity fields existed: the deploy has
	// something to observe, and a read-only data dir can still be prepared.
	writeDiscovery(t, p.Data, "", "")
	if err := os.Chmod(p.Data, dataMode); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(p.Data, 0o700) })
	return home, p, src
}

// fakeDeploy replaces the seams a deploy outside a real systemd session needs.
func fakeDeploy(t *testing.T, run func(string, ...string) error) {
	t.Helper()
	prevCheck, prevRun, prevReady := EnsureUserSession, Run, Readiness
	t.Cleanup(func() { EnsureUserSession, Run, Readiness = prevCheck, prevRun, prevReady })
	EnsureUserSession = func() error { return nil }
	Readiness = func(string) ([]Busy, error) { return nil, nil }
	Run = run
}

// ADR-0170: one attempt is one file, begun before the machine changes and
// replaced — never appended to — when it ends with what answered.
func TestDeployReceiptRoundTrip(t *testing.T) {
	repo := gitRepo(t)
	t.Chdir(repo)
	t.Setenv(termIDEnv, "term-7")

	for _, tc := range []struct {
		what    string
		outcome string
		summary string
	}{
		{"passed", receiptOutcomePassed, ""},
		{"failed", receiptOutcomeFailed, "systemctl restart: exit status 1"},
	} {
		t.Run(tc.what, func(t *testing.T) {
			data := t.TempDir()
			writeDiscovery(t, data, strings.Repeat("a", 40), "boot-1")

			facts := gatherDeployFacts(data)
			if facts.RepositoryKey != gitgraph.Key(repo) || facts.RequestedRevision != headRevision(t, repo) {
				t.Fatalf("facts from a repository: %+v", facts)
			}
			if facts.RevisionBefore != strings.Repeat("a", 40) || facts.BootBefore != "boot-1" {
				t.Fatalf("the running daemon's identity is part of the facts: %+v", facts)
			}
			if !facts.Clean || facts.UncleanReason != "" {
				t.Fatalf("a repository with no work in it is clean: %+v", facts)
			}
			if facts.Actor != "term-7" {
				t.Fatalf("actor = %q, want the terminal the deploy ran in", facts.Actor)
			}

			id := beginDeployReceipt(data, facts)
			if !receiptIDOK(id) {
				t.Fatalf("begin returned %q", id)
			}
			dir := receiptDir(data)
			if got := receiptFiles(data); len(got) != 1 || got[0] != id+".json" {
				t.Fatalf("one attempt is one file: %v", got)
			}
			if fi, err := os.Stat(dir); err != nil || fi.Mode().Perm() != 0o700 {
				t.Fatalf("receipt dir = %v, %v", fi.Mode(), err)
			}
			if fi, err := os.Stat(filepath.Join(dir, id+".json")); err != nil || fi.Mode().Perm() != 0o600 {
				t.Fatalf("receipt file = %v, %v", fi.Mode(), err)
			}

			started := readRecord(t, dir, id+".json")
			assertFields(t, started, receiptStartedFields)
			if started["outcome"] != receiptOutcomeStarted || started["kind"] != receiptKindDeploy {
				t.Fatalf("record = %v", started)
			}
			if started["schemaVersion"] != float64(receiptSchemaVersion) {
				t.Fatalf("schemaVersion = %v", started["schemaVersion"])
			}
			if started["repositoryKey"] != facts.RepositoryKey || started["clean"] != true || started["error"] != "" {
				t.Fatalf("record = %v", started)
			}
			if started["actor"] != "term-7" {
				t.Fatalf("actor = %v", started["actor"])
			}

			finishDeployReceipt(data, id, tc.outcome, strings.Repeat("b", 40), "boot-2", tc.summary)

			done := readRecord(t, dir, id+".json")
			assertFields(t, done, receiptFinishedFields)
			if done["outcome"] != tc.outcome || done["error"] != tc.summary {
				t.Fatalf("outcome/error = %v/%v", done["outcome"], done["error"])
			}
			if done["startedAt"] != started["startedAt"] {
				t.Fatalf("startedAt moved: %v → %v", started["startedAt"], done["startedAt"])
			}
			if done["observedRevision"] != strings.Repeat("b", 40) || done["bootAfter"] != "boot-2" {
				t.Fatalf("the finish carries what answered: %v", done)
			}
			if got := receiptFiles(data); len(got) != 1 {
				t.Fatalf("finish replaces, never appends: %v", got)
			}
			start, err := time.Parse(time.RFC3339Nano, started["startedAt"].(string))
			if err != nil {
				t.Fatal(err)
			}
			end, err := time.Parse(time.RFC3339Nano, done["finishedAt"].(string))
			if err != nil {
				t.Fatalf("finishedAt = %v", done["finishedAt"])
			}
			if end.Before(start) {
				t.Fatalf("finishedAt %v is before startedAt %v", end, start)
			}
		})
	}
}

// A deploy killed mid-flight leaves exactly this: a record that says the
// attempt began, with no outcome to read as passed or failed.
func TestDeployReceiptStartedIsWhatAKilledDeployLeaves(t *testing.T) {
	t.Chdir(gitRepo(t))
	data := t.TempDir()

	id := beginDeployReceipt(data, gatherDeployFacts(data))
	if id == "" {
		t.Fatal("no receipt was written")
	}
	rec := readRecord(t, receiptDir(data), id+".json")
	assertFields(t, rec, receiptStartedFields)
	if rec["outcome"] != receiptOutcomeStarted {
		t.Fatalf("outcome = %v", rec["outcome"])
	}
	if _, ok := rec["finishedAt"]; ok {
		t.Fatal("a killed deploy must not look finished")
	}
	if got := receiptFiles(data); len(got) != 1 || strings.HasSuffix(got[0], ".tmp") {
		t.Fatalf("the atomic write left something behind: %v", got)
	}
}

// A receipt nobody can attribute is not evidence: ADR-0170 requires a
// repository key, so a deploy from outside a repository records nothing rather
// than leaving a file every observer has to reject as foreign.
func TestDeployReceiptOutsideARepositoryWritesNothing(t *testing.T) {
	t.Chdir(t.TempDir())
	data := t.TempDir()

	if id := beginDeployReceipt(data, gatherDeployFacts(data)); id != "" {
		t.Fatalf("begin = %q, want no receipt", id)
	}
	if got := receiptFiles(data); len(got) != 0 {
		t.Fatalf("wrote %v", got)
	}
}

// ADR-0170 rejects symlinked receipt paths rather than following them: the
// record names this machine's deployment, and a link would let it land — and
// be read — where the data dir does not own it.
func TestDeployReceiptRefusesToWriteThroughASymlink(t *testing.T) {
	t.Chdir(gitRepo(t))
	for _, tc := range []struct {
		what string
		link func(t *testing.T, data, outside string)
	}{
		{"var", func(t *testing.T, data, outside string) {
			if err := os.Symlink(outside, filepath.Join(data, "var")); err != nil {
				t.Fatal(err)
			}
		}},
		{"delivery", func(t *testing.T, data, outside string) {
			if err := os.Mkdir(filepath.Join(data, "var"), 0o700); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(outside, filepath.Join(data, "var", "delivery")); err != nil {
				t.Fatal(err)
			}
		}},
		{"a file where the directory should be", func(t *testing.T, data, outside string) {
			if err := os.WriteFile(filepath.Join(data, "var"), []byte("not a dir"), 0o600); err != nil {
				t.Fatal(err)
			}
		}},
	} {
		t.Run(tc.what, func(t *testing.T) {
			data, outside := t.TempDir(), t.TempDir()
			tc.link(t, data, outside)

			if id := beginDeployReceipt(data, gatherDeployFacts(data)); id != "" {
				t.Fatalf("begin = %q, want a refusal", id)
			}
			// finish walks the same path, so it is refused too — even for an
			// id shaped like ours.
			finishDeployReceipt(data, "11111111-2222-4333-8444-555555555555", receiptOutcomeFailed, "", "", "boom")

			if got := receiptFiles(outside); len(got) != 0 {
				t.Fatalf("a receipt was written through the link: %v", got)
			}
			if got := receiptFiles(data); len(got) != 0 {
				t.Fatalf("a receipt was written: %v", got)
			}
		})
	}
}

// ADR-0170: a receipt that cannot be written never changes the deploy's
// result. Both halves matter — the deploy that worked still answers nil, and
// the deploy that failed answers its own error, not the receipt's.
func TestDeployReceiptWriteFailureIsSilent(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root ignores directory permissions, so nothing here can fail")
	}
	t.Chdir(gitRepo(t))

	t.Run("a successful deploy still answers nil", func(t *testing.T) {
		home, p, src := deployHome(t, 0o500)
		fakeDeploy(t, func(name string, args ...string) error {
			if isRestart(args) {
				// The file itself is still writable, so the new daemon can
				// publish; only the receipt dir cannot be created.
				_ = os.WriteFile(filepath.Join(p.Data, "server.json"), []byte(`{"revision":"","boot":"boot-2"}`), 0o600)
			}
			return nil
		})

		if err := DeployForce(src, home, "/usr/bin", true); err != nil {
			t.Fatalf("a deploy that worked must not fail on a receipt: %v", err)
		}
		if got := receiptFiles(p.Data); len(got) != 0 {
			t.Fatalf("a receipt appeared in a read-only data dir: %v", got)
		}
	})

	t.Run("a failed deploy still answers its own error", func(t *testing.T) {
		home, p, src := deployHome(t, 0o500)
		refused := errors.New("restart refused")
		fakeDeploy(t, func(name string, args ...string) error {
			if isRestart(args) {
				return refused
			}
			return nil
		})

		err := DeployForce(src, home, "/usr/bin", true)
		if !errors.Is(err, refused) {
			t.Fatalf("got %v, want the restart's own error", err)
		}
		if got := receiptFiles(p.Data); len(got) != 0 {
			t.Fatalf("a receipt appeared in a read-only data dir: %v", got)
		}
	})
}

// The deployment producer end to end: a deploy that restarts writes the
// attempt and the identity that answered after it.
func TestDeployForceRecordsAPassedReceipt(t *testing.T) {
	repo := gitRepo(t)
	t.Chdir(repo)
	home, p, src := deployHome(t, 0o700)
	before := strings.Repeat("a", 40)
	after := strings.Repeat("b", 40)
	writeDiscovery(t, p.Data, before, "boot-1")

	fakeDeploy(t, func(name string, args ...string) error {
		if isRestart(args) {
			// The daemon needs a moment to bind and republish, so the new
			// identity arrives from the side: sampling is what has to catch it.
			go func() {
				time.Sleep(150 * time.Millisecond)
				_ = os.WriteFile(filepath.Join(p.Data, "server.json"),
					[]byte(`{"revision":"`+after+`","boot":"boot-2"}`), 0o600)
			}()
		}
		return nil
	})

	if err := DeployForce(src, home, "/usr/bin", true); err != nil {
		t.Fatalf("deploy: %v", err)
	}
	if body, _ := os.ReadFile(p.Bin); string(body) != "NEW BINARY" {
		t.Fatalf("the deploy did not replace the binary: %q", body)
	}

	names := receiptFiles(p.Data)
	if len(names) != 1 {
		t.Fatalf("receipts = %v, want exactly one", names)
	}
	rec := readRecord(t, receiptDir(p.Data), names[0])
	assertFields(t, rec, receiptFinishedFields)
	if rec["id"] != strings.TrimSuffix(names[0], ".json") {
		t.Fatalf("id = %v, file = %v", rec["id"], names[0])
	}
	if rec["outcome"] != receiptOutcomePassed || rec["error"] != "" {
		t.Fatalf("outcome/error = %v/%v", rec["outcome"], rec["error"])
	}
	if rec["builtRevision"] != version.Revision() {
		t.Fatalf("builtRevision = %v, want the copied binary's %q", rec["builtRevision"], version.Revision())
	}
	if rec["repositoryKey"] != gitgraph.Key(repo) || rec["requestedRevision"] != headRevision(t, repo) {
		t.Fatalf("repository/revision = %v/%v", rec["repositoryKey"], rec["requestedRevision"])
	}
	if rec["clean"] != true {
		t.Fatalf("clean = %v", rec["clean"])
	}
	if rec["revisionBefore"] != before || rec["bootBefore"] != "boot-1" {
		t.Fatalf("before = %v/%v", rec["revisionBefore"], rec["bootBefore"])
	}
	if rec["observedRevision"] != after || rec["bootAfter"] != "boot-2" {
		t.Fatalf("after = %v/%v", rec["observedRevision"], rec["bootAfter"])
	}
	start, err := time.Parse(time.RFC3339Nano, rec["startedAt"].(string))
	if err != nil {
		t.Fatal(err)
	}
	end, err := time.Parse(time.RFC3339Nano, rec["finishedAt"].(string))
	if err != nil {
		t.Fatal(err)
	}
	if end.Before(start) {
		t.Fatalf("finishedAt %v is before startedAt %v", end, start)
	}
}

// A deploy that fails after the receipt began ends that same attempt as
// failed, with the error the caller got — and the caller's error is unchanged.
func TestDeployForceRecordsAFailedReceipt(t *testing.T) {
	t.Chdir(gitRepo(t))
	refused := errors.New("restart refused")
	for _, tc := range []struct {
		what string
		src  func(t *testing.T, ok string) string
		run  func(string, ...string) error
		want string
	}{
		{
			what: "the copy fails",
			src:  func(t *testing.T, _ string) string { return filepath.Join(t.TempDir(), "absent") },
			run:  func(string, ...string) error { return nil },
			want: "copy binary: ",
		},
		{
			what: "the restart fails",
			src:  func(_ *testing.T, ok string) string { return ok },
			run: func(name string, args ...string) error {
				if isRestart(args) {
					return refused
				}
				return nil
			},
			want: "systemctl restart: ",
		},
	} {
		t.Run(tc.what, func(t *testing.T) {
			home, p, src := deployHome(t, 0o700)
			writeDiscovery(t, p.Data, strings.Repeat("a", 40), "boot-1")
			fakeDeploy(t, tc.run)

			err := DeployForce(tc.src(t, src), home, "/usr/bin", true)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("got %v, want %q", err, tc.want)
			}
			if tc.want == "systemctl restart: " && !errors.Is(err, refused) {
				t.Fatalf("the caller's own error must survive: %v", err)
			}

			names := receiptFiles(p.Data)
			if len(names) != 1 {
				t.Fatalf("receipts = %v, want exactly one", names)
			}
			rec := readRecord(t, receiptDir(p.Data), names[0])
			assertFields(t, rec, receiptFinishedFields)
			if rec["outcome"] != receiptOutcomeFailed || rec["error"] != err.Error() {
				t.Fatalf("outcome/error = %v/%v, want failed/%q", rec["outcome"], rec["error"], err.Error())
			}
			if rec["observedRevision"] != "" || rec["bootAfter"] != "" {
				t.Fatalf("a failed attempt observed nothing: %v/%v", rec["observedRevision"], rec["bootAfter"])
			}
			if rec["revisionBefore"] != strings.Repeat("a", 40) || rec["bootBefore"] != "boot-1" {
				t.Fatalf("before = %v/%v", rec["revisionBefore"], rec["bootBefore"])
			}
		})
	}
}

// One rule, deliberately: clean is true only when `git status --porcelain`
// succeeded and printed nothing. A status read that fails leaves the question
// open, and an open question is not a clean artifact.
func TestArtifactCleanRule(t *testing.T) {
	repo, dirty, unreadable := gitRepo(t), gitRepo(t), t.TempDir()
	if err := os.WriteFile(filepath.Join(dirty, "scratch.txt"), []byte("wip"), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		what       string
		dir        string
		wantClean  bool
		wantReason bool
	}{
		{"a repository with nothing to commit", repo, true, false},
		{"a repository with an untracked file", dirty, false, false},
		{"a directory whose status cannot be read", unreadable, false, true},
	} {
		t.Run(tc.what, func(t *testing.T) {
			clean, reason := artifactClean(tc.dir)
			if clean != tc.wantClean {
				t.Fatalf("clean = %v, want %v (reason %q)", clean, tc.wantClean, reason)
			}
			if (reason != "") != tc.wantReason {
				t.Fatalf("reason = %q, want a reason: %v", reason, tc.wantReason)
			}
		})
	}
}

// An unclean-but-read artifact and an unreadable one are different claims, and
// the reason for the second rides `error`; a failed deploy's own error is the
// attempt's and replaces it. Every string is bounded and one line, and a
// malformed id is never joined into a path.
func TestDeployReceiptReasonAndBounds(t *testing.T) {
	data := t.TempDir()
	reason := "working tree not read: exit status 128"
	id := beginDeployReceipt(data, deployFacts{RepositoryKey: "/repo/.git", UncleanReason: reason})
	if id == "" {
		t.Fatal("no receipt")
	}
	dir := receiptDir(data)

	rec := readRecord(t, dir, id+".json")
	if rec["clean"] != false || rec["error"] != reason {
		t.Fatalf("clean/error = %v/%v", rec["clean"], rec["error"])
	}

	// A passed deploy has nothing to add: the reason survives the finish.
	finishDeployReceipt(data, id, receiptOutcomePassed, "", "", "")
	if rec = readRecord(t, dir, id+".json"); rec["error"] != reason || rec["clean"] != false {
		t.Fatalf("clean/error = %v/%v", rec["clean"], rec["error"])
	}

	// The failure's own error is the attempt's.
	finishDeployReceipt(data, id, receiptOutcomeFailed, "", "", "systemctl restart: exit status 1\nno such unit")
	if rec = readRecord(t, dir, id+".json"); rec["error"] != "systemctl restart: exit status 1no such unit" {
		t.Fatalf("error = %q", rec["error"])
	}

	finishDeployReceipt(data, id, receiptOutcomeFailed, "", "", strings.Repeat("x", 4*receiptFieldMax))
	if rec = readRecord(t, dir, id+".json"); len(rec["error"].(string)) != receiptFieldMax {
		t.Fatalf("error holds %d bytes", len(rec["error"].(string)))
	}

	// Nothing to finish: an empty id, an id that is not ours, a record that is
	// gone. None of them may create a file.
	finishDeployReceipt(data, "", receiptOutcomeFailed, "", "", "boom")
	finishDeployReceipt(data, "../../etc/passwd", receiptOutcomeFailed, "", "", "boom")
	if got := receiptFiles(data); len(got) != 1 {
		t.Fatalf("receipts = %v", got)
	}
}

// server.json is the daemon's identity. The receipt copies what it says, and a
// file that is missing, unreadable or without identity leaves the fields empty
// rather than inventing one.
func TestDiscoveryIdentity(t *testing.T) {
	long := strings.Repeat("9", receiptFieldMax+40)
	for _, tc := range []struct {
		what         string
		body         string
		wantRevision string
		wantBoot     string
	}{
		{"no file", "", "", ""},
		{"not json", "{{{", "", ""},
		{"no identity fields", `{"url":"https://localhost:8445"}`, "", ""},
		{"identity", `{"url":"https://localhost:8445","revision":"abc123","boot":"boot-1"}`, "abc123", "boot-1"},
		{"control characters dropped", "{\"revision\":\"a\\nb\",\"boot\":\"c\\u0007d\"}", "ab", "cd"},
		{"bounded", `{"revision":"` + long + `"}`, strings.Repeat("9", receiptFieldMax), ""},
	} {
		t.Run(tc.what, func(t *testing.T) {
			data := t.TempDir()
			if tc.body != "" {
				if err := os.WriteFile(filepath.Join(data, "server.json"), []byte(tc.body), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			revision, boot := discoveryIdentity(data)
			if revision != tc.wantRevision || boot != tc.wantBoot {
				t.Fatalf("got (%q, %q), want (%q, %q)", revision, boot, tc.wantRevision, tc.wantBoot)
			}
		})
	}
}

// The restart returns before the new daemon has republished, so the identity
// after it is sampled until something other than the replaced process answers.
func TestAwaitDaemonIdentityWaitsForANewBoot(t *testing.T) {
	data := t.TempDir()
	writeDiscovery(t, data, "old-revision", "boot-1")
	go func() {
		time.Sleep(300 * time.Millisecond)
		_ = os.WriteFile(filepath.Join(data, "server.json"), []byte(`{"revision":"new-revision","boot":"boot-2"}`), 0o600)
	}()

	start := time.Now()
	revision, boot := awaitDaemonIdentity(data, "old-revision", "boot-1")
	if revision != "new-revision" || boot != "boot-2" {
		t.Fatalf("got (%q, %q), want the restarted daemon", revision, boot)
	}
	if time.Since(start) < 250*time.Millisecond {
		t.Fatal("the first sample was returned as the answer")
	}

	// A daemon that never comes back leaves what was there: the same boot says
	// nothing new is serving, which is an answer rather than a hang.
	writeDiscovery(t, data, "old-revision", "boot-1")
	start = time.Now()
	revision, boot = awaitDaemonIdentity(data, "old-revision", "boot-1")
	if revision != "old-revision" || boot != "boot-1" {
		t.Fatalf("got (%q, %q), want the values it could see", revision, boot)
	}
	if elapsed := time.Since(start); elapsed > receiptDiscoveryWait+time.Second {
		t.Fatalf("sampling ran %v", elapsed)
	}
}
