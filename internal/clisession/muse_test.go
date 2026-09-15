package clisession

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"
)

// seedMuseDB writes a session-index.db with the shape Muse Code 1.2.1
// ships (columns probed from a real ~/.local/share/muse/session-index.db).
func seedMuseDB(t *testing.T, dir string) string {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "session-index.db")
	db, err := sql.Open("sqlite", "file:"+path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(`CREATE TABLE sessions (
		session_id TEXT PRIMARY KEY,
		session_stream_id TEXT NOT NULL,
		session_dir TEXT NOT NULL,
		session_log_path TEXT NOT NULL UNIQUE,
		layout TEXT NOT NULL,
		workspace_root TEXT,
		workspace_key TEXT,
		provider_id TEXT,
		model_id TEXT,
		git_branch TEXT,
		title TEXT NOT NULL,
		first_user_prompt TEXT,
		search_text TEXT NOT NULL,
		created_at_us INTEGER,
		updated_at_us INTEGER,
		prompt_count INTEGER NOT NULL DEFAULT 0,
		status TEXT NOT NULL,
		status_rank INTEGER NOT NULL
	)`); err != nil {
		t.Fatal(err)
	}
	// A real session: logged prompts, a workspace and a model.
	live := filepath.Join(dir, "sessions", "2026", "09", "14", "a1", "session.jsonl")
	if err := os.MkdirAll(filepath.Dir(live), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(live, []byte("{\"record_type\":\"event\"}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	// 2026-09-14T23:39:04Z and one minute later, in microseconds.
	if _, err := db.Exec(`INSERT INTO sessions
		(session_id, session_stream_id, session_dir, session_log_path, layout, workspace_root, workspace_key, provider_id, model_id, git_branch,
		 title, first_user_prompt, search_text, created_at_us, updated_at_us, prompt_count, status, status_rank)
		VALUES ('a1b2c3d4-0000-0000-0000-000000000001','a1b2c3d4','/d','` + live + `','session_jsonl','/home/goat/proj','/home/goat/proj','meta','muse-spark-1.3-contributor',NULL,
		 'Ola bem vindo ao PiCode','Ola bem vindo ao PiCode','idx',1789439944675204,1789440000000000,6,'valid',0)`); err != nil {
		t.Fatal(err)
	}
	// The launcher's own empty session: no workspace, no prompt.
	if _, err := db.Exec(`INSERT INTO sessions
		(session_id, session_stream_id, session_dir, session_log_path, layout, workspace_root, workspace_key, provider_id, model_id, git_branch,
		 title, first_user_prompt, search_text, created_at_us, updated_at_us, prompt_count, status, status_rank)
		VALUES ('a1b2c3d4-0000-0000-0000-000000000002','a1b2c3d4','/d2','/nope/session.jsonl','session_jsonl',NULL,NULL,NULL,NULL,NULL,
		 'New session','None','idx',1789438041384116,1789438046626751,0,'missing_metadata',3)`); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestMuseSourceList(t *testing.T) {
	dir := t.TempDir()
	MuseTestDB = seedMuseDB(t, dir)
	t.Cleanup(func() { MuseTestDB = "" })

	got, err := MuseSource{}.List("")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("sessions = %d, want 1 (the empty session stays out): %+v", len(got), got)
	}
	s := got[0]
	if s.CLI != "muse" || s.ID != "a1b2c3d4-0000-0000-0000-000000000001" {
		t.Fatalf("identity: %+v", s)
	}
	if s.Cwd != "/home/goat/proj" || s.Model != "muse-spark-1.3-contributor" {
		t.Fatalf("folder/model: %+v", s)
	}
	if s.Name != "Ola bem vindo ao PiCode" || s.Preview != "Ola bem vindo ao PiCode" || s.Messages != 6 {
		t.Fatalf("listing text: %+v", s)
	}
	if s.CreatedAt != "2026-09-15T02:39:04Z" || s.UpdatedAt != "2026-09-15T02:40:00Z" {
		t.Fatalf("timestamps: created=%q updated=%q", s.CreatedAt, s.UpdatedAt)
	}
	if len(s.ResumeArgs) != 2 || s.ResumeArgs[0] != "--resume" || s.ResumeArgs[1] != s.ID {
		t.Fatalf("resume args: %v", s.ResumeArgs)
	}
	if s.Size <= 0 {
		t.Fatalf("size not read from the session log: %d", s.Size)
	}

	// The folder filter is exact, like every other source.
	if scoped, _ := (MuseSource{}).List("/home/goat/proj"); len(scoped) != 1 {
		t.Fatalf("scoped = %+v", scoped)
	}
	if empty, _ := (MuseSource{}).List("/elsewhere"); len(empty) != 0 {
		t.Fatalf("elsewhere = %+v", empty)
	}
}

func TestMuseSourceMissingAndLocked(t *testing.T) {
	MuseTestDB = filepath.Join(t.TempDir(), "absent.db")
	t.Cleanup(func() { MuseTestDB = "" })
	got, err := MuseSource{}.List("")
	if err != nil || got != nil {
		t.Fatalf("a machine that never ran muse lists empty: %v %v", got, err)
	}

	// A file that is not a database (a future or truncated index) is empty,
	// never an error: the picker has to render.
	path := filepath.Join(t.TempDir(), "broken.db")
	if err := os.WriteFile(path, []byte("not a database"), 0o600); err != nil {
		t.Fatal(err)
	}
	MuseTestDB = path
	if got, err := (MuseSource{}).List(""); err != nil || got != nil {
		t.Fatalf("broken index = %v %v", got, err)
	}
}

// A narrower index (a future or trimmed schema) degrades to the columns it
// has; a row with no time at all is the one thing this source refuses, since
// the picker orders by it.
func TestMuseSourceWithoutOptionalColumns(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "session-index.db")
	db, err := sql.Open("sqlite", "file:"+path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`CREATE TABLE sessions (session_id TEXT PRIMARY KEY, workspace_root TEXT, prompt_count INTEGER, created_at_us INTEGER, updated_at_us INTEGER)`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO sessions (session_id, workspace_root, prompt_count, created_at_us, updated_at_us) VALUES ('a1','/home/goat/proj',2,1789439944675204,1789440000000000)`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO sessions (session_id, workspace_root, prompt_count, created_at_us, updated_at_us) VALUES ('a2','/home/goat/proj',3,NULL,NULL)`); err != nil {
		t.Fatal(err)
	}
	db.Close()
	MuseTestDB = path
	t.Cleanup(func() { MuseTestDB = "" })

	got, err := MuseSource{}.List("")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].ID != "a1" || got[0].Cwd != "/home/goat/proj" {
		t.Fatalf("narrow schema: %+v", got)
	}
	if got[0].Messages != 2 || got[0].Model != "" || got[0].Size != 0 {
		t.Fatalf("missing columns stay empty: %+v", got[0])
	}
}

const museExportFixture = `{"export_schema_version":1,"events":[
{"kind":"record","envelope":{"recorded_at":1789437092000000,"record_type":"event","payload_type":"runtime.session","payload":{"kind":"metadata","record":{"workspace_root":"/w","model_id":"m1"}}}},
{"kind":"record","envelope":{"recorded_at":1789437093000000,"record_type":"event","payload_type":"runtime.user_intent.accepted","payload":{"model_messages":[{"content":[{"kind":"text","text":"fix the race"}]}]}}},
{"kind":"record","envelope":{"recorded_at":1789437094000000,"record_type":"event","payload_type":"runtime.session","payload":{"kind":"run","event":{"kind":"assistant_message_committed","message_id":"m1","text":"On it."}}}},
{"kind":"record","envelope":{"recorded_at":1789437095000000,"record_type":"event","payload_type":"runtime.session","payload":{"kind":"run","event":{"kind":"assistant_tool_calls_committed","tool_calls":[{"call_id":"call_1","name":"read_file","args":"{\"path\":\"x\"}"}]}}}},
{"kind":"record","envelope":{"recorded_at":1789437096000000,"record_type":"event","payload_type":"runtime.session","payload":{"kind":"run","event":{"kind":"tool_result_batch_committed","results":[{"tool_call_id":"call_1","text":"contents"}]}}}},
{"kind":"record","envelope":{"recorded_at":1789437097000000,"record_type":"event","payload_type":"runtime.session","payload":{"kind":"run","event":{"kind":"reasoning_committed","text":"secret plan"}}}},
{"kind":"record","envelope":{"recorded_at":1789437098000000,"record_type":"event","payload_type":"runtime.session","payload":{"kind":"run","event":{"kind":"model_completed","model":"m1"}}}},
{"kind":"record","envelope":{"recorded_at":1789437098500000,"record_type":"event","payload_type":"runtime.session","payload":{"kind":"run","run_id":"r1","event":{"kind":"automated_review_completed","model":{"id":"m1"}}}}},
{"kind":"gap","envelope":{"recorded_at":0,"record_type":"","payload_type":"","payload":null}}
]}`

// seedMuseBin installs a fake muse that answers `export --session … --out
// <file>` by copying the fixture the test names in MUSE_EXPORT_FIXTURE.
func seedMuseBin(t *testing.T, fixture string) {
	t.Helper()
	dir := t.TempDir()
	script := "#!/bin/sh\nout=\"\"\nprev=\"\"\nfor a in \"$@\"; do\nif [ \"$prev\" = \"--out\" ]; then out=\"$a\"; fi\nprev=\"$a\"\ndone\ncp \"$MUSE_EXPORT_FIXTURE\" \"$out\"\n"
	if err := os.WriteFile(filepath.Join(dir, "muse"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	fix := filepath.Join(dir, "export.json")
	if err := os.WriteFile(fix, []byte(fixture), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("MUSE_EXPORT_FIXTURE", fix)
	MuseBin = filepath.Join(dir, "muse")
	t.Cleanup(func() { MuseBin = "" })
}

func TestMuseReadExport(t *testing.T) {
	seedMuseBin(t, museExportFixture)
	tl, err := (MuseSource{}).Read(context.Background(), Ref{ID: "s1", Cwd: ""})
	if err != nil {
		t.Fatal(err)
	}
	if len(tl.Events) != 4 {
		t.Fatalf("events = %d, want user + assistant + call + result", len(tl.Events))
	}
	if tl.Events[0].Role != "user" || tl.Events[0].Text != "fix the race" {
		t.Errorf("first event = %+v", tl.Events[0])
	}
	if tl.Events[1].Role != "assistant" || tl.Events[1].Text != "On it." || tl.Events[1].Model != "m1" {
		t.Errorf("second event = %+v, want the assistant turn with the model", tl.Events[1])
	}
	call, res := tl.Events[2].Call, tl.Events[3].Result
	if call == nil || call.ID != "call_1" || call.Name != "read_file" || res == nil || res.CallID != "call_1" || res.Text != "contents" {
		t.Errorf("tool beats = %+v %+v, want the linked call and result", tl.Events[2], tl.Events[3])
	}
	for _, ev := range tl.Events {
		if ev.Group != 1 {
			t.Errorf("event group = %d, want one turn in group 1", ev.Group)
		}
	}
	if tl.Header.Title != "fix the race" || tl.Header.Cwd != "/w" || tl.Header.Model != "m1" {
		t.Errorf("header = %+v", tl.Header)
	}
	if tl.Manifest.Dropped["muse.malformed"] != 0 {
		t.Errorf("manifest = %+v, the object-model review record must parse", tl.Manifest.Dropped)
	}
	if tl.Manifest.Dropped["thinking"] != 1 || tl.Manifest.Dropped["muse.gap"] != 1 {
		t.Errorf("manifest = %+v, want reasoning and the gap counted", tl.Manifest.Dropped)
	}
}

func TestMuseReadFailures(t *testing.T) {
	if _, err := (MuseSource{}).Read(context.Background(), Ref{}); err == nil {
		t.Error("empty ref reads no error")
	}
	MuseBin = filepath.Join(t.TempDir(), "absent-muse")
	t.Cleanup(func() { MuseBin = "" })
	if _, err := (MuseSource{}).Read(context.Background(), Ref{ID: "s1"}); err == nil {
		t.Error("missing binary reads no error")
	}
	seedMuseBin(t, `{"export_schema_version":2,"events":[]}`)
	if _, err := (MuseSource{}).Read(context.Background(), Ref{ID: "s1"}); err == nil {
		t.Error("future export schema reads no error")
	}
}
