package store

import (
	"strings"
	"testing"
)

func TestIntegrationSettingsLayersAndFallback(t *testing.T) {
	s := openTest(t)
	// A layer with no declaration is a fact, not an error, and the default
	// requires nothing beyond a fast-forward.
	if _, ok, err := s.IntegrationSettingsFor("ws1"); err != nil || ok {
		t.Fatalf("undeclared layer = ok:%v err:%v", ok, err)
	}
	def, err := s.EffectiveIntegrationSettings("ws1")
	if err != nil || !def.FFOnly || len(def.Checks) != 0 || def.FromScope != "default" {
		t.Fatalf("default = %+v (%v)", def, err)
	}
	machine, err := s.PutIntegrationSettings(MachineIntegrationScope, IntegrationSettingsMutation{FFOnly: false, Checks: []string{"go test ./..."}})
	if err != nil || machine.Version != 1 || machine.FFOnly {
		t.Fatalf("machine = %+v (%v)", machine, err)
	}
	ws, err := s.PutIntegrationSettings("ws1", IntegrationSettingsMutation{FFOnly: true, Checks: []string{"make ci"}})
	if err != nil || ws.Version != 1 || len(ws.Checks) != 1 {
		t.Fatalf("workspace = %+v (%v)", ws, err)
	}
	// Writing the same layer again moves its version; the layers stay separate.
	again, err := s.PutIntegrationSettings("ws1", IntegrationSettingsMutation{FFOnly: true, Checks: []string{"make ci", "go vet ./..."}})
	if err != nil || again.Version != 2 {
		t.Fatalf("second write = %+v (%v)", again, err)
	}
	if eff, err := s.EffectiveIntegrationSettings("ws1"); err != nil || eff.FromScope != "ws1" || len(eff.Checks) != 2 {
		t.Fatalf("workspace layer lost: %+v (%v)", eff, err)
	}
	// A workspace that declared nothing sees the machine default.
	if eff, err := s.EffectiveIntegrationSettings("ws2"); err != nil || eff.FromScope != MachineIntegrationScope || eff.FFOnly {
		t.Fatalf("machine fallback = %+v (%v)", eff, err)
	}
	// A workspace with no declaration of its own and no machine default gets
	// the built-in one, never an error.
	s2 := openTest(t)
	if eff, err := s2.EffectiveIntegrationSettings("ws1"); err != nil || eff.FromScope != "default" || !eff.FFOnly {
		t.Fatalf("built-in default = %+v (%v)", eff, err)
	}
}

func TestIntegrationSettingsShape(t *testing.T) {
	nine := make([]string, 9)
	for i := range nine {
		nine[i] = "make ci"
	}
	for _, tc := range []struct {
		name string
		m    IntegrationSettingsMutation
		want string
	}{
		{"too many checks", IntegrationSettingsMutation{Checks: nine}, "at most 8 checks"},
		{"an empty check", IntegrationSettingsMutation{Checks: []string{"  "}}, "must be one non-empty line"},
		{"a two-line check", IntegrationSettingsMutation{Checks: []string{"make ci\ngo vet"}}, "must be one non-empty line"},
		{"a check that is too long", IntegrationSettingsMutation{Checks: []string{strings.Repeat("x", 301)}}, "must be one non-empty line"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := openTest(t)
			if _, err := s.PutIntegrationSettings("ws1", tc.m); err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("err = %v, want %q", err, tc.want)
			}
			if err := ValidateIntegrationSettings(tc.m); err == nil {
				t.Fatal("ValidateIntegrationSettings accepted it")
			}
		})
	}
}

// The settings dialog is the first human writer: two tabs saving over each
// other must conflict, and "use the machine's" must drop the layer.
func TestIntegrationSettingsVersionAndInherit(t *testing.T) {
	s := openTest(t)
	first, err := s.PutIntegrationSettings("ws1", IntegrationSettingsMutation{FFOnly: true, ExpectedVersion: 0})
	if err != nil || first.Version != 1 {
		t.Fatalf("first = %+v (%v)", first, err)
	}
	if _, err := s.PutIntegrationSettings("ws1", IntegrationSettingsMutation{FFOnly: false, ExpectedVersion: 1}); err != nil {
		t.Fatalf("write at the read version = %v", err)
	}
	if _, err := s.PutIntegrationSettings("ws1", IntegrationSettingsMutation{FFOnly: true, ExpectedVersion: 1}); err != ErrIntegrationConflict {
		t.Fatalf("stale write = %v, want ErrIntegrationConflict", err)
	}
	if err := s.DeleteIntegrationSettings("ws1"); err != nil {
		t.Fatal(err)
	}
	if _, ok, _ := s.IntegrationSettingsFor("ws1"); ok {
		t.Fatal("the layer survived its delete")
	}
	if eff, _ := s.EffectiveIntegrationSettings("ws1"); eff.FromScope != "default" {
		t.Fatalf("after delete = %+v, want the built-in default", eff)
	}
	if err := s.DeleteIntegrationSettings("ws1"); err != nil {
		t.Fatalf("deleting an absent layer = %v, want nil", err)
	}
}

func TestListIntegrationSettings(t *testing.T) {
	s := openTest(t)
	if all, err := s.ListIntegrationSettings(); err != nil || len(all) != 0 {
		t.Fatalf("empty = %v (%v)", all, err)
	}
	if _, err := s.PutIntegrationSettings(MachineIntegrationScope, IntegrationSettingsMutation{FFOnly: true, Checks: []string{"git diff --check"}}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.PutIntegrationSettings("ws1", IntegrationSettingsMutation{FFOnly: false}); err != nil {
		t.Fatal(err)
	}
	all, err := s.ListIntegrationSettings()
	if err != nil || len(all) != 2 || len(all[""].Checks) != 1 || all["ws1"].FFOnly || all["ws1"].FromScope != "ws1" {
		t.Fatalf("layers = %+v (%v)", all, err)
	}
}

// The mode is declared, never guessed (ADR-0186), and provider mode carries no
// commands: the provider's own CI runs those.
func TestIntegrationModeIsDeclared(t *testing.T) {
	s := openTest(t)
	if _, err := s.PutIntegrationSettings("ws1", IntegrationSettingsMutation{Mode: "automatic", FFOnly: true}); err == nil ||
		!strings.Contains(err.Error(), `mode must be "provider" or "local"`) {
		t.Fatalf("unknown mode = %v", err)
	}
	if _, err := s.PutIntegrationSettings("ws1", IntegrationSettingsMutation{Mode: ModeProvider, FFOnly: true, Checks: []string{"make ci"}}); err == nil ||
		!strings.Contains(err.Error(), "provider mode runs the checks of its provider") {
		t.Fatalf("provider with commands = %v", err)
	}
	declared, err := s.PutIntegrationSettings("ws1", IntegrationSettingsMutation{Mode: ModeProvider, FFOnly: true})
	if err != nil || declared.Mode != ModeProvider || len(declared.Checks) != 0 {
		t.Fatalf("provider declaration = %+v (%v)", declared, err)
	}
	local, err := s.PutIntegrationSettings("ws2", IntegrationSettingsMutation{Mode: ModeLocal, FFOnly: true, Checks: []string{"make ci"}})
	if err != nil || local.Mode != ModeLocal {
		t.Fatalf("local declaration = %+v (%v)", local, err)
	}
	// A project that declared nothing has no mode, and the machine fallback does
	// not invent one.
	if eff, err := s.EffectiveIntegrationSettings("ws3"); err != nil || eff.Mode != "" {
		t.Fatalf("default mode = %+v (%v)", eff, err)
	}
	if eff, err := s.EffectiveIntegrationSettings("ws1"); err != nil || eff.Mode != ModeProvider {
		t.Fatalf("workspace mode = %+v (%v)", eff, err)
	}
}
