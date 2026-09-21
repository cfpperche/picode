package install

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/cfpperche/picode/internal/gitgraph"
	"github.com/cfpperche/picode/internal/version"
)

// The deployment receipt (ADR-0170): <data>/var/delivery/<uuid>.json, one
// record per deploy attempt, written by the process doing the deploy and read
// by the delivery lane. It answers what was deployed, from which revision, was
// the artifact clean, and what answered afterwards — facts nothing else on the
// machine records, and without which a restarted daemon's "which revision is
// serving?" has no counterpart to compare against. The whole producer is best
// effort: a receipt that cannot be written never changes the deploy's result,
// its error, or its exit code.
const (
	receiptSchemaVersion = 1
	receiptKindDeploy    = "deploy"

	receiptOutcomeStarted = "started"
	receiptOutcomePassed  = "passed"
	receiptOutcomeFailed  = "failed"

	// receiptFieldMax bounds every string on the wire. The contract caps
	// `error` at 512 bytes; the same budget for every field keeps one runaway
	// value from making the record unreadable to the reader that validates it.
	receiptFieldMax = 512

	// receiptGitTimeout bounds each git call. The receipt is best effort and
	// the deploy is not: a repository with a cold index, or one on a network
	// filesystem, must not turn evidence-gathering into a hang.
	receiptGitTimeout = 5 * time.Second

	// A restarted daemon needs a moment before it binds and rewrites
	// server.json, so the identity after the restart is sampled, bounded.
	receiptDiscoveryWait = 5 * time.Second
	receiptDiscoveryStep = 100 * time.Millisecond
)

// deployReceipt is the record on disk. These json tags are the contract the
// delivery lane reads; its reader rejects unknown or missing fields, so this
// struct is the whole shape. finishedAt is absent while the attempt is still
// running — the one difference between a live deploy and a killed one.
type deployReceipt struct {
	SchemaVersion     int    `json:"schemaVersion"`
	ID                string `json:"id"`
	Kind              string `json:"kind"`
	RepositoryKey     string `json:"repositoryKey"`
	Actor             string `json:"actor"`
	StartedAt         string `json:"startedAt"`
	FinishedAt        string `json:"finishedAt,omitempty"`
	Outcome           string `json:"outcome"`
	RequestedRevision string `json:"requestedRevision"`
	BuiltRevision     string `json:"builtRevision"`
	RevisionBefore    string `json:"revisionBefore"`
	ObservedRevision  string `json:"observedRevision"`
	BootBefore        string `json:"bootBefore"`
	BootAfter         string `json:"bootAfter"`
	Clean             bool   `json:"clean"`
	Error             string `json:"error"`
}

// deployFacts is what the deploying process can say about itself before the
// daemon restarts.
type deployFacts struct {
	RepositoryKey     string
	RequestedRevision string
	BuiltRevision     string
	RevisionBefore    string
	BootBefore        string
	Actor             string
	Clean             bool
	// UncleanReason explains a clean=false that came from not being able to
	// read the working tree at all. Empty when clean is the answer to a
	// question that was actually asked.
	UncleanReason string
}

// gatherDeployFacts observes the deploying process: its repository, the
// revision it asked for, its actor, and the daemon that is running right now.
func gatherDeployFacts(dataDir string) deployFacts {
	f := deployFacts{
		Actor:         os.Getenv(termIDEnv),
		BuiltRevision: version.Revision(),
	}
	f.RevisionBefore, f.BootBefore = discoveryIdentity(dataDir)
	cwd, err := os.Getwd()
	if err != nil {
		return f
	}
	// gitgraph owns what a repository key *is* — the canonical absolute common
	// dir, so every worktree of one repository agrees. A second implementation
	// here would drift from the key the lane matches receipts against.
	f.RepositoryKey = gitgraph.Key(cwd)
	if f.RepositoryKey == "" {
		return f
	}
	f.RequestedRevision = gitRevision(cwd)
	f.Clean, f.UncleanReason = artifactClean(cwd)
	return f
}

// beginDeployReceipt writes the attempt as `started` and returns its id, or ""
// when it could not write (callers ignore the failure). A receipt no reader can
// attribute is not evidence: ADR-0170 requires a non-empty repositoryKey, so a
// deploy from outside a repository records nothing rather than leaving a file
// every observer has to reject.
func beginDeployReceipt(dataDir string, facts deployFacts) string {
	key := sanitize(facts.RepositoryKey)
	if key == "" {
		return ""
	}
	dir, err := ensureReceiptDir(dataDir)
	if err != nil {
		return ""
	}
	id, err := newReceiptID()
	if err != nil {
		return ""
	}
	rec := deployReceipt{
		SchemaVersion:     receiptSchemaVersion,
		ID:                id,
		Kind:              receiptKindDeploy,
		RepositoryKey:     key,
		Actor:             sanitize(facts.Actor),
		StartedAt:         nowStamp(),
		Outcome:           receiptOutcomeStarted,
		RequestedRevision: sanitize(facts.RequestedRevision),
		BuiltRevision:     sanitize(facts.BuiltRevision),
		RevisionBefore:    sanitize(facts.RevisionBefore),
		BootBefore:        sanitize(facts.BootBefore),
		Clean:             facts.Clean,
		Error:             sanitize(facts.UncleanReason),
	}
	if err := writeReceipt(dir, id, rec); err != nil {
		return ""
	}
	return id
}

// finishDeployReceipt replaces the attempt's record with its outcome. It
// re-reads what begin wrote because startedAt is when the deploy started, not
// when this call ran: an attempt that ends before it began is exactly the kind
// of record a reader has to reject. An empty or foreign id, or a record that
// is already gone, leaves nothing to finish.
func finishDeployReceipt(dataDir, id, outcome, observedRevision, bootAfter, errSummary string) {
	if !receiptIDOK(id) {
		return
	}
	dir, err := ensureReceiptDir(dataDir)
	if err != nil {
		return
	}
	raw, err := os.ReadFile(filepath.Join(dir, id+".json"))
	if err != nil {
		return
	}
	var rec deployReceipt
	if err := json.Unmarshal(raw, &rec); err != nil || rec.ID != id {
		return
	}
	if o := sanitize(outcome); o != "" {
		rec.Outcome = o
	}
	rec.FinishedAt = nowStamp()
	rec.ObservedRevision = sanitize(observedRevision)
	rec.BootAfter = sanitize(bootAfter)
	// A caller with nothing to report (a passed deploy) leaves whatever begin
	// recorded, so an unclean artifact stays explained. A failed deploy's own
	// error is the attempt's, and replaces it.
	if summary := sanitize(errSummary); summary != "" {
		rec.Error = summary
	}
	_ = writeReceipt(dir, id, rec)
}

// receiptDir is where attempts live, beside the deploy log of ADR-0085.
func receiptDir(dataDir string) string {
	return filepath.Join(dataDir, "var", "delivery")
}

// ensureReceiptDir creates the directory (0700) and refuses to walk through a
// symlinked var/ or delivery/. The receipt names *this* machine's deployment,
// and a link would let the record be written — and read — somewhere the data
// dir does not own, which is what ADR-0170 rejects rather than follows.
func ensureReceiptDir(dataDir string) (string, error) {
	if strings.TrimSpace(dataDir) == "" {
		return "", errors.New("delivery receipt: no data dir")
	}
	dir := dataDir
	for _, name := range []string{"var", "delivery"} {
		next := filepath.Join(dir, name)
		fi, err := os.Lstat(next)
		if err != nil {
			if !errors.Is(err, fs.ErrNotExist) {
				return "", err
			}
			if err := os.Mkdir(next, 0o700); err != nil {
				return "", err
			}
			dir = next
			continue
		}
		if fi.Mode()&os.ModeSymlink != 0 {
			return "", fmt.Errorf("delivery receipt: %s is a symlink", next)
		}
		if !fi.IsDir() {
			return "", fmt.Errorf("delivery receipt: %s is not a directory", next)
		}
		dir = next
	}
	return dir, nil
}

// writeReceipt marshals the record and lands it in dir.
func writeReceipt(dir, id string, rec deployReceipt) error {
	raw, err := json.Marshal(rec)
	if err != nil {
		return err
	}
	return writeReceiptFile(dir, id+".json", raw)
}

// writeReceiptFile writes name into dir atomically: a temp file beside it and
// then a rename, so a reader polling the directory never sees a half-written
// record — the `started` state is written by a completed write, never by a
// partial one. Deliberately no fsync: a receipt is evidence about a deploy, not
// state the next boot depends on, and the cost would land on every deploy.
func writeReceiptFile(dir, name string, raw []byte) error {
	// O_EXCL: the temp name is derived from the receipt id, and refusing an
	// existing path keeps a leftover (or a planted link) from being written
	// through.
	tmp := filepath.Join(dir, name+".tmp")
	f, err := os.OpenFile(tmp, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	_, werr := f.Write(raw)
	if cerr := f.Close(); werr == nil {
		werr = cerr
	}
	if werr != nil {
		_ = os.Remove(tmp)
		return werr
	}
	if err := os.Rename(tmp, filepath.Join(dir, name)); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}

// newReceiptID is a RFC 4122 v4 UUID from crypto/rand: random, so two deploys
// starting in the same instant cannot collide on one file name.
func newReceiptID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	b[6] = b[6]&0x0f | 0x40 // version 4
	b[8] = b[8]&0x3f | 0x80 // RFC 4122 variant
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16]), nil
}

// receiptIDOK reports whether id is the shape newReceiptID produces. It becomes
// a file name, so anything else — a separator, a relative name — is refused
// instead of joined into a path.
func receiptIDOK(id string) bool {
	if len(id) != 36 {
		return false
	}
	for i, r := range id {
		switch i {
		case 8, 13, 18, 23:
			if r != '-' {
				return false
			}
		default:
			if (r < '0' || r > '9') && (r < 'a' || r > 'f') {
				return false
			}
		}
	}
	return true
}

// discoveryIdentity is the running daemon's identity from <data>/server.json:
// revision is the artifact it was built from, boot names the process. Absent,
// unreadable or malformed answers "" — before the first start, and after it
// stops, there is no daemon to name.
func discoveryIdentity(dataDir string) (revision, boot string) {
	raw, err := os.ReadFile(filepath.Join(dataDir, "server.json"))
	if err != nil {
		return "", ""
	}
	var disc struct {
		Revision string `json:"revision"`
		Boot     string `json:"boot"`
	}
	if err := json.Unmarshal(raw, &disc); err != nil {
		return "", ""
	}
	return sanitize(disc.Revision), sanitize(disc.Boot)
}

// awaitDaemonIdentity samples server.json until a daemon other than the one
// that was running before answers. `systemctl restart` returns before the new
// process has bound and rewritten the file, so the first sample is usually the
// daemon that was just replaced — reading once would record the attempt as
// having observed the revision it retired. A daemon that never comes back
// leaves the pre-restart values, which is its own answer: the same boot, so
// nothing new is serving.
func awaitDaemonIdentity(dataDir, revisionBefore, bootBefore string) (revision, boot string) {
	deadline := time.Now().Add(receiptDiscoveryWait)
	for {
		revision, boot = discoveryIdentity(dataDir)
		switch {
		case boot != "" && boot != bootBefore:
			return revision, boot
		case boot == "" && revision != "" && revision != revisionBefore:
			// A daemon too old to publish a boot still names its artifact.
			return revision, boot
		}
		if time.Now().After(deadline) {
			return revision, boot
		}
		time.Sleep(receiptDiscoveryStep)
	}
}

// gitOutput runs git in dir, bounded by receiptGitTimeout.
func gitOutput(dir string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), receiptGitTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	out, err := cmd.Output()
	return strings.TrimSpace(string(out)), err
}

// gitRevision is the deploying checkout's HEAD: the revision the deploy asks
// for. A repository with no commits yet has none, and says so with "".
func gitRevision(dir string) string {
	out, err := gitOutput(dir, "rev-parse", "HEAD")
	if err != nil {
		return ""
	}
	return out
}

// artifactClean answers "does the deploying checkout have uncommitted work?",
// with ONE rule, deliberately: clean is true only when `git status --porcelain`
// succeeded and printed nothing. A status that fails — no git, a broken index,
// the timeout above — leaves the question open, and an open question is not a
// clean artifact: claiming clean without having looked is the one lie that
// would let the lane show a green light for a tree nobody checked (ADR-0170).
// The reason rides `error` in the receipt, so the false is explained.
func artifactClean(dir string) (clean bool, reason string) {
	out, err := gitOutput(dir, "status", "--porcelain")
	if err != nil {
		return false, "working tree not read: " + err.Error()
	}
	return out == "", ""
}

// nowStamp is the receipt's time format: RFC3339Nano, UTC, so two attempts
// from machines in different zones still sort.
func nowStamp() string { return time.Now().UTC().Format(time.RFC3339Nano) }

// sanitize makes one string safe to put on the wire: control characters are
// dropped — a receipt is one record whose fields a reader parses as evidence,
// and a newline inside a path or a term id would make it read as something it
// is not — and the result is bounded, never splitting a rune.
func sanitize(s string) string {
	if len(s) <= receiptFieldMax && strings.IndexFunc(s, unicode.IsControl) < 0 {
		return s
	}
	var b strings.Builder
	b.Grow(min(len(s), receiptFieldMax))
	for _, r := range s {
		if unicode.IsControl(r) {
			continue
		}
		if b.Len()+utf8.RuneLen(r) > receiptFieldMax {
			break
		}
		b.WriteRune(r)
	}
	return b.String()
}
