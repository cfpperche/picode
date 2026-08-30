package termopts

import (
	"reflect"
	"testing"
)

func ptr(s string) *string { return &s }

func TestResolveLayersLaterOverEarlier(t *testing.T) {
	got := Resolve(map[string]string{"mouse": "off"}, map[string]string{"mouse": "on"})
	if got["mouse"] != "on" {
		t.Fatalf("mouse = %q, want the last layer to win", got["mouse"])
	}
}

func TestResolveFallsBackToTheBuiltInDefault(t *testing.T) {
	got := Resolve(nil, map[string]string{})
	if !reflect.DeepEqual(got, Defaults()) {
		t.Fatalf("Resolve with nothing set = %v, want the defaults %v", got, Defaults())
	}
}

// A row written by a version that knew a flag this binary does not — a
// rollback, or a flag retired later — must not reach `tmux set-option`.
func TestResolveDropsWhatTheRegistryDoesNotKnow(t *testing.T) {
	got := Resolve(map[string]string{"mouse": "off", "history-limit": "50000"})
	if _, ok := got["history-limit"]; ok {
		t.Fatalf("unknown key survived Resolve: %v", got)
	}
	if got["mouse"] != "off" {
		t.Fatalf("mouse = %q, want the known key to survive alongside it", got["mouse"])
	}
}

// Same for a known key carrying a value it does not take: it falls back to the
// default rather than being passed through.
func TestResolveDropsAValueTheFlagDoesNotTake(t *testing.T) {
	got := Resolve(map[string]string{"mouse": "yes"})
	if got["mouse"] != "on" {
		t.Fatalf("mouse = %q, want the default when the stored value is not one of %v", got["mouse"], []string{"on", "off"})
	}
}

func TestValidateRefusesAnUnknownKey(t *testing.T) {
	if err := Validate(map[string]*string{"mouse-speed": ptr("fast")}); err == nil {
		t.Fatal("want an error for an unknown key, got nil")
	}
}

func TestValidateRefusesAValueTheFlagDoesNotTake(t *testing.T) {
	if err := Validate(map[string]*string{"mouse": ptr("maybe")}); err == nil {
		t.Fatal("want an error for a bad value, got nil")
	}
}

func TestValidateAcceptsAClear(t *testing.T) {
	if err := Validate(map[string]*string{"mouse": nil}); err != nil {
		t.Fatalf("clearing a field must be legal: %v", err)
	}
}

// Clearing has to remove the key. Writing the inherited value instead would
// pin it: the terminal would stop following a later change to the global and
// nothing in the panel would say so.
func TestApplyClearRemovesRatherThanPins(t *testing.T) {
	got := Apply(map[string]string{"mouse": "off"}, map[string]*string{"mouse": nil})
	if _, ok := got["mouse"]; ok {
		t.Fatalf("cleared field is still stored: %v", got)
	}
	if v := Resolve(map[string]string{"mouse": "off"}, got)["mouse"]; v != "off" {
		t.Fatalf("after the clear the terminal reads %q, want to inherit %q", v, "off")
	}
}

func TestApplyDoesNotMutateTheStoredLayer(t *testing.T) {
	base := map[string]string{"mouse": "off"}
	Apply(base, map[string]*string{"mouse": ptr("on")})
	if base["mouse"] != "off" {
		t.Fatalf("Apply wrote through to its input: %v", base)
	}
}

func TestFlagsCopyIsIndependent(t *testing.T) {
	got := Flags()
	if len(got) == 0 {
		t.Fatal("no flags registered")
	}
	got[0].Default = "tampered"
	if again := Flags(); again[0].Default == "tampered" {
		t.Fatal("Flags() handed out the registry itself")
	}
}

// Every flag must be self-consistent, or the panel renders a default it cannot
// select. Cheap to assert, and it fails the moment a second flag is added
// carelessly.
func TestEveryFlagOffersItsOwnDefault(t *testing.T) {
	for _, f := range Flags() {
		if f.Key == "" || f.Label == "" || f.Help == "" || f.Effect == "" {
			t.Errorf("%q: a flag needs a key, label, help and effect", f.Key)
		}
		if !f.accepts(f.Default) {
			t.Errorf("%q: default %q is not among %v", f.Key, f.Default, f.Values)
		}
	}
}
