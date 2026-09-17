package browser

import "testing"

// The raw-CDP decision table (ADR-0144), in the package that owns the rule:
// developer mode *and* the full tier, never one alone.
func TestAllowsRaw(t *testing.T) {
	cases := []struct {
		devMode bool
		tier    string
		want    bool
	}{
		{false, "full", false},
		{false, "act", false},
		{false, "read", false},
		{true, "read", false},
		{true, "act", false},
		{true, "full", true},
	}
	for _, tc := range cases {
		p := Policy{Tier: tc.tier}
		if got := p.AllowsRaw(tc.devMode); got != tc.want {
			t.Fatalf("AllowsRaw(dev=%v, tier=%s) = %v, want %v", tc.devMode, tc.tier, got, tc.want)
		}
	}
	// No tier at all (the zero policy) is never raw-capable, even with the
	// switch on: ranks are compared, not guessed.
	if (Policy{}).AllowsRaw(true) {
		t.Fatal("an empty tier reached raw CDP")
	}
}

// The catalog still gates the catalog: adding the raw verb must not widen
// what a lower tier can name.
func TestRawVerbNeedsFullTierOnTheCatalogPath(t *testing.T) {
	verb, ok := VerbFor("cdp")
	if !ok || !verb.Raw || verb.Tier != "full" {
		t.Fatalf("cdp verb = %+v ok=%v", verb, ok)
	}
	if verb.Method != "" {
		t.Fatalf("the raw verb must not carry a catalog method, got %q", verb.Method)
	}
	for _, tier := range []string{"read", "act"} {
		if (Policy{Tier: tier}).Allows(verb) {
			t.Fatalf("tier %s reaches the raw verb through Allows", tier)
		}
	}
	if !(Policy{Tier: "full"}).Allows(verb) {
		t.Fatal("full tier must reach the verb check; raw is refused by AllowsRaw, not here")
	}
}

func TestRawMethod(t *testing.T) {
	if _, err := RawMethod(map[string]any{}); err == nil {
		t.Fatal("a missing method was accepted")
	}
	if _, err := RawMethod(map[string]any{"method": "   "}); err == nil {
		t.Fatal("a blank method was accepted")
	}
	if _, err := RawMethod(map[string]any{"method": 42}); err == nil {
		t.Fatal("a non-string method was accepted")
	}
	got, err := RawMethod(map[string]any{"method": " Network.getAllCookies "})
	if err != nil || got != "Network.getAllCookies" {
		t.Fatalf("method = %q err = %v", got, err)
	}
}
