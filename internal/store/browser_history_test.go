package store

// The browser history store, table-tested: the first visit inserts, a
// re-reported URL updates the newest row in place instead of piling up,
// list orders newest-first and filters, delete misses answer sql.ErrNoRows,
// and clear empties everything.

import (
	"database/sql"
	"errors"
	"path/filepath"
	"strings"
	"testing"
)

func historyFixture(t *testing.T) *Store {
	t.Helper()
	st, err := Open(filepath.Join(t.TempDir(), "picode.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	return st
}

func TestBrowserVisitInsertsAndUpdatesInPlace(t *testing.T) {
	st := historyFixture(t)
	if _, err := st.AddBrowserVisit("", "x", false); err == nil {
		t.Fatal("an empty URL must be refused")
	}
	first, err := st.AddBrowserVisit("https://Example.com/a?b=1", "Example", false)
	if err != nil {
		t.Fatal(err)
	}
	if first.Host != "example.com" || first.Typed {
		t.Fatalf("host/typed wrong: %+v", first)
	}
	again, err := st.AddBrowserVisit("https://example.com/a?b=1", "Example — docs", true)
	if err != nil {
		t.Fatal(err)
	}
	if again.ID != first.ID {
		t.Fatalf("the same URL must update its row in place: %d then %d", first.ID, again.ID)
	}
	if !again.Typed {
		t.Fatal("a typed re-report must set the typed flag")
	}
	other, err := st.AddBrowserVisit("https://example.org/", "", true)
	if err != nil {
		t.Fatal(err)
	}
	if other.ID == first.ID {
		t.Fatal("a different URL must be a new row")
	}
}

func TestBrowserHistoryListFiltersNewestFirst(t *testing.T) {
	st := historyFixture(t)
	if _, err := st.AddBrowserVisit("https://a.test/one", "One", false); err != nil {
		t.Fatal(err)
	}
	if _, err := st.AddBrowserVisit("https://b.test/two", "Two docs", true); err != nil {
		t.Fatal(err)
	}
	visits, err := st.ListBrowserHistory(0, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(visits) != 2 || visits[0].URL != "https://b.test/two" {
		t.Fatalf("want newest first, got %+v", visits)
	}
	filtered, err := st.ListBrowserHistory(0, "docs")
	if err != nil {
		t.Fatal(err)
	}
	if len(filtered) != 1 || !strings.Contains(filtered[0].Title, "docs") {
		t.Fatalf("filter by title failed: %+v", filtered)
	}
}

func TestBrowserHistoryDeleteAndClear(t *testing.T) {
	st := historyFixture(t)
	v, err := st.AddBrowserVisit("https://a.test/one", "", false)
	if err != nil {
		t.Fatal(err)
	}
	if err := st.DeleteBrowserVisit(v.ID); err != nil {
		t.Fatal(err)
	}
	if err := st.DeleteBrowserVisit(v.ID); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("a second delete must be a miss, got %v", err)
	}
	if _, err := st.AddBrowserVisit("https://b.test/two", "", false); err != nil {
		t.Fatal(err)
	}
	if err := st.ClearBrowserHistory(); err != nil {
		t.Fatal(err)
	}
	visits, err := st.ListBrowserHistory(0, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(visits) != 0 {
		t.Fatalf("clear must empty the table, got %+v", visits)
	}
}
