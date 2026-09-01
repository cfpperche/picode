package store

import (
	"path/filepath"
	"testing"
)

func TestPushSubscriptionsCRUD(t *testing.T) {
	st, err := Open(filepath.Join(t.TempDir(), "p.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	if _, err := st.UpsertPushSubscription("http://insecure", "k", "a", "", "", PushPrefs{}); err == nil {
		t.Fatal("http endpoint must be refused")
	}
	sub, err := st.UpsertPushSubscription("https://push.example/1", "p256", "auth", "dev1", "iPhone", PushPrefs{Actions: true, Finished: false})
	if err != nil {
		t.Fatal(err)
	}
	if sub.Prefs.Actions != true || sub.Prefs.Finished != false || sub.P256dh != "p256" {
		t.Fatalf("sub = %+v", sub)
	}
	// Re-subscribe refreshes keys and resets failures.
	_ = st.MarkPushFailure(sub.Endpoint)
	again, err := st.UpsertPushSubscription("https://push.example/1", "p256-new", "auth2", "dev1", "iPhone", PushPrefs{Actions: true, Finished: true})
	if err != nil || again.P256dh != "p256-new" || again.Failures != 0 || again.ID != sub.ID {
		t.Fatalf("upsert = %+v, %v", again, err)
	}
	list, _ := st.ListPushSubscriptions()
	if len(list) != 1 {
		t.Fatalf("list = %d", len(list))
	}
	p, err := st.SetPushPrefs(sub.Endpoint, PushPrefs{Actions: false, Finished: true})
	if err != nil || p.Prefs.Actions || !p.Prefs.Finished {
		t.Fatalf("prefs = %+v, %v", p, err)
	}
	if _, err := st.SetPushPrefs("https://push.example/none", PushPrefs{}); err != ErrNotFound {
		t.Fatalf("missing prefs = %v", err)
	}
	if err := st.MarkPushOK(sub.Endpoint); err != nil {
		t.Fatal(err)
	}
	got, _ := st.GetPushSubscription(sub.Endpoint)
	if got.LastOKAt == nil {
		t.Fatal("last_ok_at not set")
	}
	if err := st.DeletePushSubscription(sub.Endpoint); err != nil {
		t.Fatal(err)
	}
	if _, err := st.GetPushSubscription(sub.Endpoint); err != ErrNotFound {
		t.Fatalf("after delete = %v", err)
	}
}
