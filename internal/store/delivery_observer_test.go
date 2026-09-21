package store

import (
	"encoding/json"
	"reflect"
	"sort"
	"strings"
	"testing"
)

// observerWorkspace registers a real workspace: workspace_id is a foreign key
// (migration 065), so a test cannot invent one.
func observerWorkspace(t *testing.T, s *Store) Workspace {
	t.Helper()
	w, err := s.AddWorkspace("App", t.TempDir())
	if err != nil {
		t.Fatalf("AddWorkspace: %v", err)
	}
	return w
}

func observerEventCount(t *testing.T, s *Store) int {
	t.Helper()
	evs, err := s.EventsOfType("delivery.observer.changed", 100)
	if err != nil {
		t.Fatalf("EventsOfType: %v", err)
	}
	return len(evs)
}

// TestDeliveryObserverWrites is the whole write/read cycle: a binding, a second
// one that replaces it, and the "none" sentinel that removes it. A "none" for a
// workspace that is not bound commits silently — the alternative, an event
// describing no change, would wake every subscriber for nothing (the same call
// RevokePeer makes on a row that matched nothing).
func TestDeliveryObserverWrites(t *testing.T) {
	const repoA, repoB = "/repos/a/.git", "/repos/b/.git"
	cases := []struct {
		name      string
		bound     string // repository bound before the case runs; "" for unbound
		repo      string
		observer  string
		wantBound bool
		wantEvent bool
	}{
		{"bind", "", repoA, "picode-self", true, true},
		{"rebind replaces the row", repoA, repoB, "picode-self", true, true},
		{"none removes the row", repoA, repoB, "none", false, true},
		{"none with nothing bound", "", repoB, "none", false, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s := openTest(t)
			w := observerWorkspace(t, s)
			if c.bound != "" {
				if err := s.SetDeliveryObserver(w.ID, c.bound, "picode-self"); err != nil {
					t.Fatalf("setup bind: %v", err)
				}
			}
			before, wasBound, err := s.GetDeliveryObserver(w.ID)
			if err != nil || wasBound != (c.bound != "") {
				t.Fatalf("before = %+v, %v, %v", before, wasBound, err)
			}
			events := observerEventCount(t, s)

			if err := s.SetDeliveryObserver(w.ID, c.repo, c.observer); err != nil {
				t.Fatalf("SetDeliveryObserver: %v", err)
			}

			got, isBound, err := s.GetDeliveryObserver(w.ID)
			if err != nil {
				t.Fatalf("GetDeliveryObserver: %v", err)
			}
			if isBound != c.wantBound {
				t.Fatalf("bound = %v, want %v (%+v)", isBound, c.wantBound, got)
			}
			if c.wantBound {
				if got.WorkspaceID != w.ID || got.Repo != c.repo || got.Observer != c.observer || got.UpdatedAt == "" {
					t.Fatalf("row = %+v", got)
				}
				// The second write replaces what the first one stored, at a
				// newer instant (RFC3339Nano, so an unchanged timestamp would
				// mean the row was left alone).
				if c.bound != "" && (got.Repo == before.Repo || got.UpdatedAt == before.UpdatedAt) {
					t.Fatalf("second upsert did not replace the row: %+v then %+v", before, got)
				}
			}
			if delta := observerEventCount(t, s) - events; delta != boolToInt(c.wantEvent) {
				t.Fatalf("events appended = %d, want %v", delta, c.wantEvent)
			}
		})
	}
}

// The event carries the workspace, the repository that was resolved and the
// observer, so a subscriber needs no second query to know what changed.
func TestDeliveryObserverEventPayload(t *testing.T) {
	const repo = "/repos/a/.git"
	s := openTest(t)
	w := observerWorkspace(t, s)
	if err := s.SetDeliveryObserver(w.ID, repo, "picode-self"); err != nil {
		t.Fatal(err)
	}
	if err := s.SetDeliveryObserver(w.ID, repo, "none"); err != nil {
		t.Fatal(err)
	}
	evs, err := s.EventsOfType("delivery.observer.changed", 100)
	if err != nil || len(evs) != 2 {
		t.Fatalf("events = %+v, %v", evs, err)
	}
	var payload map[string]string
	if err := json.Unmarshal(evs[0].Data, &payload); err != nil {
		t.Fatal(err)
	}
	if payload["workspaceId"] != w.ID || payload["repo"] != repo || payload["observer"] != "none" {
		t.Fatalf("payload = %+v", payload)
	}
}

// TestDeliveryObserverValidation: nothing the store cannot mean reaches a row.
func TestDeliveryObserverValidation(t *testing.T) {
	const repo = "/repos/a/.git"
	s := openTest(t)
	w := observerWorkspace(t, s)
	if err := s.SetDeliveryObserver(w.ID, repo, "picode-self"); err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name      string
		workspace string
		repo      string
		observer  string
		want      string
	}{
		{"empty workspace", " ", repo, "picode-self", "workspaceId is required"},
		{"empty repo", w.ID, "  ", "picode-self", "repository key is required"},
		{"repo over the bound", w.ID, "/" + strings.Repeat("x", 4096), "picode-self", "at most 4096 bytes"},
		{"nul in repo", w.ID, "/repos/a\x00/.git", "picode-self", "one printable line"},
		{"newline in repo", w.ID, "/repos/a\n/.git", "picode-self", "one printable line"},
		{"unknown observer", w.ID, repo, "watcher", `unknown observer "watcher"`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := s.SetDeliveryObserver(c.workspace, c.repo, c.observer)
			if err == nil || !strings.Contains(err.Error(), c.want) {
				t.Fatalf("err = %v, want %q", err, c.want)
			}
		})
	}
	// Every rejection left the stored binding and the feed alone.
	got, ok, err := s.GetDeliveryObserver(w.ID)
	if err != nil || !ok || got.Repo != repo || got.Observer != "picode-self" {
		t.Fatalf("binding = %+v, %v, %v", got, ok, err)
	}
	if n := observerEventCount(t, s); n != 1 {
		t.Fatalf("events = %d, want the one the accepted write appended", n)
	}
}

// TestDeliveryObserverListIsScopedToRepo: one repository's bindings are the
// answer to "does another workspace claim this repo" — a row bound elsewhere is
// never mixed in, and the order does not drift between reads.
func TestDeliveryObserverListIsScopedToRepo(t *testing.T) {
	const repoA, repoB = "/repos/a/.git", "/repos/b/.git"
	s := openTest(t)
	var bound []string
	for i := 0; i < 3; i++ {
		w := observerWorkspace(t, s)
		bound = append(bound, w.ID)
		if err := s.SetDeliveryObserver(w.ID, repoA, "picode-self"); err != nil {
			t.Fatal(err)
		}
	}
	other := observerWorkspace(t, s)
	if err := s.SetDeliveryObserver(other.ID, repoB, "picode-self"); err != nil {
		t.Fatal(err)
	}

	got, err := s.ListDeliveryObservers(repoA)
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(bound)
	if len(got) != len(bound) {
		t.Fatalf("rows = %+v, want %d", got, len(bound))
	}
	for i, o := range got {
		if o.WorkspaceID != bound[i] || o.Repo != repoA || o.Observer != "picode-self" || o.UpdatedAt == "" {
			t.Fatalf("row %d = %+v, want workspace %s", i, o, bound[i])
		}
	}
	again, err := s.ListDeliveryObservers(repoA)
	if err != nil || !reflect.DeepEqual(got, again) {
		t.Fatalf("order drifted: %+v then %+v (%v)", got, again, err)
	}
	onlyB, err := s.ListDeliveryObservers(repoB)
	if err != nil || len(onlyB) != 1 || onlyB[0].WorkspaceID != other.ID {
		t.Fatalf("repoB = %+v, %v", onlyB, err)
	}
	if none, err := s.ListDeliveryObservers("/repos/none/.git"); err != nil || len(none) != 0 {
		t.Fatalf("unbound repo = %+v, %v", none, err)
	}
}

// TestDeliveryObserverRollsBackWithItsEvent: the row and its announcement share
// one transaction (ADR-0048), so a failed event takes the row with it — an
// unbind whose event cannot be written leaves the binding in place.
func TestDeliveryObserverRollsBackWithItsEvent(t *testing.T) {
	const repo = "/repos/a/.git"
	cases := []struct {
		name      string
		bindFirst bool
		observer  string
		wantBound bool
	}{
		{"bind", false, "picode-self", false},
		{"unbind", true, "none", true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s := openTest(t)
			w := observerWorkspace(t, s)
			if c.bindFirst {
				if err := s.SetDeliveryObserver(w.ID, repo, "picode-self"); err != nil {
					t.Fatal(err)
				}
			}
			if _, err := s.db.Exec(`CREATE TRIGGER delivery_observer_fail_event BEFORE INSERT ON events WHEN NEW.type='delivery.observer.changed' BEGIN SELECT RAISE(ABORT,'fixture failure'); END`); err != nil {
				t.Fatal(err)
			}
			if err := s.SetDeliveryObserver(w.ID, repo, c.observer); err == nil {
				t.Fatal("expected the event failure to fail the write")
			}
			got, ok, err := s.GetDeliveryObserver(w.ID)
			if err != nil || ok != c.wantBound {
				t.Fatalf("binding = %+v, %v, %v: the row did not follow its event", got, ok, err)
			}
		})
	}
}

// A binding for a workspace that does not exist has no surface to appear on, and
// the migration's foreign key says so. Removing the workspace takes the binding
// with it.
func TestDeliveryObserverRequiresARealWorkspace(t *testing.T) {
	const repo = "/repos/a/.git"
	s := openTest(t)
	err := s.SetDeliveryObserver("ws_missing", repo, "picode-self")
	if err == nil || !strings.Contains(err.Error(), "constraint failed") {
		t.Fatalf("err = %v, want the foreign key to refuse it", err)
	}
	if _, ok, err := s.GetDeliveryObserver("ws_missing"); ok || err != nil {
		t.Fatalf("row = ok %v, err %v", ok, err)
	}

	w := observerWorkspace(t, s)
	if err := s.SetDeliveryObserver(w.ID, repo, "picode-self"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.RemoveWorkspace(w.ID); err != nil {
		t.Fatal(err)
	}
	if _, ok, err := s.GetDeliveryObserver(w.ID); ok || err != nil {
		t.Fatalf("binding survived its workspace: ok %v, err %v", ok, err)
	}
}

func boolToInt(v bool) int {
	if v {
		return 1
	}
	return 0
}
