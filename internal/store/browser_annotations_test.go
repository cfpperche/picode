package store

// The annotation rows, table-tested: what makes one valid, the host
// normalization, the search, the clamp and the delete.

import "testing"

func TestBrowserAnnotations(t *testing.T) {
	st := testStore(t)

	if _, err := st.CreateBrowserAnnotation(BrowserAnnotation{Selector: ".x"}); err == nil {
		t.Fatal("an annotation without a page must be refused")
	}
	if _, err := st.CreateBrowserAnnotation(BrowserAnnotation{URL: "https://x.test/a"}); err == nil {
		t.Fatal("an annotation with nothing to look at must be refused")
	}

	a, err := st.CreateBrowserAnnotation(BrowserAnnotation{
		URL: "https://X.test/a", Selector: ".hero", Comment: "the spacing here", DOM: "<div/>",
		Shot: "s.png", Note: "n.md", TerminalID: "t1",
	})
	if err != nil {
		t.Fatal(err)
	}
	if a.Host != "x.test" {
		t.Fatalf("host must be normalized: %+v", a)
	}
	// A screenshot alone is something to look at, and the comment is optional.
	b, err := st.CreateBrowserAnnotation(BrowserAnnotation{URL: "https://y.test/b", Shot: "s2.png"})
	if err != nil {
		t.Fatalf("a screenshot alone is valid: %v", err)
	}

	list, err := st.ListBrowserAnnotations(10, "")
	if err != nil || len(list) != 2 || list[0].ID != b.ID {
		t.Fatalf("newest first: %+v %v", list, err)
	}
	if found, _ := st.ListBrowserAnnotations(10, "spacing"); len(found) != 1 || found[0].ID != a.ID {
		t.Fatalf("search by comment: %+v", found)
	}
	if found, _ := st.ListBrowserAnnotations(10, "y.test"); len(found) != 1 || found[0].ID != b.ID {
		t.Fatalf("search by host: %+v", found)
	}
	if found, _ := st.ListBrowserAnnotations(999, ""); len(found) != 2 {
		t.Fatalf("the limit clamps to 200: %+v", found)
	}

	if err := st.DeleteBrowserAnnotation(a.ID); err != nil {
		t.Fatal(err)
	}
	if err := st.DeleteBrowserAnnotation(a.ID); err == nil {
		t.Fatal("deleting twice must report no row")
	}
	if left, _ := st.ListBrowserAnnotations(10, ""); len(left) != 1 {
		t.Fatalf("one row left: %+v", left)
	}
}
