package browser

import (
	"path/filepath"
	"strings"
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

func TestResolveCallerIsTheHouseIdentity(t *testing.T) {
	// ADR-0143: managed agent (id) → terminal (term id) → unmanaged, and
	// unmanaged reads the tab on screen — never more.
	st := testStore(t)
	if err := Save(st, "agent-1", Policy{Tier: "act"}); err != nil {
		t.Fatal(err)
	}
	if err := Save(st, TerminalPrefix+"term-9", Policy{Tier: "full"}); err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name        string
		agent, term string
		want        string
	}{
		{"managed agent wins over the terminal", "agent-1", "term-9", "act"},
		{"terminal principal", "", "term-9", "full"},
		{"agent without a grant", "agent-2", "", "read"},
		{"terminal without a grant", "", "term-8", "read"},
		{"unmanaged caller", "", "", "read"},
		{"ungranted agent next to a granted terminal", "agent-2", "term-9", "read"},
	}
	for _, tc := range cases {
		if got := ResolveCaller(st, tc.agent, tc.term).Tier; got != tc.want {
			t.Errorf("%s: %s, want %s", tc.name, got, tc.want)
		}
	}
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
	for _, name := range []string{"", "Runtime.evaluate", "Page.navigate", "full", "poke"} {
		if _, ok := VerbFor(name); ok {
			t.Fatalf("%q should not be a verb", name)
		}
	}
	// Read verbs stay read. Session drive verbs (ADR-0172) are act and do
	// not need a stored grant — the handler raises the policy. full is the
	// raw door (ADR-0144). A new verb has to be added here on purpose.
	// `history` is read-tier because it drives nothing, and it has its own
	// gate on top (ADR-0146).
	read := map[string]bool{"snapshot": true, "screenshot": true, "events": true, "history": true}
	full := map[string]bool{"cdp": true}
	for name, v := range Verbs() {
		if read[name] && v.Tier != "read" {
			t.Fatalf("%s: tier %q, want read", name, v.Tier)
		}
		if full[name] && (v.Tier != "full" || !v.Raw) {
			t.Fatalf("%s: tier %q raw %v, want full + raw", name, v.Tier, v.Raw)
		}
		if !read[name] && !full[name] && v.Tier != "act" {
			t.Fatalf("%s: tier %q, want act", name, v.Tier)
		}
		// A tool names a verb, never a method: no verb is a dotted name. The
		// raw verb carries no method at all — the caller supplies it.
		if strings.Contains(name, ".") {
			t.Fatalf("%q looks like a CDP method", name)
		}
		if v.Raw && v.Method != "" {
			t.Fatalf("%s: the raw verb must not pin a method, got %q", name, v.Method)
		}
	}
}
