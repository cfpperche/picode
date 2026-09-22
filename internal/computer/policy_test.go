package computer

import (
	"path/filepath"
	"testing"

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

func TestGrantIsOffUnlessWrittenOn(t *testing.T) {
	st := openStore(t)
	if Resolve(st, "agent-1").Enabled {
		t.Fatal("a missing grant is on")
	}
	if Resolve(nil, "agent-1").Enabled || Resolve(st, " ").Enabled {
		t.Fatal("no store or no key is on")
	}
	if err := st.SetSetting(SettingPrefix+"agent-1", "not json"); err != nil {
		t.Fatal(err)
	}
	if Resolve(st, "agent-1").Enabled {
		t.Fatal("a broken grant is on")
	}
	if err := Save(st, "agent-1", true); err != nil {
		t.Fatal(err)
	}
	if !Resolve(st, "agent-1").Enabled {
		t.Fatal("a saved grant is off")
	}
	if err := Save(st, "agent-1", false); err != nil {
		t.Fatal(err)
	}
	if Resolve(st, "agent-1").Enabled {
		t.Fatal("a revoked grant is on")
	}
}

func TestResolveCallerIsTheHouseIdentity(t *testing.T) {
	st := openStore(t)
	_ = Save(st, "agent-1", true)
	_ = Save(st, "term:t-9", true)
	rows := []struct {
		agent, term string
		want        bool
	}{
		{"agent-1", "", true},
		{"agent-1", "t-9", true},  // the agent id wins
		{"agent-2", "t-9", false}, // and does not fall back to the terminal
		{"", "t-9", false},        // a terminal holds no grant (ADR-0184)
		{"", "t-8", false},
		{"", "", false}, // no identity, no grant
	}
	for _, r := range rows {
		if got := ResolveCaller(st, r.agent, r.term).Enabled; got != r.want {
			t.Errorf("ResolveCaller(%q, %q) = %v, want %v", r.agent, r.term, got, r.want)
		}
	}
}

func TestTheCatalogIsClosed(t *testing.T) {
	if got := len(Actions()); got != 23 {
		t.Fatalf("catalog has %d actions, want 23", got)
	}
	for _, name := range []string{"screenshot", " Left_Click ", "OPEN"} {
		if _, ok := ActionFor(name); !ok {
			t.Errorf("%q is not in the catalog", name)
		}
	}
	if n, ok := ActionFor("format_disk"); ok || n != "format_disk" {
		t.Errorf("a stranger got in: %q %v", n, ok)
	}
	if Timeout("wait") <= Timeout("left_click") {
		t.Error("wait deserves a longer timeout than a click")
	}
}
