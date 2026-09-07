package server

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"sort"
	"syscall"
	"time"

	"github.com/cfpperche/picode/internal/tmux"
	"github.com/cfpperche/picode/internal/version"
)

// Session forensics (ADR-0085): the flight recorder for tmux session loss.
// On graceful shutdown the daemon records which PiCode-owned sessions were
// alive; on boot it diffs that snapshot against reality and says, in plain
// log lines and a durable report, exactly which sessions died inside the
// restart window. The 2026-09-06 incidents were diagnosed days-of-mtimes
// later; this makes the next one self-reporting.

// snapshotName / reportPrefix live under <dataDir>/var. The snapshot is one
// atomic file (temp + rename); reports accumulate and are pruned to the
// most recent keepReports.
const (
	snapshotName = "shutdown-snapshot.json"
	reportPrefix = "restart-report-"
	keepReports  = 10
)

// ShutdownSnapshot is what was alive when the daemon exited gracefully.
type ShutdownSnapshot struct {
	At       string              `json:"at"`
	Version  string              `json:"version"`
	Graceful bool                `json:"graceful"`
	Sessions []tmux.OwnedSession `json:"sessions"`
}

// RestartReport is the boot-time verdict for one previous shutdown.
type RestartReport struct {
	ShutdownAt string   `json:"shutdownAt"`
	BootedAt   string   `json:"bootedAt"`
	Version    string   `json:"version"`
	Alive      int      `json:"alive"`
	Lost       []string `json:"lost"`
}

// WriteShutdownSnapshot records the live PiCode-owned sessions. Called on
// SIGTERM/SIGINT after the HTTP server is down; never fails the shutdown.
func WriteShutdownSnapshot(ctx context.Context, deps Deps) {
	if deps.Tmux == nil {
		return
	}
	WriteShutdownSnapshotWith(ctx, forensicsDir(deps.DataDir), deps.Tmux)
}

// BootDiff compares the shutdown snapshot with the sessions alive now.
// It logs the verdict, appends a durable report, and returns the set of
// session names that did not survive. Missing or stale snapshot means an
// unclean exit — reported as such, never fatal, never more than a log line.
func BootDiff(ctx context.Context, deps Deps) map[string]bool {
	if deps.Tmux == nil {
		return map[string]bool{}
	}
	return BootDiffWith(ctx, forensicsDir(deps.DataDir), deps.Tmux)
}

// sessionSource is the tmux seam the forensics tests swap out.
type sessionSource interface {
	ListOwned(ctx context.Context) ([]tmux.OwnedSession, error)
	Available() bool
}

// WriteShutdownSnapshotWith records the live sessions into dir.
func WriteShutdownSnapshotWith(ctx context.Context, dir string, src sessionSource) {
	if dir == "" || src == nil || !src.Available() {
		return
	}
	sctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	sessions, err := src.ListOwned(sctx)
	if err != nil {
		log.Printf("forensics: shutdown snapshot skipped: %v", err)
		return
	}
	snap := ShutdownSnapshot{
		At:       time.Now().UTC().Format(time.RFC3339),
		Version:  version.Build(),
		Graceful: true,
		Sessions: sessions,
	}
	writeAtomic(filepath.Join(dir, snapshotName), snap)
	log.Printf("forensics: shutdown snapshot recorded %d session(s)", len(sessions))
}

// BootDiffWith compares the snapshot in dir with src's live sessions.
func BootDiffWith(ctx context.Context, dir string, src sessionSource) map[string]bool {
	lost := map[string]bool{}
	if dir == "" || src == nil || !src.Available() {
		return lost
	}
	sctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	current, err := src.ListOwned(sctx)
	if err != nil {
		log.Printf("forensics: boot diff skipped: %v", err)
		return lost
	}
	alive := map[string]string{} // name -> root cmd
	for _, s := range current {
		alive[s.Name] = s.RootCmd
	}

	raw, err := os.ReadFile(filepath.Join(dir, snapshotName))
	if err != nil {
		// Never silent: a missing snapshot at boot is itself evidence
		// (unclean exit, or the previous daemon was too old to record one).
		log.Printf("forensics: no shutdown snapshot (unclean exit or pre-forensics binary) — %d owned session(s) alive now", len(current))
		return lost
	}
	var snap ShutdownSnapshot
	if err := json.Unmarshal(raw, &snap); err != nil {
		log.Printf("forensics: unreadable shutdown snapshot: %v", err)
		return lost
	}
	for _, s := range snap.Sessions {
		if _, ok := alive[s.Name]; !ok {
			lost[s.Name] = true
		}
	}
	report := RestartReport{
		ShutdownAt: snap.At,
		BootedAt:   time.Now().UTC().Format(time.RFC3339),
		Version:    version.Build(),
		Alive:      len(snap.Sessions) - len(lost),
	}
	names := make([]string, 0, len(lost))
	for name := range lost {
		names = append(names, name)
	}
	sort.Strings(names)
	report.Lost = names
	switch len(lost) {
	case 0:
		log.Printf("forensics: all %d session(s) from the %s shutdown are still alive", len(snap.Sessions), snap.At)
	default:
		log.Printf("forensics: %d of %d session(s) from the %s shutdown did NOT survive: %v",
			len(lost), len(snap.Sessions), snap.At, names)
	}
	writeAtomic(filepath.Join(dir, reportPrefix+report.BootedAt), report)
	pruneReports(dir)
	return lost
}

// forensicsDir creates and returns <dataDir>/var.
func forensicsDir(dataDir string) string {
	if dataDir == "" {
		return ""
	}
	dir := filepath.Join(dataDir, "var")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return ""
	}
	return dir
}

// writeAtomic writes JSON via a temp file + rename, so a crash mid-write
// never leaves a truncated snapshot.
func writeAtomic(path string, v any) {
	raw, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, append(raw, '\n'), 0o600); err != nil {
		return
	}
	_ = os.Rename(tmp, path)
}

// pruneReports keeps only the most recent keepReports restart reports.
func pruneReports(dir string) {
	ents, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	var reports []string
	for _, e := range ents {
		name := e.Name()
		if !e.IsDir() && len(name) > len(reportPrefix) && name[:len(reportPrefix)] == reportPrefix {
			reports = append(reports, name)
		}
	}
	if len(reports) <= keepReports {
		return
	}
	sort.Strings(sortedByTime(reports))
	for _, name := range reports[:len(reports)-keepReports] {
		_ = os.Remove(filepath.Join(dir, name))
	}
}

// sortedByTime is identity: report names embed a UTC RFC3339 timestamp, so
// lexicographic order is chronological order. Named for the review.
func sortedByTime(names []string) []string { return names }

// hupSignal is SIGHUP (kept in one place for the forensics tests).
func hupSignal() os.Signal { return syscall.SIGHUP }
