package store

// The download list, table-tested: the start report inserts (and a re-download
// onto the same path updates in place), the outcome lands on that row, the
// search matches name/url/host, and delete/clear behave.

import (
	"database/sql"
	"errors"
	"testing"
)

func TestBrowserDownloads(t *testing.T) {
	st := historyFixture(t)

	first, err := st.AddBrowserDownload("https://files.example/a.zip", `C:\Users\me\Downloads\a.zip`, 1024)
	if err != nil {
		t.Fatal(err)
	}
	if first.Name != "a.zip" || first.Host != "files.example" || first.Status != "started" || first.Total != 1024 {
		t.Fatalf("the start report must fill name, host, status and size: %+v", first)
	}

	// The outcome lands on the same row.
	if err := st.FinishBrowserDownload(first.Path, "completed", 1024); err != nil {
		t.Fatal(err)
	}
	rows, err := st.ListBrowserDownloads(10, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].Status != "completed" || rows[0].Received != 1024 {
		t.Fatalf("the outcome must land on the row: %+v", rows)
	}

	// An outcome for a path nobody recorded is not an error, and neither is
	// an outcome word the list does not know.
	if err := st.FinishBrowserDownload(`C:\gone\b.zip`, "completed", 1); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("an unknown path must report sql.ErrNoRows, got %v", err)
	}
	if err := st.FinishBrowserDownload(first.Path, "melted", 1); err == nil {
		t.Fatal("an unknown status must be refused")
	}
	if err := st.FinishBrowserDownload("", "completed", 1); err != nil {
		t.Fatalf("an empty path is a no-op, got %v", err)
	}

	// Re-downloading onto the same path updates the row instead of adding one.
	again, err := st.AddBrowserDownload("https://files.example/a.zip?v=2", first.Path, 2048)
	if err != nil {
		t.Fatal(err)
	}
	if again.ID != first.ID {
		t.Fatalf("the same path must update its row, got %d then %d", first.ID, again.ID)
	}
	if again.Status != "started" || again.Received != 0 || again.Total != 2048 {
		t.Fatalf("a re-download must reset outcome and size: %+v", again)
	}

	if _, err := st.AddBrowserDownload("", "x", 0); err == nil {
		t.Fatal("a download without a URL must be refused")
	}

	second, err := st.AddBrowserDownload("https://cdn.example/report.pdf", `C:\Users\me\Downloads\report.pdf`, 10)
	if err != nil {
		t.Fatal(err)
	}

	// Search matches name, URL and host.
	for _, q := range []string{"report", "cdn.example", "pdf"} {
		got, err := st.ListBrowserDownloads(10, q)
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 1 || got[0].ID != second.ID {
			t.Fatalf("query %q must find only the report, got %+v", q, got)
		}
	}
	if got, _ := st.ListBrowserDownloads(10, "nothing"); len(got) != 0 {
		t.Fatalf("a miss must be empty, got %+v", got)
	}

	// Newest first.
	all, err := st.ListBrowserDownloads(10, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 2 || all[0].ID != second.ID {
		t.Fatalf("newest first, got %+v", all)
	}

	if err := st.DeleteBrowserDownload(second.ID); err != nil {
		t.Fatal(err)
	}
	if err := st.DeleteBrowserDownload(second.ID); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("deleting twice must report sql.ErrNoRows, got %v", err)
	}
	if err := st.ClearBrowserDownloads(); err != nil {
		t.Fatal(err)
	}
	if got, _ := st.ListBrowserDownloads(10, ""); len(got) != 0 {
		t.Fatalf("clear must empty the list, got %+v", got)
	}
}
