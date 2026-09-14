package browser

import (
	"path/filepath"
	"testing"

	"github.com/cfpperche/picode/internal/store"
)

func testStore(t *testing.T) *store.Store {
	t.Helper()
	st, err := store.Open(filepath.Join(t.TempDir(), "picode.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	return st
}

func TestDefaultIsReadOnTheTabOnScreen(t *testing.T) {
	// ADR-0134: no grant reads the tab the human has on screen, and nothing
	// else. The domains list stays empty — the target is the tab, not an origin.
	p := Default()
	if p.Tier != "read" || len(p.Domains) != 0 {
		t.Fatalf("default = %+v", p)
	}
}

func TestResolveFallsBackToTheDefault(t *testing.T) {
	st := testStore(t)
	cases := []struct {
		name    string
		agent   string
		setting string
	}{
		{"no agent", "", `{"tier":"full"}`},
		{"no setting", "agent-1", ""},
		{"broken json", "agent-1", `{"tier":`},
		{"unknown tier", "agent-1", `{"tier":"admin"}`},
	}
	for _, c := range cases {
		if c.setting != "" {
			if err := st.SetSetting(SettingPrefix+c.agent, c.setting); err != nil {
				t.Fatal(err)
			}
		}
		if got := Resolve(st, c.agent); got.Tier != Default().Tier || len(got.Domains) != 0 {
			t.Fatalf("%s: resolve = %+v, want the default", c.name, got)
		}
	}
}

func TestSaveAndResolveAGrant(t *testing.T) {
	st := testStore(t)
	want := Policy{Tier: "act", Domains: []string{"example.com", "localhost:*"}}
	if err := Save(st, "agent-1", want); err != nil {
		t.Fatal(err)
	}
	got := Resolve(st, "agent-1")
	if got.Tier != want.Tier || len(got.Domains) != 2 || got.Domains[0] != "example.com" {
		t.Fatalf("resolve = %+v", got)
	}
	if err := Save(st, "agent-1", Policy{Tier: "admin"}); err != ErrBadTier {
		t.Fatalf("save bad tier = %v", err)
	}
	// One agent's grant never leaks into another's.
	if other := Resolve(st, "agent-2"); other.Tier != Default().Tier || len(other.Domains) != 0 {
		t.Fatalf("agent-2 = %+v", other)
	}
}

func TestPolicyAllowsByTier(t *testing.T) {
	snapshot := Verb{Method: "Accessibility.getFullAXTree", Tier: "read"}
	evaluate := Verb{Method: "Runtime.evaluate", Tier: "act"}
	rows := []struct {
		tier  string
		verb  Verb
		allow bool
	}{
		{"read", snapshot, true},
		{"read", evaluate, false},
		{"act", snapshot, true},
		{"act", evaluate, true},
		{"full", evaluate, true},
		{"", snapshot, false},
		{"nonsense", snapshot, false},
	}
	for _, row := range rows {
		if got := (Policy{Tier: row.tier}).Allows(row.verb); got != row.allow {
			t.Fatalf("tier %q + %s = %v, want %v", row.tier, row.verb.Method, got, row.allow)
		}
	}
}

func TestVerbsAreClosedAndCaseInsensitive(t *testing.T) {
	for _, name := range []string{"snapshot", " Snapshot ", "SCREENSHOT", "events"} {
		if _, ok := VerbFor(name); !ok {
			t.Fatalf("%q is not a verb", name)
		}
	}
	for _, name := range []string{"", "evaluate", "navigate", "Runtime.evaluate", "full"} {
		if _, ok := VerbFor(name); ok {
			t.Fatalf("%q should not be a verb", name)
		}
	}
	// Every verb the daemon will ask for is read-tier today (ADR-0134).
	for name, v := range Verbs() {
		if v.Tier != "read" {
			t.Fatalf("%s: tier %q, want read until the grants exist", name, v.Tier)
		}
	}
}
