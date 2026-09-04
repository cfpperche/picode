package tmux

import (
	"context"
	"strings"
	"testing"
	"time"
)

func testOptionCatalog(t *testing.T) []CatalogEntry {
	t.Helper()
	m := requireTmux(t)
	ctx := context.Background()
	name := SessionName("catalog-" + time.Now().Format("150405-000000000"))
	if err := m.NewSession(ctx, name, t.TempDir(), "sleep", "30"); err != nil {
		t.Fatalf("session: %v", err)
	}
	t.Cleanup(func() { _ = m.KillSession(ctx, name) })
	cat, err := m.OptionCatalog(ctx)
	if err != nil {
		t.Fatalf("catalog: %v", err)
	}
	return cat
}

func TestOptionCatalogCoversAllThreeScopes(t *testing.T) {
	cat := testOptionCatalog(t)
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
	cat := testOptionCatalog(t)
	byName := map[string]CatalogEntry{}
	for _, e := range cat {
		byName[e.Scope+"/"+e.Name] = e
	}
	if e := byName[ScopeServer+"/command-alias"]; e.Kind != "array" {
		t.Errorf("command-alias kind = %q, want array (indexed rows folded)", e.Kind)
	} else if !strings.Contains(e.Value, "\n") || !strings.Contains(e.Value, "=") {
		// tmux ships several default aliases, so the folded row must carry
		// them as a block — an empty Value means the fold dropped the entries.
		t.Errorf("command-alias value = %q, want the entries joined as a block", e.Value)
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

func TestSplitArrayDropsBlanksAndKeepsOrder(t *testing.T) {
	got := SplitArray("one\n\ntwo\r\n   \nthree\n")
	want := []string{"one", "two", "three"}
	if len(got) != len(want) {
		t.Fatalf("SplitArray = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("SplitArray[%d] = %q, want %q", i, got[i], want[i])
		}
	}
	if out := SplitArray("  \n\n"); len(out) != 0 {
		t.Fatalf("blank block = %v, want no entries", out)
	}
}

// Rewriting an array must make the layer EXACTLY the new list: shrinking from
// three entries to two has to unset the third, or it survives every rewrite
// forever. This is the trap the per-index unsets exist for.
func TestSetArrayOptionReplacesTheLayerList(t *testing.T) {
	m := New()
	if !m.Available() {
		t.Skip("tmux not installed")
	}
	ctx := context.Background()
	name := SessionName("catalog-array")
	if err := m.NewSession(ctx, name, t.TempDir(), "cat"); err != nil {
		t.Fatalf("session: %v", err)
	}
	t.Cleanup(func() { _ = m.KillSession(ctx, name) })

	layer := func() []int {
		idx, _ := m.layerIndexes(ctx, ScopeSession, name, "update-environment")
		return idx
	}
	if err := m.SetArrayOption(ctx, ScopeSession, name, "update-environment", []string{"AAA", "BBB", "CCC"}); err != nil {
		t.Fatalf("set 3: %v", err)
	}
	if got := layer(); len(got) != 3 {
		t.Fatalf("after set 3, layer holds %v", got)
	}
	if err := m.SetArrayOption(ctx, ScopeSession, name, "update-environment", []string{"DDD", "EEE"}); err != nil {
		t.Fatalf("set 2: %v", err)
	}
	if got := layer(); len(got) != 2 {
		t.Fatalf("after shrinking to 2, layer holds %v — the stale index survived", got)
	}
	out, _ := m.run(ctx, "show-options", "-t", name+":", "update-environment")
	if !strings.Contains(out, "update-environment[0] DDD") || strings.Contains(out, "CCC") {
		t.Fatalf("layer content = %q, want DDD/EEE and no CCC", out)
	}
	if err := m.UnsetScopedOption(ctx, ScopeSession, name, "update-environment"); err != nil {
		t.Fatalf("unset: %v", err)
	}
	if got := layer(); len(got) != 0 {
		t.Fatalf("after unset, layer still holds %v", got)
	}
}
