package push

import (
	"context"
	"encoding/json"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/cfpperche/picode/internal/store"
)

type fakePresence struct{ online bool }

func (f fakePresence) AnyHostOnline() bool { return f.online }

type capture struct {
	mu   sync.Mutex
	sent []Message
	tags []string
	err  error
}

func (c *capture) send(_ context.Context, _ Target, payload []byte, _ int, _ string, topic string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	var m Message
	_ = json.Unmarshal(payload, &m)
	c.sent = append(c.sent, m)
	c.tags = append(c.tags, topic)
	return c.err
}

func (c *capture) wait(n int) []Message {
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		c.mu.Lock()
		if len(c.sent) >= n {
			out := append([]Message(nil), c.sent...)
			c.mu.Unlock()
			return out
		}
		c.mu.Unlock()
		time.Sleep(10 * time.Millisecond)
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]Message(nil), c.sent...)
}

func newNotifier(t *testing.T, online bool, prefs store.PushPrefs) (*Notifier, *capture, *store.Store) {
	t.Helper()
	st, err := store.Open(filepath.Join(t.TempDir(), "n.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })
	if _, err := st.UpsertPushSubscription("https://push.example/dev", "k", "a", "d1", "phone", prefs); err != nil {
		t.Fatal(err)
	}
	c := &capture{}
	n := &Notifier{Store: st, Presence: fakePresence{online}, send: c.send}
	return n, c, st
}

// Decision table rows (ADR-0047).
func TestNotifierDecisionTable(t *testing.T) {
	both := store.PushPrefs{Actions: true, Finished: true}
	cases := []struct {
		name   string
		online bool
		prefs  store.PushPrefs
		item   store.InboxItem
		want   int
		tag    string
	}{
		{"host online skips", true, both, store.InboxItem{ID: "i1", Kind: store.InboxApproval, Blocking: true, Title: "x"}, 0, ""},
		{"blocking approval → actions", false, both, store.InboxItem{ID: "i2", Kind: store.InboxApproval, Blocking: true, Title: "Deploy?", Reason: "prod"}, 1, "inbox:i2"},
		{"blocking but actions off", false, store.PushPrefs{Actions: false, Finished: true}, store.InboxItem{ID: "i3", Kind: store.InboxQuestion, Blocking: true, Title: "x"}, 0, ""},
		{"result → finished", false, both, store.InboxItem{ID: "i4", Kind: store.InboxResult, SourceID: "ag1", Title: "builder finished", Body: "done"}, 1, "result:ag1"},
		{"result but finished off", false, store.PushPrefs{Actions: true, Finished: false}, store.InboxItem{ID: "i5", Kind: store.InboxResult, SourceID: "ag1", Title: "x"}, 0, ""},
		{"non-blocking fyi never", false, both, store.InboxItem{ID: "i6", Kind: store.InboxFYI, Blocking: false, Title: "x"}, 0, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			n, c, _ := newNotifier(t, tc.online, tc.prefs)
			n.OnInbox(tc.item)
			got := c.wait(tc.want)
			if len(got) != tc.want {
				t.Fatalf("sent %d, want %d: %+v", len(got), tc.want, got)
			}
			if tc.want == 1 {
				if c.tags[0] != tc.tag || got[0].Hash != "#/inbox/"+tc.item.ID {
					t.Fatalf("tag=%q hash=%q", c.tags[0], got[0].Hash)
				}
			}
		})
	}
}

func TestNotifierWaitingAndGone(t *testing.T) {
	n, c, st := newNotifier(t, false, store.PushPrefs{Actions: true, Finished: true})
	n.OnWaiting("ag1", "builder", "Run the migration?", "drops a table")
	got := c.wait(1)
	if len(got) != 1 || got[0].Title != "builder needs you" || got[0].Hash != "#/agent/ag1" || c.tags[0] != "ask:ag1" {
		t.Fatalf("waiting push = %+v tags=%v", got, c.tags)
	}
	// A 410 from the service drops the subscription.
	c.err = ErrGone
	n.OnWaiting("ag1", "builder", "again", "")
	c.wait(2)
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		list, _ := st.ListPushSubscriptions()
		if len(list) == 0 {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("gone subscription was not dropped")
}

func TestNotifierSendTestIgnoresPresence(t *testing.T) {
	n, c, _ := newNotifier(t, true, store.PushPrefs{})
	if err := n.SendTest(context.Background(), "https://push.example/dev"); err != nil {
		t.Fatal(err)
	}
	if len(c.wait(1)) != 1 || c.tags[0] != "test" {
		t.Fatalf("test push = %+v", c.sent)
	}
	if err := n.SendTest(context.Background(), "https://push.example/nope"); err == nil {
		t.Fatal("unknown endpoint must error")
	}
}
