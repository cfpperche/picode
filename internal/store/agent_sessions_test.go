package store

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/cfpperche/picode/internal/session"
)

func TestAgentSessionsHistory(t *testing.T) {
	s := openTest(t)
	proj := t.TempDir()
	_, agent, err := addWorkspaceWithAgent(s, "App", proj)
	if err != nil {
		t.Fatal(err)
	}

	// A fresh pending session: minted before spawn, resolvable later.
	sid := s.NewPendingAgentSession(agent.ID)
	if sid == "" {
		t.Fatal("NewPendingAgentSession returned empty id")
	}
	keys, err := s.AgentSessionKeys(agent.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !keys.IDs[sid] {
		t.Fatalf("pending session id not historized: %+v", keys)
	}
	if len(keys.Paths) != 0 {
		t.Fatalf("pending session should have no path yet: %+v", keys)
	}

	// Resolving fills in the path once the file exists on disk.
	s.ResolveAgentSessionID(agent.ID, sid, "/x/session.jsonl")
	keys, err = s.AgentSessionKeys(agent.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !keys.Paths["/x/session.jsonl"] {
		t.Fatalf("resolved path missing: %+v", keys)
	}
	// Resolve never touches an already-resolved row.
	s.ResolveAgentSessionID(agent.ID, sid, "/should-not-overwrite.jsonl")
	keys, _ = s.AgentSessionKeys(agent.ID)
	if keys.Paths["/should-not-overwrite.jsonl"] {
		t.Fatalf("resolve overwrote an already-resolved path: %+v", keys)
	}

	// A resume/fork/clone/adopt/import historizes directly by path.
	if err := s.RecordAgentSessionPath(agent.ID, "/y/other.jsonl"); err != nil {
		t.Fatal(err)
	}
	// Idempotent: recording the same path twice neither errors nor duplicates.
	if err := s.RecordAgentSessionPath(agent.ID, "/y/other.jsonl"); err != nil {
		t.Fatal(err)
	}
	keys, err = s.AgentSessionKeys(agent.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(keys.Paths) != 2 {
		t.Fatalf("paths = %+v, want 2 (resolved + recorded)", keys.Paths)
	}

	// Deleting the agent cascades (ON DELETE CASCADE, matching tasks/events).
	if err := s.DeleteAgent(agent.ID); err != nil {
		t.Fatal(err)
	}
	var count int
	if err := s.db.QueryRow(`SELECT COUNT(1) FROM agent_sessions WHERE agent_id = ?`, agent.ID).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("agent_sessions not cascaded: %d rows remain", count)
	}
}

func TestUpdateAgentHistorizesSessionPath(t *testing.T) {
	s := openTest(t)
	proj := t.TempDir()
	_, agent, err := addWorkspaceWithAgent(s, "App", proj)
	if err != nil {
		t.Fatal(err)
	}

	path := "/z/resumed.jsonl"
	if _, err := s.UpdateAgent(agent.ID, AgentPatch{SessionPath: &path}); err != nil {
		t.Fatal(err)
	}
	keys, err := s.AgentSessionKeys(agent.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !keys.Paths[path] {
		t.Fatalf("UpdateAgent did not historize session_path: %+v", keys)
	}

	// Clearing session_path (handleNewSession's "" patch, emptyToNil ->
	// nil) must not historize an empty path.
	empty := ""
	if _, err := s.UpdateAgent(agent.ID, AgentPatch{SessionPath: &empty}); err != nil {
		t.Fatal(err)
	}
	keys, err = s.AgentSessionKeys(agent.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(keys.Paths) != 1 {
		t.Fatalf("clearing session_path historized something: %+v", keys.Paths)
	}
}

// TestMigrationBackfillsExistingSessionPath simulates upgrading a
// pre-ADR-0039 database: an agent already has a session_path when
// migration 015 runs. Re-applies the real embedded migration (not a
// hand-copied approximation of it) against that state.
func TestMigrationBackfillsExistingSessionPath(t *testing.T) {
	s := openTest(t)
	proj := t.TempDir()
	_, agent, err := addWorkspaceWithAgent(s, "App", proj)
	if err != nil {
		t.Fatal(err)
	}
	path := "/pre-existing/session.jsonl"
	if _, err := s.db.Exec(`UPDATE agents SET session_path = ? WHERE id = ?`, path, agent.ID); err != nil {
		t.Fatal(err)
	}

	// Undo migration 015 as if it had never run, then let Store.migrate
	// re-apply it for real.
	if _, err := s.db.Exec(`DROP TABLE agent_sessions`); err != nil {
		t.Fatal(err)
	}
	if _, err := s.db.Exec(`DELETE FROM schema_migrations WHERE version = 15`); err != nil {
		t.Fatal(err)
	}
	if err := s.migrate(); err != nil {
		t.Fatalf("re-migrate: %v", err)
	}

	keys, err := s.AgentSessionKeys(agent.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !keys.Paths[path] {
		t.Fatalf("backfill did not historize pre-existing session_path: %+v", keys)
	}
}

// TestResolvePendingAgentSession exercises the adopt-at-spawn lookup
// (ADR-0053): a pending --session-id from an earlier run (ADR-0039)
// whose file has since appeared in the agent's private dir (ADR-0040)
// resolves to that file instead of the caller minting a competitor.
//
//	pending id | file in private dir | return | row path     | session_path
//	-----------+---------------------+--------+--------------+---------------------
//	yes        | yes (newest wins)   | path   | resolved     | backfilled when empty
//	yes        | no                  | ""     | untouched    | untouched
//	yes        | yes                 | path   | resolved     | kept (already set)
func TestResolvePendingAgentSession(t *testing.T) {
	s := openTest(t)
	old := session.TestRoot
	session.TestRoot = t.TempDir()
	t.Cleanup(func() { session.TestRoot = old })

	proj := t.TempDir()
	ws, agent, err := addWorkspaceWithAgent(s, "App", proj)
	if err != nil {
		t.Fatal(err)
	}

	// Nothing pending: no ids, no files — the plain fresh-agent case.
	if got := s.ResolvePendingAgentSession(agent.ID); got != "" {
		t.Fatalf("resolve with nothing pending = %q, want \"\"", got)
	}

	// Pending id whose file exists: adopted, newest wins when two pend.
	older := s.NewPendingAgentSession(agent.ID)
	newer := s.NewPendingAgentSession(agent.ID)
	if older == "" || newer == "" {
		t.Fatal("NewPendingAgentSession returned empty id")
	}
	dir := session.AgentDir(agent.ID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeSessionFile := func(name, sid string, mod time.Time) string {
		t.Helper()
		p := filepath.Join(dir, name)
		body := `{"type":"session","id":"` + sid + `","cwd":"` + proj + `"}` + "\n"
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.Chtimes(p, mod, mod); err != nil {
			t.Fatal(err)
		}
		return p
	}
	oldPath := writeSessionFile("old.jsonl", older, time.Now().Add(-time.Hour))
	newPath := writeSessionFile("new.jsonl", newer, time.Now())

	got := s.ResolvePendingAgentSession(agent.ID)
	if got != newPath {
		t.Fatalf("resolve = %q, want newest %q (over %q)", got, newPath, oldPath)
	}
	keys, err := s.AgentSessionKeys(agent.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !keys.Paths[newPath] || !keys.Paths[oldPath] {
		t.Fatalf("rows not resolved: %+v", keys.Paths)
	}
	a, err := s.GetAgent(agent.ID)
	if err != nil {
		t.Fatal(err)
	}
	if a.SessionPath == nil || *a.SessionPath != newPath {
		t.Fatalf("session_path = %v, want %q", a.SessionPath, newPath)
	}

	// An agent whose current session is already set keeps it.
	sid := s.NewPendingAgentSession(agent.ID)
	keep := writeSessionFile("third.jsonl", sid, time.Now())
	if got := s.ResolvePendingAgentSession(agent.ID); got != keep {
		t.Fatalf("resolve = %q, want %q", got, keep)
	}
	a, err = s.GetAgent(agent.ID)
	if err != nil {
		t.Fatal(err)
	}
	if a.SessionPath == nil || *a.SessionPath != newPath {
		t.Fatalf("resolve overwrote a live session_path: %v", a.SessionPath)
	}

	// Pending id with no file on disk yet: nothing to adopt.
	fresh, err := s.AddAgent(ws.ID, "Fresh", proj)
	if err != nil {
		t.Fatal(err)
	}
	if sid := s.NewPendingAgentSession(fresh.ID); sid == "" {
		t.Fatal("mint failed")
	}
	if got := s.ResolvePendingAgentSession(fresh.ID); got != "" {
		t.Fatalf("resolve without a file = %q, want \"\"", got)
	}
}

// Sealing (the explicit "new session" flow) records every pending id
// that already has a file, so adoption can never resurrect the thread
// the user just abandoned — while leaving the fileless pendings (and the
// agent's pointer) alone. The sealed session stays owned: still listed,
// still resumable.
func TestSealPendingAgentSessions(t *testing.T) {
	s := openTest(t)
	old := session.TestRoot
	session.TestRoot = t.TempDir()
	t.Cleanup(func() { session.TestRoot = old })

	proj := t.TempDir()
	_, agent, err := addWorkspaceWithAgent(s, "App", proj)
	if err != nil {
		t.Fatal(err)
	}
	dir := session.AgentDir(agent.ID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	withFile := s.NewPendingAgentSession(agent.ID)
	withoutFile := s.NewPendingAgentSession(agent.ID)
	if withFile == "" || withoutFile == "" {
		t.Fatal("NewPendingAgentSession returned empty id")
	}
	p := filepath.Join(dir, "sealed.jsonl")
	body := `{"type":"session","id":"` + withFile + `","cwd":"` + proj + `"}` + "\n"
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}

	s.SealPendingAgentSessions(agent.ID)

	keys, err := s.AgentSessionKeys(agent.ID)
	if err != nil {
		t.Fatal(err)
	}
	// Ownership moved to the path row; the id attribution is gone —
	// that is what makes the thread unadoptable.
	if !keys.Paths[p] {
		t.Fatalf("sealed session no longer owned: %+v", keys)
	}
	if keys.IDs[withFile] {
		t.Fatalf("sealed id row should have been replaced by path ownership: %+v", keys)
	}
	pending, err := s.PendingAgentSessionIDs(agent.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(pending) != 1 || pending[0] != withoutFile {
		t.Fatalf("pending after seal = %v, want only the fileless %q", pending, withoutFile)
	}
	a, err := s.GetAgent(agent.ID)
	if err != nil {
		t.Fatal(err)
	}
	if a.SessionPath != nil && *a.SessionPath != "" {
		t.Fatalf("seal must not set the pointer, got %q", *a.SessionPath)
	}
	// The point of sealing: adoption can no longer pick the thread up.
	if got := s.ResolvePendingAgentSession(agent.ID); got != "" {
		t.Fatalf("adoption after seal = %q, want \"\"", got)
	}
}

// The live collision: a session becomes current through a path-recording
// flow (resume, or a pointer backfill), so the table holds BOTH the
// pending id row and a path row for the same file. Sealing must move
// ownership to the path row anyway — a plain UPDATE would collide on
// UNIQUE (agent_id, session_path), leave the id row pending, and the
// next spawn would re-adopt the thread an explicit New just abandoned.
func TestSealPendingWithSiblingPathRow(t *testing.T) {
	s := openTest(t)
	old := session.TestRoot
	session.TestRoot = t.TempDir()
	t.Cleanup(func() { session.TestRoot = old })

	proj := t.TempDir()
	_, agent, err := addWorkspaceWithAgent(s, "App", proj)
	if err != nil {
		t.Fatal(err)
	}
	dir := session.AgentDir(agent.ID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	sid := s.NewPendingAgentSession(agent.ID)
	p := filepath.Join(dir, "twin.jsonl")
	body := `{"type":"session","id":"` + sid + `","cwd":"` + proj + `"}` + "\n"
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	// The pointer went through a path-recording flow: both rows exist.
	if err := s.RecordAgentSessionPath(agent.ID, p); err != nil {
		t.Fatal(err)
	}
	empty := ""
	if _, err := s.UpdateAgent(agent.ID, AgentPatch{SessionPath: &empty}); err != nil {
		t.Fatal(err)
	}

	s.SealPendingAgentSessions(agent.ID)

	// The pending row is gone; ownership lives on the path row, and
	// adoption can no longer pick the file up.
	ids, err := s.PendingAgentSessionIDs(agent.ID)
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range ids {
		if id == sid {
			t.Fatalf("pending id row survived the seal")
		}
	}
	keys, err := s.AgentSessionKeys(agent.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !keys.Paths[p] {
		t.Fatalf("sealed session lost its path ownership: %+v", keys.Paths)
	}
	if got := s.ResolvePendingAgentSession(agent.ID); got != "" {
		t.Fatalf("adoption after seal = %q, want \"\"", got)
	}
}
