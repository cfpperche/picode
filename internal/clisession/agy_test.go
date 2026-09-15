package clisession

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"
)

// seedAgyDB writes a conversation_summaries.db with the shape Antigravity
// 1.2.2 ships (columns probed from a real
// ~/.gemini/antigravity-cli/conversation_summaries.db).
func seedAgyDB(t *testing.T, dir string) string {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "conversation_summaries.db")
	db, err := sql.Open("sqlite", "file:"+path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(`CREATE TABLE conversation_summaries (
		conversation_id TEXT PRIMARY KEY,
		title TEXT NOT NULL DEFAULT '',
		preview TEXT NOT NULL DEFAULT '',
		step_count INTEGER NOT NULL DEFAULT 0,
		last_modified_time datetime NOT NULL DEFAULT '',
		workspace_uris TEXT NOT NULL DEFAULT '',
		parent_conversation_id TEXT NOT NULL DEFAULT '',
		agent_name TEXT NOT NULL DEFAULT ''
	)`); err != nil {
		t.Fatal(err)
	}
	conv := filepath.Join(dir, "conversations")
	if err := os.MkdirAll(conv, 0o755); err != nil {
		t.Fatal(err)
	}
	ins := `INSERT INTO conversation_summaries (conversation_id, title, preview, step_count, last_modified_time, workspace_uris, parent_conversation_id, agent_name) VALUES (?,?,?,?,?,?,?,?)`
	insert := func(id, title, preview string, steps int, time, ws, parent string) (string, error) {
		_, err := db.Exec(ins, id, title, preview, steps, time, ws, parent, "")
		return id, err
	}
	// The conversations the picker shows: one in this folder, one elsewhere.
	if _, err := insert("c4fdab02-f9e5-486f-be70-63db2b6a9a82", "Friendly Conversation Starter", "Friendly Conversation Starter", 2, "2026-09-15 13:49:14.466367991+00:00", `["file:///home/goat/picode"]`, ""); err != nil {
		t.Fatal(err)
	}
	if _, err := insert("1bda9883-a93d-4caa-a91f-88b52c717f2f", "", "Greeting And Language Inquiry", 6, "2026-06-23 17:33:38.877243985+00:00", `["file:///home/goat/tachyon"]`, ""); err != nil {
		t.Fatal(err)
	}
	// Kept out: no steps, no workspace, the zero time, a subagent run.
	for _, row := range []struct {
		id, title, preview string
		steps              int
		time, ws, parent   string
	}{
		{"7bd5d316-0000-0000-0000-000000000001", "", "", 0, "0001-01-01 00:00:00+00:00", `["file:///home/goat"]`, ""},
		{"817a857d-0000-0000-0000-000000000002", "", "Identity Verification Confirmation", 4, "2026-07-02 20:39:58.218084313+00:00", "", ""},
		{"9956e08f-0000-0000-0000-000000000003", "", "Confirming System Runtime Status", 4, "2026-07-02 18:12:23.661420388+00:00", "", ""},
		{"b60fcc98-0000-0000-0000-000000000004", "Sub run", "Sub run", 3, "2026-09-15 12:00:00+00:00", `["file:///home/goat/picode"]`, "c4fdab02-f9e5-486f-be70-63db2b6a9a82"},
	} {
		if _, err := insert(row.id, row.title, row.preview, row.steps, row.time, row.ws, row.parent); err != nil {
			t.Fatal(err)
		}
	}
	// The per-conversation store backs the size column.
	store := filepath.Join(conv, "c4fdab02-f9e5-486f-be70-63db2b6a9a82.db")
	if err := os.WriteFile(store, []byte("sqlite placeholder"), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestAgySourceList(t *testing.T) {
	dir := t.TempDir()
	AgyTestDB = seedAgyDB(t, dir)
	t.Cleanup(func() { AgyTestDB = "" })

	got, err := AgySource{}.List("")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("conversations = %d, want 2: %+v", len(got), got)
	}
	// Newest first, and the title/workspace/time come off the index.
	s := got[0]
	if s.CLI != "agy" || s.ID != "c4fdab02-f9e5-486f-be70-63db2b6a9a82" {
		t.Fatalf("identity: %+v", s)
	}
	if s.Name != "Friendly Conversation Starter" || s.Preview != "Friendly Conversation Starter" {
		t.Fatalf("text: %+v", s)
	}
	if s.Cwd != "/home/goat/picode" || s.Messages != 2 {
		t.Fatalf("folder/steps: %+v", s)
	}
	if s.CreatedAt != "2026-09-15T13:49:14Z" || s.UpdatedAt != s.CreatedAt {
		t.Fatalf("time: %q", s.CreatedAt)
	}
	if len(s.ResumeArgs) != 2 || s.ResumeArgs[0] != "--conversation" || s.ResumeArgs[1] != s.ID {
		t.Fatalf("resume args: %v", s.ResumeArgs)
	}
	if s.Size != int64(len("sqlite placeholder")) {
		t.Fatalf("size from the conversation store: %d", s.Size)
	}
	// A row with no title falls back to the preview, never to an empty label.
	if got[1].Name != "Greeting And Language Inquiry" || got[1].Cwd != "/home/goat/tachyon" {
		t.Fatalf("title fallback: %+v", got[1])
	}

	if scoped, _ := (AgySource{}).List("/home/goat/picode"); len(scoped) != 1 || scoped[0].ID != s.ID {
		t.Fatalf("scoped = %+v", scoped)
	}
	if empty, _ := (AgySource{}).List("/elsewhere"); len(empty) != 0 {
		t.Fatalf("elsewhere = %+v", empty)
	}
}

func TestAgySourceMissingAndLocked(t *testing.T) {
	AgyTestDB = filepath.Join(t.TempDir(), "absent.db")
	t.Cleanup(func() { AgyTestDB = "" })
	if got, err := (AgySource{}).List(""); err != nil || got != nil {
		t.Fatalf("a machine that never ran antigravity lists empty: %v %v", got, err)
	}
	path := filepath.Join(t.TempDir(), "broken.db")
	if err := os.WriteFile(path, []byte("not a database"), 0o600); err != nil {
		t.Fatal(err)
	}
	AgyTestDB = path
	if got, err := (AgySource{}).List(""); err != nil || got != nil {
		t.Fatalf("broken index = %v %v", got, err)
	}
}

func TestAgyWorkspaceAndTime(t *testing.T) {
	for _, tc := range []struct{ in, want string }{
		{`["file:///home/goat/picode"]`, "/home/goat/picode"},
		{`["file:///home/goat/tachyon","file:///other"]`, "/home/goat/tachyon"},
		{`["file:///home/goat/my%20project"]`, "/home/goat/my project"},
		{`/home/goat/plain`, "/home/goat/plain"},
		{`[]`, ""},
		{``, ""},
	} {
		if got := agyWorkspace(tc.in); got != tc.want {
			t.Errorf("agyWorkspace(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
	for _, tc := range []struct{ in, want string }{
		{"2026-09-15 13:49:14.466367991+00:00", "2026-09-15T13:49:14Z"},
		{"2026-06-23 17:33:38.877243985+00:00", "2026-06-23T17:33:38Z"},
		{"2026-09-15 10:49:14-03:00", "2026-09-15T13:49:14Z"},
		// The driver's spelling of the same column (declared `datetime`).
		{"2026-09-15T13:49:14.466367991Z", "2026-09-15T13:49:14Z"},
		{"2026-09-15T10:49:14-03:00", "2026-09-15T13:49:14Z"},
		{"0001-01-01 00:00:00+00:00", ""},
		{"0001-01-01T00:00:00Z", ""},
		{"", ""},
		{"not a time", ""},
	} {
		if got := agyTime(tc.in); got != tc.want {
			t.Errorf("agyTime(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
