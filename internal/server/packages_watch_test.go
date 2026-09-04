package server

import (
	"context"
	"sort"
	"sync/atomic"
	"testing"
	"time"

	"github.com/cfpperche/picode/internal/feed"
	"github.com/cfpperche/picode/internal/pipkg"
	"github.com/cfpperche/picode/internal/store"
)

func TestUpdatesFingerprint(t *testing.T) {
	a := pipkg.UpdateReport{Updates: []pipkg.Available{
		{Source: "pi-roles", Current: "1.0.0", Latest: "1.1.0"},
		{Source: "pi-checklist", Current: "0.2.0", Latest: "0.3.0"},
	}}
	sameRowsOtherOrder := pipkg.UpdateReport{Updates: []pipkg.Available{
		{Source: "pi-checklist", Current: "0.2.0", Latest: "0.3.0"},
		{Source: "pi-roles", Current: "1.0.0", Latest: "1.1.0"},
	}}
	bumped := pipkg.UpdateReport{Updates: []pipkg.Available{
		{Source: "pi-roles", Current: "1.0.0", Latest: "1.2.0"},
		{Source: "pi-checklist", Current: "0.2.0", Latest: "0.3.0"},
	}}
	fewer := pipkg.UpdateReport{Updates: []pipkg.Available{
		{Source: "pi-roles", Current: "1.0.0", Latest: "1.1.0"},
	}}

	fa := updatesFingerprint(a)
	if updatesFingerprint(sameRowsOtherOrder) != fa {
		t.Fatal("row order must not change the fingerprint")
	}
	if updatesFingerprint(bumped) == fa {
		t.Fatal("a newer latest must change the fingerprint")
	}
	if updatesFingerprint(fewer) == fa {
		t.Fatal("a fixed package must change the fingerprint")
	}
	if updatesFingerprint(pipkg.UpdateReport{}) != updatesFingerprint(pipkg.UpdateReport{}) {
		t.Fatal("empty reports must be stable")
	}
}

// TestStartPackageUpdatesWatchPublishesOnChange drives the watcher with a
// stubbed scan: the first pass publishes every scope, unchanged results
// publish nothing, and a changed result publishes that scope only.
func TestStartPackageUpdatesWatchPublishesOnChange(t *testing.T) {
	st := newTestStore(t)
	ws, err := st.AddWorkspace("QA", t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	f := &feed.Feed{Store: st}
	var got []string
	f.Listen(func(ev store.Event) {
		if ev.Type == "packages.updates" {
			got = append(got, string(ev.Data))
		}
	})

	reports := []pipkg.UpdateReport{
		{Updates: []pipkg.Available{{Source: "pi-roles", Current: "1.0.0", Latest: "1.1.0"}}},
		{Updates: []pipkg.Available{{Source: "pi-roles", Current: "1.0.0", Latest: "1.1.0"}}}, // same rows
		{Updates: []pipkg.Available{{Source: "pi-roles", Current: "1.0.0", Latest: "1.2.0"}}}, // changed
	}
	var calls atomic.Int32
	old := pipkgCheck
	pipkgCheck = func(ctx context.Context, userDir, projectDir string) (pipkg.UpdateReport, error) {
		n := int(calls.Add(1) - 1)
		return reports[min(n, len(reports)-1)], nil
	}
	defer func() { pipkgCheck = old }()

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { StartPackageUpdatesWatch(ctx, Deps{Feed: f, Store: st}, 20*time.Millisecond); close(done) }()

	// Three passes over two scopes = six scans; pass two changed both
	// scopes, pass three changed nothing.
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) && calls.Load() < 6 {
		time.Sleep(10 * time.Millisecond)
	}
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("watcher did not stop")
	}

	sort.Strings(got)
	want := []string{
		`{"scope":"user","updates":[{"source":"pi-roles","scope":"","current":"1.0.0","latest":"1.1.0"}]}`,
		`{"scope":"user","updates":[{"source":"pi-roles","scope":"","current":"1.0.0","latest":"1.2.0"}]}`,
		`{"scope":"workspace:` + ws.ID + `","updates":[{"source":"pi-roles","scope":"","current":"1.0.0","latest":"1.1.0"}]}`,
		`{"scope":"workspace:` + ws.ID + `","updates":[{"source":"pi-roles","scope":"","current":"1.0.0","latest":"1.2.0"}]}`,
	}
	if len(got) != len(want) {
		t.Fatalf("got %d events (%v), want %d", len(got), got, len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("event %d:\n got %s\nwant %s", i, got[i], want[i])
		}
	}
}
