package store

// The site-permission rows, table-tested: a decision inserts, the same pair
// changes in place, a standing is never demoted by a later one-off decision,
// the kind and origin vocabularies are closed, and delete/clear behave.

import (
	"database/sql"
	"errors"
	"testing"
)

func TestBrowserPermissions(t *testing.T) {
	st := historyFixture(t)

	first, err := st.SetBrowserPermission("Meet.Example.com", "camera", "allow", false)
	if err != nil {
		t.Fatal(err)
	}
	if first.Origin != "meet.example.com" || first.Kind != "camera" || first.Decision != "allow" {
		t.Fatalf("the row must be normalized: %+v", first)
	}
	if first.Standing {
		t.Fatalf("a reported decision is not a standing: %+v", first)
	}

	// The same site and kind changes in place.
	same, err := st.SetBrowserPermission("meet.example.com", "camera", "deny", false)
	if err != nil {
		t.Fatal(err)
	}
	if same.ID != first.ID || same.Decision != "deny" {
		t.Fatalf("the pair must update its row: %+v then %+v", first, same)
	}
	rows, err := st.ListBrowserPermissions(10, "", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 {
		t.Fatalf("one pair, one row, got %+v", rows)
	}

	// A standing is policy: a later one-off decision on the same pair updates
	// the word but must not demote it (the shell reports every decision,
	// including the ones the standing itself produced).
	standing, err := st.SetBrowserPermission("meet.example.com", "camera", "allow", true)
	if err != nil {
		t.Fatal(err)
	}
	if !standing.Standing {
		t.Fatalf("the saved answer must be a standing: %+v", standing)
	}
	after, err := st.SetBrowserPermission("meet.example.com", "camera", "deny", false)
	if err != nil {
		t.Fatal(err)
	}
	if after.Decision != "deny" || !after.Standing {
		t.Fatalf("a one-off decision cannot demote a standing: %+v", after)
	}

	// Two kinds for the same site are two rows.
	if _, err := st.SetBrowserPermission("meet.example.com", "microphone", "allow", false); err != nil {
		t.Fatal(err)
	}
	onlyCamera, err := st.ListBrowserPermissions(10, "camera", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(onlyCamera) != 1 || onlyCamera[0].Kind != "camera" {
		t.Fatalf("a kind filter must narrow to that kind, got %+v", onlyCamera)
	}
	if got, _ := st.ListBrowserPermissions(10, "", "meet"); len(got) != 2 {
		t.Fatalf("the origin search must find both, got %+v", got)
	}
	if got, _ := st.ListBrowserPermissions(10, "", "other"); len(got) != 0 {
		t.Fatalf("a miss must be empty, got %+v", got)
	}

	// The vocabularies are closed; "ask" is a policy word, so the every-site
	// entry accepts it.
	if _, err := st.SetBrowserPermission("", "camera", "allow", false); err == nil {
		t.Fatal("an empty origin must be refused")
	}
	if _, err := st.SetBrowserPermission("a.example", "telepathy", "allow", false); err == nil {
		t.Fatal("an unknown kind must be refused")
	}
	if _, err := st.SetBrowserPermission("a.example", "camera", "maybe", false); err == nil {
		t.Fatal("an unknown decision must be refused")
	}
	ask, err := st.SetBrowserPermission("*", "camera", "ask", true)
	if err != nil {
		t.Fatal(err)
	}
	if ask.Decision != "ask" || !ask.Standing {
		t.Fatalf("an ask policy must be stored as a standing: %+v", ask)
	}

	// Per-kind reset leaves the other kinds alone.
	if err := st.ClearBrowserPermissions("camera"); err != nil {
		t.Fatal(err)
	}
	rows, err = st.ListBrowserPermissions(10, "", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].Kind != "microphone" {
		t.Fatalf("only the camera rows must go, got %+v", rows)
	}

	if err := st.DeleteBrowserPermission(rows[0].ID); err != nil {
		t.Fatal(err)
	}
	if err := st.DeleteBrowserPermission(rows[0].ID); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("deleting twice must report sql.ErrNoRows, got %v", err)
	}
	if err := st.ClearBrowserPermissions(""); err != nil {
		t.Fatal(err)
	}
	if got, _ := st.ListBrowserPermissions(10, "", ""); len(got) != 0 {
		t.Fatalf("a full clear must empty the table, got %+v", got)
	}
}
