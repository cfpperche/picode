package clisession

import (
	"database/sql"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"testing"
	"time"
)

func seedOpenCodeDB(t *testing.T, dir string) string {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "opencode.db")
	db, err := sql.Open("sqlite", "file:"+path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	_, err = db.Exec(`CREATE TABLE session (
		id TEXT PRIMARY KEY,
		project_id TEXT NOT NULL DEFAULT '',
		parent_id TEXT,
		slug TEXT NOT NULL DEFAULT '',
		directory TEXT NOT NULL DEFAULT '',
		title TEXT NOT NULL DEFAULT '',
		version TEXT NOT NULL DEFAULT '',
		time_created INTEGER NOT NULL,
		time_updated INTEGER NOT NULL,
		time_archived INTEGER,
		agent TEXT,
		model TEXT,
		cost REAL NOT NULL DEFAULT 0
	);
	CREATE TABLE message (
		id TEXT PRIMARY KEY,
		session_id TEXT NOT NULL,
		time_created INTEGER,
		time_updated INTEGER,
		data TEXT
	);`)
	if err != nil {
		t.Fatal(err)
	}
	return path
}

func insertOpenCode(t *testing.T, path, id, parent, directory, title, model string, created, updated, archived int64, messages int) {
	t.Helper()
	db, err := sql.Open("sqlite", "file:"+path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var arch any
	if archived != 0 {
		arch = archived
	}
	_, err = db.Exec(`INSERT INTO session (id, parent_id, slug, directory, title, version, time_created, time_updated, time_archived, model, cost)
		VALUES (?, ?, ?, ?, ?, '1.18.29', ?, ?, ?, ?, 9.99)`,
		id, nullIfEmpty(parent), "slug-"+id, directory, title, created, updated, arch, model)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < messages; i++ {
		if _, err := db.Exec(`INSERT INTO message (id, session_id) VALUES (?, ?)`, id+"-m"+strconv.Itoa(i), id); err != nil {
			t.Fatal(err)
		}
	}
}

func nullIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func TestOpenCodeListDecisionTable(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_DATA_HOME", "")
	root := filepath.Join(home, ".local", "share", "opencode")
	path := seedOpenCodeDB(t, root)

	ms := int64(1787595346906) // 2026-08-24T18:15:46.906Z
	newer := ms + 60_000
	insertOpenCode(t, path, "ses_keep1", "", "/home/goat/proj", "Auth refactor", `{"id":"big-pickle","providerID":"opencode"}`, ms, newer, 0, 4)
	insertOpenCode(t, path, "ses_keep2", "", "/home/goat/proj", "TUI session", `{"id":"glm-5.2","providerID":"zai"}`, ms, ms+30_000, 0, 2)
	insertOpenCode(t, path, "ses_ghost", "", "/home/goat/proj", "Ghost", `{"id":"x"}`, ms, ms, 0, 0)
	insertOpenCode(t, path, "ses_child", "ses_keep1", "/home/goat/proj", "Explore subagent", `{"id":"x"}`, ms, ms, 0, 3)
	insertOpenCode(t, path, "ses_arch", "", "/home/goat/proj", "Old", `{"id":"x"}`, ms, ms, ms, 5)
	insertOpenCode(t, path, "ses_nodir", "", "", "No folder", `{"id":"x"}`, ms, ms, 0, 2)
	insertOpenCode(t, path, "ses_other", "", "/home/goat/other", "Other folder", `{"id":"x"}`, ms, ms+10_000, 0, 1)

	got, err := OpenCodeSource{}.List("")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 {
		t.Fatalf("got %d rows, want 3 (top-level with folder and messages): %+v", len(got), got)
	}
	if got[0].ID != "ses_keep1" || got[0].Name != "Auth refactor" {
		t.Errorf("newest = %+v", got[0])
	}
	if !reflect.DeepEqual(got[0].ResumeArgs, []string{"--session", "ses_keep1"}) {
		t.Errorf("resumeArgs = %v", got[0].ResumeArgs)
	}
	if got[0].Preview != "Auth refactor" || got[0].Size != 0 || got[0].Cost != 0 {
		t.Errorf("preview/size/cost = %+v", got[0])
	}
	if got[0].Model != "opencode/big-pickle" || got[0].Messages != 4 {
		t.Errorf("model/messages = %+v", got[0])
	}
	wantTime := time.UnixMilli(newer).UTC().Format(time.RFC3339)
	if got[0].UpdatedAt != wantTime {
		t.Errorf("UpdatedAt = %q, want %q (milliseconds)", got[0].UpdatedAt, wantTime)
	}

	var ids []string
	for _, s := range got {
		ids = append(ids, s.ID)
	}
	if !reflect.DeepEqual(ids, []string{"ses_keep1", "ses_keep2", "ses_other"}) {
		t.Errorf("ids = %v", ids)
	}

	scoped, err := OpenCodeSource{}.List("/home/goat/proj")
	if err != nil || len(scoped) != 2 {
		t.Fatalf("cwd filter = %d, %v want 2", len(scoped), err)
	}
	empty, err := OpenCodeSource{}.List("/elsewhere")
	if err != nil || len(empty) != 0 {
		t.Fatalf("other folder = %d, %v", len(empty), err)
	}
}

func TestOpenCodeListMissingAndXDGDataHome(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_DATA_HOME", "")
	got, err := OpenCodeSource{}.List("")
	if err != nil || len(got) != 0 {
		t.Fatalf("empty home = %v, %v", got, err)
	}

	alt := t.TempDir()
	path := seedOpenCodeDB(t, filepath.Join(alt, "opencode"))
	insertOpenCode(t, path, "ses_xdg", "", "/work", "From env", `{"id":"m","providerID":"p"}`, 1787595346906, 1787595346906, 0, 1)
	t.Setenv("XDG_DATA_HOME", alt)
	got, err = OpenCodeSource{}.List("")
	if err != nil || len(got) != 1 || got[0].ID != "ses_xdg" {
		t.Fatalf("XDG_DATA_HOME listing = %+v, %v", got, err)
	}
}

func TestOpenCodeListCorruptIsEmpty(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_DATA_HOME", "")
	root := filepath.Join(home, ".local", "share", "opencode")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "opencode.db"), []byte("not a database"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := OpenCodeSource{}.List("")
	if err != nil || len(got) != 0 {
		t.Fatalf("corrupt db = %v, %v; want empty, no error", got, err)
	}
}
