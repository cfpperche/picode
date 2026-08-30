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

func TestResolveFallsBackToTheBuiltInDefaults(t *testing.T) {
	got := Resolve(nil, map[string]string{})
	if !reflect.DeepEqual(got, Defaults()) {
		t.Fatalf("Resolve with nothing set = %v, want %v", got, Defaults())
	}
}

// The defaults now carry what used to be forced in code. If one goes missing
// the force is gone with nothing replacing it — status bars reappear,
// clipboard dies — so pin the set.
func TestDefaultsCarryTheFormerlyForcedOptions(t *testing.T) {
	d := Defaults()
	for key, want := range map[string]string{
		"mouse":                "on",
		"status":               "off",
		"allow-passthrough":    "on",
		"extended-keys":        "on",
		"extended-keys-format": "xterm",
	} {
		if d[key] != want {
			t.Errorf("default %s = %q, want %q", key, d[key], want)
		}
	}
}

// The registry used to be a whitelist; now it is a curated tier over an open
// catalog. A non-curated key must pass through Resolve untouched — it was
// validated against the live catalog when stored, and dropping it here would
// silently unset a real option the user chose.
func TestResolvePassesNonCuratedKeysThrough(t *testing.T) {
	got := Resolve(map[string]string{"history-limit": "50000"})
	if got["history-limit"] != "50000" {
		t.Fatalf("history-limit = %q, want the stored value passed through", got["history-limit"])
	}
}

// Curated keys keep their enum check: a stored value the flag no longer
// takes falls back to the default rather than reaching tmux.
func TestResolveDropsABadValueForACuratedFlag(t *testing.T) {
	got := Resolve(map[string]string{"mouse": "yes"})
	if got["mouse"] != "on" {
		t.Fatalf("mouse = %q, want the default when the stored value is not one of on/off", got["mouse"])
	}
}

func TestValidateCuratedRefusesABadValue(t *testing.T) {
	if err := ValidateCurated(map[string]*string{"mouse": ptr("maybe")}); err == nil {
		t.Fatal("want an error for a bad curated value, got nil")
	}
}

func TestValidateCuratedLetsNonCuratedKeysPass(t *testing.T) {
	if err := ValidateCurated(map[string]*string{"history-limit": ptr("50000")}); err != nil {
		t.Fatalf("non-curated keys are the catalog's to validate, got %v", err)
	}
}

func TestValidateCuratedAcceptsAClear(t *testing.T) {
	if err := ValidateCurated(map[string]*string{"mouse": nil}); err != nil {
		t.Fatalf("clearing a field must be legal: %v", err)
	}
}

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

func TestDangerForCoversBothTiers(t *testing.T) {
	if DangerFor("extended-keys") == "" {
		t.Error("a curated flag's danger is empty")
	}
	if DangerFor("destroy-unattached") == "" {
		t.Error("a known-dangerous catalog option has no warning")
	}
	if DangerFor("history-limit") != "" {
		t.Error("a harmless option grew a warning")
	}
}

func TestEveryFlagIsSelfConsistent(t *testing.T) {
	for _, f := range Flags() {
		if f.Key == "" || f.Label == "" || f.Help == "" || f.Effect == "" || f.Scope == "" {
			t.Errorf("%q: a flag needs key, scope, label, help and effect", f.Key)
		}
		if !f.accepts(f.Default) {
			t.Errorf("%q: default %q is not among %v", f.Key, f.Default, f.Values)
		}
	}
}

func TestFlagsCopyIsIndependent(t *testing.T) {
	got := Flags()
	got[0].Default = "tampered"
	if again := Flags(); again[0].Default == "tampered" {
		t.Fatal("Flags() handed out the registry itself")
	}
}
