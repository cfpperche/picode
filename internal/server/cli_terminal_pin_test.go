package server

import (
	"database/sql"
	"encoding/json"
	"github.com/cfpperche/picode/internal/clilaunch"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/cfpperche/picode/internal/clisession"
)

// seedPinMuseDB points the muse reader at a fixture store holding one live
// conversation in cwd. The timestamp runs minutes in the future on purpose:
// terminal creation stamps nanoseconds while the reader resolves
// microseconds, so a same-instant fixture can truncate older than the run
// and trip the no-stealing guard for no reason.
func seedPinMuseDB(t *testing.T, dir, cwd, sid string) {
	t.Helper()
	path := filepath.Join(dir, "session-index.db")
	db, err := sql.Open("sqlite", "file:"+path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(`CREATE TABLE sessions (
		session_id TEXT PRIMARY KEY, session_log_path TEXT, workspace_root TEXT,
		title TEXT, first_user_prompt TEXT, prompt_count INTEGER,
		created_at_us INTEGER, updated_at_us INTEGER, model_id TEXT)`); err != nil {
		t.Fatal(err)
	}
	now := time.Now().Add(5 * time.Minute).UnixMicro()
	if _, err := db.Exec(`INSERT INTO sessions
		(session_id, session_log_path, workspace_root, title, first_user_prompt, prompt_count, created_at_us, updated_at_us, model_id)
		VALUES (?, ?, ?, 'pin probe', 'say plum', 2, ?, ?, 'muse-spark')`,
		sid, filepath.Join(dir, "s.jsonl"), cwd, now-1000000, now); err != nil {
		t.Fatal(err)
	}
	clisession.MuseTestDB = path
	t.Cleanup(func() { clisession.MuseTestDB = "" })
}

// seedPinAgyDB points the agy reader at a fixture store holding one live
// conversation in cwd.
func seedPinAgyDB(t *testing.T, dir, cwd, cid string) {
	t.Helper()
	path := filepath.Join(dir, "conversation_summaries.db")
	db, err := sql.Open("sqlite", "file:"+path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(`CREATE TABLE conversation_summaries (
		conversation_id TEXT PRIMARY KEY, title TEXT, preview TEXT, step_count INTEGER,
		last_modified_time TEXT, workspace_uris TEXT, parent_conversation_id TEXT)`); err != nil {
		t.Fatal(err)
	}
	uris, _ := json.Marshal([]string{"file://" + cwd})
	now := time.Now().Add(5 * time.Minute).UTC().Format(time.RFC3339Nano)
	if _, err := db.Exec(`INSERT INTO conversation_summaries
		(conversation_id, title, preview, step_count, last_modified_time, workspace_uris, parent_conversation_id)
		VALUES (?, 'pin probe', 'say quince', 3, ?, ?, '')`,
		cid, now, string(uris)); err != nil {
		t.Fatal(err)
	}
	clisession.AgyTestDB = path
	t.Cleanup(func() { clisession.AgyTestDB = "" })
}

// seedPinOmpSessions points the omp reader at a fixture sessions root
// holding one live conversation in cwd (bucket encoding: "/" → "-").
func seedPinOmpSessions(t *testing.T, dir, cwd, sid string) {
	t.Helper()
	at := time.Now().Add(5 * time.Minute).UTC().Format("2006-01-02T15:04:05.000Z")
	body := `{"type":"session","version":3,"id":"` + sid + `","timestamp":"` + at + `","cwd":"` + cwd + `"}` + "\n" +
		`{"type":"message","id":"m1","timestamp":"` + at + `","message":{"role":"user","content":[{"type":"text","text":"say plum"}]}}` + "\n"
	bucket := filepath.Join(dir, strings.ReplaceAll(cwd, "/", "-"))
	if err := os.MkdirAll(bucket, 0o755); err != nil {
		t.Fatal(err)
	}
	name := "2026-09-17T00-00-00-000Z_" + sid + ".jsonl"
	if err := os.WriteFile(filepath.Join(bucket, name), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	clisession.OmpTestRoot = dir
	t.Cleanup(func() { clisession.OmpTestRoot = "" })
}

// Stopping a wrapper-less terminal pins the latest session written in its
// folder, so Continue in… and resume work for muse, agy and omp the way the
// runtime pin does for wrapper CLIs. A session predating the terminal
// never pins (no stealing across terminals sharing a folder).
func TestStopPinsWrapperlessSession(t *testing.T) {
	ts, _, home := cleanupServer(t)
	cwd := filepath.Join(home, "proj")
	if err := os.MkdirAll(cwd, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		cli, id string
		seed    func(t *testing.T, dir, cwd, id string)
		resume  []string
	}{
		{"muse", "pin-muse-1", seedPinMuseDB, []string{"resume", "pin-muse-1"}},
		{"agy", "pin-agy-1", seedPinAgyDB, []string{"--conversation", "pin-agy-1"}},
		{"omp", "pin-omp-1", seedPinOmpSessions, []string{"--resume", "pin-omp-1"}},
	} {
		t.Run(tc.cli, func(t *testing.T) {
			// ADR-0184: a launch is an agent's; the CLI runs as `cat`.
			cliRequest(t, ts, "PUT", "/api/clis/"+tc.cli, clilaunch.Config{Executable: "/bin/cat"}, 200)
			created := launchFixture(t, ts, tc.cli, map[string]any{"name": "pin-" + tc.cli, "cwd": cwd}, 201)
			id, _ := created["id"].(string)
			// Seed after the terminal exists: the pin only accepts sessions
			// written after the run started (no stealing across terminals).
			tc.seed(t, t.TempDir(), cwd, tc.id)
			stopped := cliRequestFull(t, ts, "POST", "/api/terminals/"+id+"/launch/stop", map[string]any{"confirm": true})
			if stopped["status"] != "200" {
				t.Fatalf("stop = %v", stopped)
			}
			body, _ := stopped["body"].(map[string]any)
			ls, _ := body["lastSession"].(map[string]any)
			if ls == nil || ls["sessionId"] != tc.id {
				t.Fatalf("%s lastSession = %+v, want %s pinned", tc.cli, ls, tc.id)
			}
			var args []string
			for _, a := range ls["resumeArgs"].([]any) {
				args = append(args, a.(string))
			}
			if !slices.Equal(args, tc.resume) {
				t.Fatalf("%s resumeArgs = %v, want %v", tc.cli, args, tc.resume)
			}
		})
	}
}

// Sessions predating the terminal never pin: with no recent conversation
// in the folder, stop leaves the terminal unpinned instead of stealing a
// stale session from a shared folder.
func TestStopSkipsStaleSession(t *testing.T) {
	ts, _, home := cleanupServer(t)
	cwd := filepath.Join(home, "proj")
	if err := os.MkdirAll(cwd, 0o755); err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "session-index.db")
	db, err := sql.Open("sqlite", "file:"+path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(`CREATE TABLE sessions (
		session_id TEXT PRIMARY KEY, session_log_path TEXT, workspace_root TEXT,
		title TEXT, first_user_prompt TEXT, prompt_count INTEGER,
		created_at_us INTEGER, updated_at_us INTEGER, model_id TEXT)`); err != nil {
		t.Fatal(err)
	}
	// Updated a week ago: older than any terminal created now.
	week := time.Now().Add(-7 * 24 * time.Hour).UnixMicro()
	if _, err := db.Exec(`INSERT INTO sessions
		(session_id, session_log_path, workspace_root, title, first_user_prompt, prompt_count, created_at_us, updated_at_us, model_id)
		VALUES ('stale-1', ?, ?, 'old', 'old prompt', 2, ?, ?, 'm')`,
		filepath.Join(dir, "s.jsonl"), cwd, week, week); err != nil {
		t.Fatal(err)
	}
	clisession.MuseTestDB = path
	t.Cleanup(func() { clisession.MuseTestDB = "" })
	cliRequest(t, ts, "PUT", "/api/clis/muse", clilaunch.Config{Executable: "/bin/cat"}, 200)
	created := launchFixture(t, ts, "muse", map[string]any{"name": "pin-stale", "cwd": cwd}, 201)
	id, _ := created["id"].(string)
	cliRequest(t, ts, "POST", "/api/terminals/"+id+"/launch/stop", map[string]any{"confirm": true}, 200)
	rows := cliRequest(t, ts, "GET", "/api/terminals", nil, 200)["terminals"].([]any)
	for _, raw := range rows {
		if row := raw.(map[string]any); row["id"] == id {
			if ls, ok := row["lastSession"].(map[string]any); ok && ls != nil {
				t.Fatalf("stale session pinned: %+v", ls)
			}
		}
	}
}
