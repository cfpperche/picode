package backup

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/cfpperche/picode/internal/store"
)

func testEngine(t *testing.T) (*Engine, string, string) {
	t.Helper()
	root := t.TempDir()
	data := filepath.Join(root, "data")
	pi := filepath.Join(root, "pi")
	if err := os.MkdirAll(data, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(pi, 0o755); err != nil {
		t.Fatal(err)
	}
	st, err := store.Open(filepath.Join(data, "picode.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	now := time.Date(2026, 8, 26, 15, 0, 0, 0, time.UTC)
	e := &Engine{Store: st, DataDir: data, PiDir: pi, Version: "test", Now: func() time.Time { return now }}
	return e, data, pi
}

func TestValidateDest(t *testing.T) {
	data := t.TempDir()
	pi := t.TempDir()
	if err := ValidateDest("", data, pi); err == nil {
		t.Fatal("empty dest")
	}
	if err := ValidateDest(filepath.Join(data, "inside"), data, pi); err == nil {
		t.Fatal("inside data")
	}
	if err := ValidateDest(data, data, pi); err == nil {
		t.Fatal("data itself")
	}
	if err := ValidateDest(filepath.Join(pi, "x"), data, pi); err == nil {
		t.Fatal("inside pi")
	}
	ok := t.TempDir()
	if err := ValidateDest(ok, data, pi); err != nil {
		t.Fatal(err)
	}
}

func TestDue(t *testing.T) {
	now := time.Date(2026, 8, 26, 15, 0, 0, 0, time.UTC)
	if Due(Settings{}, now) {
		t.Fatal("disabled")
	}
	if Due(Settings{Dir: "/tmp/x", IntervalMin: 60}, now) {
		t.Fatal("dest without start")
	}
	s := Settings{Dir: "/tmp/x", Scheduled: true, Enabled: true, IntervalMin: 60}
	if !Due(s, now) {
		t.Fatal("first run after start")
	}
	s.LastOK = now.Add(-30 * time.Minute).Format(time.RFC3339)
	if Due(s, now) {
		t.Fatal("not due yet")
	}
	s.LastOK = now.Add(-61 * time.Minute).Format(time.RFC3339)
	if !Due(s, now) {
		t.Fatal("should be due")
	}
}

func TestSnapshotRestoreMatrix(t *testing.T) {
	e, data, pi := testEngine(t)
	proj := t.TempDir()
	if _, _, err := e.Store.AddWorkspace("App", proj); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(data, "accounts.json"), []byte(`{"k":1}`), 0o600); err != nil {
		t.Fatal(err)
	}
	pinDir := filepath.Join(data, "pins", "p1")
	if err := os.MkdirAll(pinDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pinDir, "f"), []byte("img"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pi, "settings.json"), []byte(`{"a":1}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pi, "auth.json"), []byte(`{"secret":true}`), 0o600); err != nil {
		t.Fatal(err)
	}
	sess := filepath.Join(pi, "sessions", "s")
	if err := os.MkdirAll(sess, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sess, "a.jsonl"), []byte("{}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	dest := t.TempDir()
	snap, err := e.Snapshot(true, true, dest)
	if err != nil {
		t.Fatal(err)
	}
	if snap.ID == "" || !snap.Sessions || !snap.Secrets {
		t.Fatalf("snap = %+v", snap)
	}
	if _, err := os.Stat(filepath.Join(snap.Path, "manifest.json")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(snap.Path, "picode", "picode.db")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(snap.Path, "pi", "auth.json")); err != nil {
		t.Fatal("secrets missing")
	}
	if _, err := os.Stat(filepath.Join(snap.Path, "pi", "sessions", "s", "a.jsonl")); err != nil {
		t.Fatal("sessions missing")
	}

	// Second snapshot should hardlink the unchanged session file.
	e.Now = func() time.Time { return time.Date(2026, 8, 26, 16, 0, 0, 0, time.UTC) }
	snap2, err := e.Snapshot(true, true, dest)
	if err != nil {
		t.Fatal(err)
	}
	m, err := readManifest(snap2.Path)
	if err != nil {
		t.Fatal(err)
	}
	linked := false
	for _, f := range m.Files {
		if f.Link && filepath.Base(f.Path) == "a.jsonl" {
			linked = true
		}
	}
	if !linked {
		t.Fatalf("expected hardlink on session file: %+v", m.Files)
	}

	// sessions=false omits sessions
	e.Now = func() time.Time { return time.Date(2026, 8, 26, 17, 0, 0, 0, time.UTC) }
	noSess, err := e.Snapshot(false, true, dest)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(noSess.Path, "pi", "sessions")); !os.IsNotExist(err) {
		t.Fatal("sessions should be omitted")
	}

	// secrets=false omits auth + accounts
	e.Now = func() time.Time { return time.Date(2026, 8, 26, 18, 0, 0, 0, time.UTC) }
	noSec, err := e.Snapshot(true, false, dest)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(noSec.Path, "pi", "auth.json")); !os.IsNotExist(err) {
		t.Fatal("auth should be omitted")
	}
	if _, err := os.Stat(filepath.Join(noSec.Path, "picode", "accounts.json")); !os.IsNotExist(err) {
		t.Fatal("accounts should be omitted")
	}

	// dest inside data refused
	if _, err := e.Snapshot(true, true, filepath.Join(data, "nope")); err == nil {
		t.Fatal("inside data accepted")
	}

	// restore from first snapshot after mutating live files
	if err := os.WriteFile(filepath.Join(pi, "settings.json"), []byte(`{"a":9}`), 0o644); err != nil {
		t.Fatal(err)
	}
	schema, err := e.Store.SchemaVersion()
	if err != nil {
		t.Fatal(err)
	}
	if err := e.Restore(dest, snap.ID, schema); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(pi, "settings.json"))
	if err != nil || string(got) != `{"a":1}` {
		t.Fatalf("settings after restore = %s %v", got, err)
	}
	if _, err := e.Store.GetWorkspace((func() string {
		ws, _ := e.Store.ListWorkspaces()
		if len(ws) == 0 {
			return ""
		}
		return ws[0].ID
	})()); err != nil {
		t.Fatalf("workspace missing after restore: %v", err)
	}

	// newer schema refused
	if err := e.Restore(dest, snap.ID, 0); err == nil {
		t.Fatal("newer schema should refuse")
	}

	// prune keeps newest
	if err := Prune(dest, 1, time.Date(2026, 9, 10, 0, 0, 0, 0, time.UTC)); err != nil {
		t.Fatal(err)
	}
	left, err := List(dest)
	if err != nil || len(left) != 1 {
		t.Fatalf("after prune = %d %v", len(left), err)
	}
}

func TestRestoreOmitsLeaveLiveFiles(t *testing.T) {
	e, _, pi := testEngine(t)
	if err := os.WriteFile(filepath.Join(pi, "auth.json"), []byte("live"), 0o600); err != nil {
		t.Fatal(err)
	}
	dest := t.TempDir()
	snap, err := e.Snapshot(false, false, dest)
	if err != nil {
		t.Fatal(err)
	}
	schema, _ := e.Store.SchemaVersion()
	if err := e.Restore(dest, snap.ID, schema); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(pi, "auth.json"))
	if err != nil || string(got) != "live" {
		t.Fatalf("auth wiped = %s %v", got, err)
	}
}

func TestWSLToWin(t *testing.T) {
	got, ok := WSLToWin("/mnt/e/picode-backups/x")
	if !ok || got != `E:\picode-backups\x` {
		t.Fatalf("got %q %v", got, ok)
	}
	if _, ok := WSLToWin("/home/goat"); ok {
		t.Fatal("linux path")
	}
}

func TestRemoveSnapshot(t *testing.T) {
	e, _, _ := testEngine(t)
	dest := t.TempDir()
	snap, err := e.Snapshot(false, false, dest)
	if err != nil {
		t.Fatal(err)
	}
	if err := Remove(dest, snap.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(snap.Path); !os.IsNotExist(err) {
		t.Fatal("still there")
	}
	if err := Remove(dest, snap.ID); err == nil {
		t.Fatal("missing id should fail")
	}
}

func TestVacuumInto(t *testing.T) {
	s, err := store.Open(filepath.Join(t.TempDir(), "picode.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	out := filepath.Join(t.TempDir(), "copy.db")
	if err := s.VacuumInto(out); err != nil {
		t.Fatal(err)
	}
	st, err := os.Stat(out)
	if err != nil || st.Size() == 0 {
		t.Fatalf("vacuum = %v %v", st, err)
	}
}
