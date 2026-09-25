package server

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/cfpperche/picode/internal/clilaunch"
	"github.com/cfpperche/picode/internal/clisession"
	"github.com/cfpperche/picode/internal/store"
	_ "modernc.org/sqlite"
)

// A start run on a guest CLI ends with the CLI's own last answer (ADR-0217),
// read where the CLI keeps it even when the hook that pinned the session
// named no file (Hermes); a message from before the prompt is not the run's.
func TestCLIRunFinalReadsTheSessionsLastAnswer(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "picode.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	deps := Deps{Store: st, DataDir: t.TempDir()}
	oldTries, oldEvery := cliFinalTries, cliFinalEvery
	cliFinalTries, cliFinalEvery = 1, time.Millisecond
	t.Cleanup(func() { cliFinalTries, cliFinalEvery = oldTries, oldEvery })

	db := filepath.Join(t.TempDir(), "state.db")
	old := clisession.HermesTestDB
	clisession.HermesTestDB = db
	t.Cleanup(func() { clisession.HermesTestDB = old })
	sent := time.Now()
	at := func(d time.Duration) string { return fmt.Sprintf("%f", float64(sent.Add(d).UnixNano())/1e9) }
	h, err := sql.Open("sqlite", "file:"+db)
	if err != nil {
		t.Fatal(err)
	}
	for _, q := range []string{
		`CREATE TABLE sessions (id TEXT PRIMARY KEY, source TEXT, title TEXT, model TEXT, cwd TEXT, started_at REAL)`,
		`CREATE TABLE messages (id INTEGER PRIMARY KEY, session_id TEXT, role TEXT, content TEXT, timestamp REAL)`,
		`INSERT INTO sessions VALUES ('h1','cli','T','nous/hermes','/repo',` + at(0) + `)`,
		`INSERT INTO messages VALUES (1,'h1','user','run the checks',` + at(0) + `)`,
		`INSERT INTO messages VALUES (2,'h1','assistant','Working on it.',` + at(time.Second) + `)`,
		`INSERT INTO messages VALUES (3,'h1','assistant','All 12 checks pass.',` + at(2*time.Second) + `)`,
		`INSERT INTO sessions VALUES ('h0','cli','T','nous/hermes','/repo',` + at(-time.Hour) + `)`,
		`INSERT INTO messages VALUES (4,'h0','assistant','An answer from an hour ago.',` + at(-time.Hour) + `)`,
	} {
		if _, err := h.Exec(q); err != nil {
			t.Fatalf("%s: %v", q, err)
		}
	}
	h.Close()

	term, err := st.CreateTerminalIn("", "Hermes run", t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	pin := func(id string) {
		if err := st.SetTerminalLastSession(term.ID, store.TerminalLastSession{CLI: "hermes", SessionID: id}); err != nil {
			t.Fatal(err)
		}
	}
	if err := st.SetTerminalLaunch(term.ID, "hermes", clilaunch.Overrides{}); err != nil {
		t.Fatal(err)
	}
	pin("h1")
	if got := cliRunFinal(context.Background(), deps, term, "hermes", sent); got != "All 12 checks pass." {
		t.Fatalf("final = %q, want the session's last answer", got)
	}
	pin("h0")
	if got := cliRunFinal(context.Background(), deps, term, "hermes", sent); got != "" {
		t.Fatalf("final = %q: an answer from before the prompt is not the run's", got)
	}
	if got := cliRunFinal(context.Background(), deps, term, "agy-unknown", sent); got != "" {
		t.Fatalf("a CLI with no reader gave %q", got)
	}
}
