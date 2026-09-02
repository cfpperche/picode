package feed

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/cfpperche/picode/internal/store"
)

func openStore(t *testing.T) *store.Store {
	t.Helper()
	st, err := store.Open(filepath.Join(t.TempDir(), "picode.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	return st
}

func recv(t *testing.T, ch <-chan store.Event) store.Event {
	t.Helper()
	select {
	case ev := <-ch:
		return ev
	case <-time.After(2 * time.Second):
		t.Fatal("no event")
		return store.Event{}
	}
}

func TestLiveThenReplayThenReset(t *testing.T) {
	st := openStore(t)
	f := &Feed{Store: st}
	st.OnEvent = f.Publish

	_, live, unsub, err := f.Subscribe(0)
	if err != nil {
		t.Fatal(err)
	}
	_ = st.SetSetting("a", "1")
	ev := recv(t, live)
	if ev.Type != "setting.updated" || ev.ID == 0 {
		t.Fatalf("live = %+v", ev)
	}
	unsub()
	if f.Subscribers() != 0 {
		t.Fatal("unsub left a subscriber")
	}

	// Missed two while away: replay from the cursor, then live continues.
	_ = st.SetSetting("b", "2")
	_ = st.SetSetting("c", "3")
	replay, live, unsub, err := f.Subscribe(ev.ID)
	if err != nil || len(replay) != 2 || replay[0].ID != ev.ID+1 || replay[1].ID != ev.ID+2 {
		t.Fatalf("replay = %+v %v", replay, err)
	}
	_ = st.SetSetting("d", "4")
	if got := recv(t, live); got.ID != ev.ID+3 {
		t.Fatalf("live after replay = %+v", got)
	}
	unsub()

	// Cursor at latest: nothing to replay, still subscribed.
	replay, _, unsub, err = f.Subscribe(f.Latest())
	if err != nil || len(replay) != 0 {
		t.Fatalf("at latest: %v %v", replay, err)
	}
	unsub()

	// Retention pruned everything the cursor points at → reset.
	if _, err := st.PruneEvents(time.Now().Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	_ = st.SetSetting("e", "5")
	if _, _, _, err := f.Subscribe(1); err != ErrReset {
		t.Fatalf("stale cursor = %v, want ErrReset", err)
	}
}

func TestEphemeralAndListeners(t *testing.T) {
	f := &Feed{}
	var seen []string
	f.Listen(func(ev store.Event) { seen = append(seen, ev.Type) })
	_, live, unsub, _ := f.Subscribe(0)
	defer unsub()
	f.Ephemeral("device.online", map[string]string{"id": "d1"})
	ev := recv(t, live)
	if ev.ID != 0 || ev.Type != "device.online" || string(ev.Data) != `{"id":"d1"}` {
		t.Fatalf("ephemeral = %+v", ev)
	}
	if len(seen) != 1 || seen[0] != "device.online" {
		t.Fatalf("listener = %v", seen)
	}
}

func TestSlowSubscriberDropsNotBlocks(t *testing.T) {
	f := &Feed{}
	_, _, unsub, _ := f.Subscribe(0)
	defer unsub()
	done := make(chan struct{})
	go func() {
		for i := 0; i < 1000; i++ {
			f.Ephemeral("x", nil)
		}
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Publish blocked on a slow subscriber")
	}
}
