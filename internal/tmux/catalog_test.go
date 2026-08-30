package tmux

import (
	"context"
	"testing"
)

func TestOptionCatalogCoversAllThreeScopes(t *testing.T) {
	m := New()
	if !m.Available() {
		t.Skip("tmux not installed")
	}
	cat, err := m.OptionCatalog(context.Background())
	if err != nil {
		t.Fatalf("catalog: %v", err)
	}
	byScope := map[string]int{}
	byName := map[string]CatalogEntry{}
	for _, e := range cat {
		byScope[e.Scope]++
		byName[e.Scope+"/"+e.Name] = e
	}
	// One well-known option per scope; counts vary by tmux version, presence
	// does not.
	for _, probe := range []string{
		ScopeServer + "/escape-time",
		ScopeSession + "/mouse",
		ScopeWindow + "/mode-keys",
	} {
		if _, ok := byName[probe]; !ok {
			t.Errorf("catalog is missing %s", probe)
		}
	}
	for scope, n := range byScope {
		if n < 10 {
			t.Errorf("scope %s has only %d options — the parse likely broke", scope, n)
		}
	}
}

func TestOptionCatalogFoldsArraysAndInfersKinds(t *testing.T) {
	m := New()
	if !m.Available() {
		t.Skip("tmux not installed")
	}
	cat, _ := m.OptionCatalog(context.Background())
	byName := map[string]CatalogEntry{}
	for _, e := range cat {
		byName[e.Scope+"/"+e.Name] = e
	}
	if e := byName[ScopeServer+"/command-alias"]; e.Kind != "array" {
		t.Errorf("command-alias kind = %q, want array (indexed rows folded)", e.Kind)
	}
	if _, ok := byName[ScopeServer+"/command-alias[0]"]; ok {
		t.Error("an indexed row leaked into the catalog")
	}
	if e := byName[ScopeSession+"/mouse"]; e.Kind != "bool" {
		t.Errorf("mouse kind = %q, want bool", e.Kind)
	}
	if e := byName[ScopeSession+"/history-limit"]; e.Kind != "number" {
		t.Errorf("history-limit kind = %q, want number", e.Kind)
	}
	if e := byName[ScopeSession+"/default-shell"]; e.Kind != "text" {
		t.Errorf("default-shell kind = %q, want text", e.Kind)
	}
}

func TestUnquote(t *testing.T) {
	for in, want := range map[string]string{
		`"lock -np"`:    "lock -np",
		`''`:            "",
		`plain`:         "plain",
		`"a \"b\" c"`:   `a "b" c`,
		`"back\\slash"`: `back\slash`,
	} {
		if got := unquote(in); got != want {
			t.Errorf("unquote(%s) = %q, want %q", in, got, want)
		}
	}
}

// A window option set through the scoped setter must land on the session's
// window, and unset must return it to the inherited value.
func TestScopedSetAndUnsetOnAWindowOption(t *testing.T) {
	m := New()
	if !m.Available() {
		t.Skip("tmux not installed")
	}
	ctx := context.Background()
	name := SessionName("catalog-scoped")
	if err := m.NewSession(ctx, name, t.TempDir(), "cat"); err != nil {
		t.Fatalf("session: %v", err)
	}
	t.Cleanup(func() { _ = m.KillSession(ctx, name) })

	if err := m.SetScopedOption(ctx, ScopeWindow, name, "mode-keys", "vi"); err != nil {
		t.Fatalf("set: %v", err)
	}
	out, err := m.run(ctx, "show-options", "-w", "-t", name+":", "-v", "mode-keys")
	if err != nil || out != "vi\n" {
		t.Fatalf("mode-keys = %q (%v), want vi", out, err)
	}
	if err := m.UnsetScopedOption(ctx, ScopeWindow, name, "mode-keys"); err != nil {
		t.Fatalf("unset: %v", err)
	}
	// After unset, the window-level value is gone (show without -g errors or
	// prints nothing for an unset option).
	out, _ = m.run(ctx, "show-options", "-w", "-t", name+":", "-v", "mode-keys")
	if out == "vi\n" {
		t.Fatal("unset did not remove the window-level value")
	}
}

// tmux itself is the validator: a bad value must come back as an error with
// tmux's own message, which is what the API surfaces to the panel.
func TestTmuxRejectsABadValueWithItsOwnWords(t *testing.T) {
	m := New()
	if !m.Available() {
		t.Skip("tmux not installed")
	}
	ctx := context.Background()
	name := SessionName("catalog-badval")
	if err := m.NewSession(ctx, name, t.TempDir(), "cat"); err != nil {
		t.Fatalf("session: %v", err)
	}
	t.Cleanup(func() { _ = m.KillSession(ctx, name) })

	err := m.SetScopedOption(ctx, ScopeSession, name, "mouse", "sideways")
	if err == nil {
		t.Fatal("tmux accepted mouse=sideways; expected its validation error")
	}
}
