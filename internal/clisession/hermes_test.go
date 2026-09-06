package clisession

import (
	"database/sql"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	_ "modernc.org/sqlite"
)

func seedHermesDB(t *testing.T, dir string, extra string) string {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "state.db")
	db, err := sql.Open("sqlite", "file:"+path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	_, err = db.Exec(`CREATE TABLE sessions (
		id TEXT PRIMARY KEY,
		source TEXT NOT NULL,
		title TEXT,
		model TEXT,
		cwd TEXT,
		git_repo_root TEXT,
		started_at REAL NOT NULL,
		ended_at REAL,
		message_count INTEGER DEFAULT 0,
		archived INTEGER DEFAULT 0
	);` + extra)
	if err != nil {
		t.Fatal(err)
	}
	return path
}

func insertHermes(t *testing.T, path, id, source, title, cwd, repo string, started float64, messages, archived int) {
	t.Helper()
	db, err := sql.Open("sqlite", "file:"+path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	_, err = db.Exec(`INSERT INTO sessions (id, source, title, model, cwd, git_repo_root, started_at, message_count, archived)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		id, source, title, "nous/hermes", cwd, repo, started, messages, archived)
	if err != nil {
		t.Fatal(err)
	}
}

func TestHermesListDecisionTable(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("HERMES_HOME", "")
	root := filepath.Join(home, ".hermes")
	path := seedHermesDB(t, root, "")
	insertHermes(t, path, "20260906_101500_abc123", "cli", "Auth refactor", "/home/goat/proj", "", 1788720000, 4, 0)
	insertHermes(t, path, "20260906_111500_tui001", "tui", "TUI session", "/home/goat/proj", "", 1788723600, 2, 0)
	insertHermes(t, path, "20260906_090000_ghost", "cli", "Ghost", "/home/goat/proj", "", 1788710000, 0, 0)
	insertHermes(t, path, "20260906_080000_tele", "telegram", "Bot chat", "/home/goat/proj", "", 1788700000, 8, 0)
	insertHermes(t, path, "20260906_070000_acp", "acp", "Editor", "/home/goat/proj", "", 1788690000, 3, 0)
	insertHermes(t, path, "20260906_060000_arch", "cli", "Old", "/home/goat/proj", "", 1788680000, 5, 1)
	insertHermes(t, path, "20260906_050000_nocwd", "cli", "No folder", "", "", 1788670000, 2, 0)
	insertHermes(t, path, "20260906_040000_repo", "cli", "Repo only", "", "/home/goat/other", 1788660000, 1, 0)

	got, err := HermesSource{}.List("")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 {
		t.Fatalf("got %d rows, want 3 (cli+tui with folder and messages): %+v", len(got), got)
	}
	if got[0].ID != "20260906_111500_tui001" || got[0].Name != "TUI session" {
		t.Errorf("newest = %+v", got[0])
	}
	if !reflect.DeepEqual(got[0].ResumeArgs, []string{"--resume", "20260906_111500_tui001"}) {
		t.Errorf("resumeArgs = %v", got[0].ResumeArgs)
	}
	if got[0].Preview != "TUI session" || got[0].Size != 0 || got[0].Cost != 0 {
		t.Errorf("preview/size/cost = %+v", got[0])
	}
	if got[0].Model != "nous/hermes" || got[0].Messages != 2 {
		t.Errorf("model/messages = %+v", got[0])
	}

	var ids []string
	for _, s := range got {
		ids = append(ids, s.ID)
	}
	if !reflect.DeepEqual(ids, []string{"20260906_111500_tui001", "20260906_101500_abc123", "20260906_040000_repo"}) {
		t.Errorf("ids = %v", ids)
	}
	if got[2].Cwd != "/home/goat/other" {
		t.Errorf("git_repo_root fallback cwd = %q", got[2].Cwd)
	}

	scoped, err := HermesSource{}.List("/home/goat/proj")
	if err != nil || len(scoped) != 2 {
		t.Fatalf("cwd filter = %d, %v want 2", len(scoped), err)
	}
	empty, err := HermesSource{}.List("/elsewhere")
	if err != nil || len(empty) != 0 {
		t.Fatalf("other folder = %d, %v", len(empty), err)
	}
}

func TestHermesListMissingAndHERMESHome(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("HERMES_HOME", "")
	got, err := HermesSource{}.List("")
	if err != nil || len(got) != 0 {
		t.Fatalf("empty home = %v, %v", got, err)
	}

	alt := t.TempDir()
	path := seedHermesDB(t, alt, "")
	insertHermes(t, path, "20260906_120000_home1", "cli", "From env", "/work", "", 1788730000, 1, 0)
	t.Setenv("HERMES_HOME", alt)
	got, err = HermesSource{}.List("")
	if err != nil || len(got) != 1 || got[0].ID != "20260906_120000_home1" {
		t.Fatalf("HERMES_HOME listing = %+v, %v", got, err)
	}

	// A profiles/ database is ignored: only the active home is read.
	prof := filepath.Join(home, ".hermes", "profiles", "private")
	p := seedHermesDB(t, prof, "")
	insertHermes(t, p, "20260906_130000_priv", "cli", "Private", "/work", "", 1788740000, 9, 0)
	t.Setenv("HERMES_HOME", "")
	got, err = HermesSource{}.List("")
	if err != nil || len(got) != 0 {
		t.Fatalf("profile DB must not be scanned: %+v, %v", got, err)
	}
}

func TestHermesListCorruptIsEmpty(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("HERMES_HOME", "")
	root := filepath.Join(home, ".hermes")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "state.db"), []byte("not a database"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := HermesSource{}.List("")
	if err != nil || len(got) != 0 {
		t.Fatalf("corrupt db = %v, %v; want empty, no error", got, err)
	}
}
