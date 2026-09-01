package store

import "testing"

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
// migration 014 runs. Re-applies the real embedded migration (not a
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

	// Undo migration 014 as if it had never run, then let Store.migrate
	// re-apply it for real.
	if _, err := s.db.Exec(`DROP TABLE agent_sessions`); err != nil {
		t.Fatal(err)
	}
	if _, err := s.db.Exec(`DELETE FROM schema_migrations WHERE version = 14`); err != nil {
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
