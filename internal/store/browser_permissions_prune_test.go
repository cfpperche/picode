package store

// The prune: "forget the sites nothing has visited inside the window". One row
// per condition that changes the outcome — the table is also written in
// PruneBrowserPermissions' own comment.

import (
	"testing"
	"time"
)

func TestPruneBrowserPermissions(t *testing.T) {
	st := testStore(t)
	old := time.Now().AddDate(0, 0, -200).UTC().Format(time.RFC3339Nano)
	recent := time.Now().AddDate(0, 0, -2).UTC().Format(time.RFC3339Nano)
	backdate := func(url, host, at string) {
		t.Helper()
		if _, err := st.db.Exec(
			`INSERT INTO browser_history (url, title, host, typed, visited_at) VALUES (?, ?, ?, 0, ?)`,
			url, "t", host, at,
		); err != nil {
			t.Fatal(err)
		}
	}
	// A site visited two days ago, one visited only in the spring, and one the
	// history has never heard of.
	backdate("https://recent.test/a", "recent.test", recent)
	backdate("https://stale.test/a", "stale.test", old)
	for _, row := range []struct {
		origin, kind string
	}{
		{"recent.test", "camera"},
		{"stale.test", "camera"},
		{"never.test", "camera"},
		{"*", "camera"},
	} {
		if _, err := st.SetBrowserPermission(row.origin, row.kind, "allow", true); err != nil {
			t.Fatal(err)
		}
	}

	pruned, err := st.PruneBrowserPermissions(time.Now().AddDate(0, 0, -90))
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]bool{}
	for _, p := range pruned {
		got[p.Origin] = true
	}
	if len(pruned) != 2 || !got["stale.test"] || !got["never.test"] {
		t.Fatalf("pruned %+v, want stale.test and never.test", got)
	}
	left, err := st.ListBrowserPermissions(50, "", "")
	if err != nil {
		t.Fatal(err)
	}
	kept := map[string]bool{}
	for _, p := range left {
		kept[p.Origin] = true
	}
	if !kept["recent.test"] || !kept["*"] || kept["stale.test"] || kept["never.test"] {
		t.Fatalf("kept %+v, want recent.test and the every-site policy", kept)
	}

	// Nothing left to forget: no rows, no error, and no second event.
	again, err := st.PruneBrowserPermissions(time.Now().AddDate(0, 0, -90))
	if err != nil || len(again) != 0 {
		t.Fatalf("second prune: %+v %v", again, err)
	}
}
