package store

// The Servers panel's hides: one row per (port, pid, start token), so "hidden"
// always means one listener — a new process on the same port comes back
// visible, and a hide whose process is gone is forgotten.

import (
	"database/sql"
	"errors"
	"testing"
)

func TestDevServerHideRoundTrip(t *testing.T) {
	s := openTest(t)
	var events []string
	s.OnEvent = func(ev Event) { events = append(events, ev.Type) }

	first, err := s.HideDevServer(5173, 4242, "900", `terminal "web" · PiCode`, "vite")
	if err != nil {
		t.Fatal(err)
	}
	if first.ID == 0 || first.Port != 5173 || first.PID != 4242 || first.StartKey != "900" || first.Owner == "" {
		t.Fatalf("hide = %+v", first)
	}
	// Hiding the same listener again updates the label in place.
	again, err := s.HideDevServer(5173, 4242, "900", `terminal "web"`, "vite-dev")
	if err != nil {
		t.Fatal(err)
	}
	if again.ID != first.ID || again.Tool != "vite-dev" || again.Owner != `terminal "web"` {
		t.Fatalf("second hide = %+v (first %+v)", again, first)
	}
	// The same port from a different process is a different listener.
	other, err := s.HideDevServer(5173, 4243, "901", "", "astro")
	if err != nil {
		t.Fatal(err)
	}
	if other.ID == first.ID {
		t.Fatalf("a new process reused the hide: %+v", other)
	}

	hides, err := s.ListDevServerHides()
	if err != nil {
		t.Fatal(err)
	}
	if len(hides) != 2 {
		t.Fatalf("hides = %+v", hides)
	}

	if err := s.UnhideDevServer(first.ID); err != nil {
		t.Fatal(err)
	}
	if err := s.UnhideDevServer(first.ID); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("second unhide = %v, want sql.ErrNoRows", err)
	}

	// Pruning keeps what the caller says is alive and forgets the rest.
	stale, err := s.HideDevServer(45683, 4245, "903", "", "agy")
	if err != nil {
		t.Fatal(err)
	}
	n, err := s.PruneDevServerHides([]int64{other.ID})
	if err != nil || n != 1 {
		t.Fatalf("prune = (%d, %v) want (1, nil)", n, err)
	}
	if n, err := s.PruneDevServerHides([]int64{other.ID}); err != nil || n != 0 {
		t.Fatalf("second prune = (%d, %v) want (0, nil)", n, err)
	}
	if hides, _ := s.ListDevServerHides(); len(hides) != 1 || hides[0].ID != other.ID {
		t.Fatalf("after prune = %+v", hides)
	}
	if _, err := s.devServerHideFor(stale.Port, stale.PID, stale.StartKey); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("the stale hide survived: %v", err)
	}

	// Every mutation announced itself, and only the ones that changed
	// something did: hide, hide (in place), hide, unhide, hide, prune. The
	// second unhide and the empty prune are not in the list.
	want := []string{
		"devserver.hidden", "devserver.hidden", "devserver.hidden",
		"devserver.hidden", "devserver.hidden", "devserver.hidden",
	}
	if len(events) != len(want) {
		t.Fatalf("events = %v want %v", events, want)
	}
	for i := range want {
		if events[i] != want[i] {
			t.Fatalf("events = %v want %v", events, want)
		}
	}
}

func TestDevServerHideValidation(t *testing.T) {
	s := openTest(t)
	cases := []struct {
		name     string
		port     int
		pid      int
		startKey string
	}{
		{"no port", 0, 4242, "900"},
		{"port above the range", 70000, 4242, "900"},
		{"no process", 5173, 0, "900"},
		{"no start token", 5173, 4242, ""},
		{"blank start token", 5173, 4242, "   "},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if _, err := s.HideDevServer(c.port, c.pid, c.startKey, "", ""); err == nil {
				t.Fatal("a hide without an identity was accepted")
			}
		})
	}
	if hides, _ := s.ListDevServerHides(); len(hides) != 0 {
		t.Fatalf("a refused hide wrote a row: %+v", hides)
	}
}
