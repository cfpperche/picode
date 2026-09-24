package apps

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestRegistry(t *testing.T) {
	var nilReg *Registry
	if got := nilReg.All(); len(got) != 0 {
		t.Fatalf("nil registry All() = %v, want empty", got)
	}
	if _, ok := nilReg.Find("demo"); ok {
		t.Fatalf("nil registry Find() = ok, want miss")
	}

	r := NewRegistry(BuiltIns(true)...)
	if len(r.All()) != 5 {
		t.Fatalf("demo registry has %d apps, want 5 (docker + canvas + tmux + demo + demo-native)", len(r.All()))
	}
	a, ok := r.Find("demo")
	if !ok {
		t.Fatalf("Find(demo) missed")
	}
	m := a.Manifest()
	if m.ID != "demo" || m.Name == "" || m.Icon == "" || m.APIVersion != APIVersion {
		t.Fatalf("demo manifest incomplete: %+v", m)
	}
	if _, ok := r.Find("nope"); ok {
		t.Fatalf("Find(nope) = ok, want miss")
	}
	prod := BuiltIns(false)
	if len(prod) != 3 || prod[0].Manifest().ID != "docker" || prod[1].Manifest().ID != "canvas" || prod[2].Manifest().ID != "tmux" {
		t.Fatalf("BuiltIns(false) = %v, want docker, canvas and tmux (both demos must stay hidden)", prod)
	}
}

// The Canvas (plan docs/plans/matrix-app.md, ADR-0108/0109) ships in every
// build as a native surface: no badge, one honest primitives line, no
// action — the desktop registers the id and renders the plane itself.
func TestCanvasApp(t *testing.T) {
	a, ok := NewRegistry(BuiltIns(false)...).Find("canvas")
	if !ok {
		t.Fatalf("canvas missing from BuiltIns(false)")
	}
	m := a.Manifest()
	if m.ID != "canvas" || m.Name != "Canvas" || m.Icon != "canvas" || m.APIVersion != APIVersion || m.Surface != SurfaceNative {
		t.Fatalf("canvas manifest = %+v", m)
	}
	ctx := context.Background()
	if b, err := a.Badge(ctx, Host{}); err != nil || b != (Badge{}) {
		t.Fatalf("Badge = %+v, %v (want none: plan §4.10)", b, err)
	}
	v, err := a.View(ctx, Host{}, "")
	if err != nil {
		t.Fatalf("View error: %v", err)
	}
	if err := v.Validate(); err != nil {
		t.Fatalf("View invalid: %v", err)
	}
	if len(v.Blocks) != 1 || !strings.Contains(v.Blocks[0].Markdown, "Canvas opens on the desktop") {
		t.Fatalf("View = %+v", v)
	}
	if _, err := a.Action(ctx, Host{}, ActionRequest{Action: "open"}); err == nil {
		t.Fatalf("Action = nil error, want refusal")
	}
}

// The tmux app (docs/plans/tmux-app.md; ADR-0133 as amended 2026-09-14): the
// server inventory as a PRIMITIVES app on the Docker mold — no native body,
// no badge, and the frozen vocabulary doing the rendering. The amendment is
// the point of the manifest assertions: a surface field here would put the
// custom component back over other tabs.
func TestTmuxApp(t *testing.T) {
	a, ok := NewRegistry(BuiltIns(false)...).Find("tmux")
	if !ok {
		t.Fatalf("tmux missing from BuiltIns(false)")
	}
	m := a.Manifest()
	if m.ID != "tmux" || m.Name != "tmux" || m.Icon != "tmux" || m.APIVersion != APIVersion {
		t.Fatalf("tmux manifest = %+v", m)
	}
	if m.Surface != "" {
		t.Fatalf("tmux manifest Surface = %q, want primitives (the 2026-09-14 amendment)", m.Surface)
	}
	ctx := context.Background()
	if b, err := a.Badge(ctx, Host{}); err != nil || b != (Badge{}) {
		t.Fatalf("Badge = %+v, %v (want none)", b, err)
	}
	// No tmux source: the honest blankslate, never a 500.
	v, err := a.View(ctx, Host{}, "")
	if err != nil {
		t.Fatalf("View error: %v", err)
	}
	if err := v.Validate(); err != nil {
		t.Fatalf("View invalid: %v", err)
	}
	if v.Empty == "" || len(v.Blocks) != 0 {
		t.Fatalf("View = %+v, want the blankslate", v)
	}
	if _, err := a.Action(ctx, Host{}, ActionRequest{Action: "open"}); err == nil {
		t.Fatalf("Action = nil error, want refusal")
	}
}

// The native surface (ADR-0109): the manifest carries surface only when
// set, the primitives routes answer one honest line, and nothing acts.
func TestNativeDemoApp(t *testing.T) {
	a := nativeDemoApp{}
	ctx := context.Background()
	m := a.Manifest()
	if m.ID != "demo-native" || m.Name == "" || m.Icon == "" || m.APIVersion != APIVersion || m.Surface != SurfaceNative {
		t.Fatalf("native demo manifest = %+v", m)
	}
	raw, err := json.Marshal(m)
	if err != nil || !strings.Contains(string(raw), `"surface":"native"`) {
		t.Fatalf("native manifest JSON = %s, %v (want surface on the wire)", raw, err)
	}
	// A primitives app's manifest must look exactly as it did before the
	// field existed: an older client never sees a key it does not know.
	raw, _ = json.Marshal(demoApp{}.Manifest())
	if strings.Contains(string(raw), "surface") {
		t.Fatalf("primitives manifest JSON = %s (surface must be omitted)", raw)
	}

	for _, path := range []string{"", "anything/deeper"} {
		v, err := a.View(ctx, Host{}, path)
		if err != nil {
			t.Fatalf("View(%q) error: %v", path, err)
		}
		if err := v.Validate(); err != nil {
			t.Fatalf("View(%q) invalid: %v", path, err)
		}
		if len(v.Blocks) != 1 || v.Blocks[0].Type != "detail" || !strings.Contains(v.Blocks[0].Markdown, "opens on the desktop") {
			t.Fatalf("View(%q) = %+v, want one detail block saying it opens on the desktop", path, v)
		}
	}
	if b, err := a.Badge(ctx, Host{}); err != nil || b != (Badge{}) {
		t.Fatalf("Badge = %+v, %v (want none)", b, err)
	}
	if _, err := a.Action(ctx, Host{}, ActionRequest{Action: "open"}); err == nil {
		t.Fatalf("Action = nil error, want refusal")
	}
}

func TestViewValidate(t *testing.T) {
	ok := func(v View) View { return v }
	cases := []struct {
		name string
		v    View
		want string // "" = valid; otherwise substring of the error
	}{
		{"valid list", ok(View{APIVersion: 1, Blocks: []Block{{Type: "list", Items: []ListItem{{ID: "a", Title: "A"}}}}}), ""},
		{"wrong version", View{APIVersion: 2}, "apiVersion"},
		{"unknown block", View{APIVersion: 1, Blocks: []Block{{Type: "video"}}}, "unknown"},
		{"item missing title", View{APIVersion: 1, Blocks: []Block{{Type: "list", Items: []ListItem{{ID: "a"}}}}}, "id and title"},
		{"detail empty", View{APIVersion: 1, Blocks: []Block{{Type: "detail"}}}, "markdown"},
		{"form no id", View{APIVersion: 1, Blocks: []Block{{Type: "form", Form: &Form{}}}}, "form needs an id"},
		{"field bad method", View{APIVersion: 1, Blocks: []Block{{Type: "form", Form: &Form{ID: "f", Fields: []Field{{Name: "x", Method: "slider"}}}}}}, "unknown"},
		{"field no name", View{APIVersion: 1, Blocks: []Block{{Type: "form", Form: &Form{ID: "f", Fields: []Field{{Method: "input"}}}}}}, "needs a name"},
		{"action no label", View{APIVersion: 1, Blocks: []Block{{Type: "actions", Actions: []Action{{ID: "x"}}}}}, "id and label"},
		{"split layout", ok(View{APIVersion: 1, Layout: "split", Blocks: []Block{{Type: "detail", Pane: "detail", Markdown: "x"}}}), ""},
		{"bad layout", View{APIVersion: 1, Layout: "carousel"}, "layout"},
		{"bad pane", View{APIVersion: 1, Blocks: []Block{{Type: "detail", Pane: "middle", Markdown: "x"}}}, "pane"},
		{"bad tone", View{APIVersion: 1, Blocks: []Block{{Type: "list", Items: []ListItem{{ID: "a", Title: "A", Tone: "chartreuse"}}}}}, "tone"},
		{"good tone", ok(View{APIVersion: 1, Blocks: []Block{{Type: "list", Items: []ListItem{{ID: "a", Title: "A", Tone: "warn", Unread: true, At: "2026-09-01T00:00:00Z", Meta: []string{"a", "b"}}}}}}), ""},
		{"good tabs", ok(View{APIVersion: 1, Tabs: []Tab{{ID: "active", Label: "Active", Path: ""}, {ID: "done", Label: "Done", Path: "done", Badge: "3"}}}), ""},
		{"tab missing label", View{APIVersion: 1, Tabs: []Tab{{ID: "x"}}}, "needs id and label"},
		{"tab missing id", View{APIVersion: 1, Tabs: []Tab{{Label: "X"}}}, "needs id and label"},
	}
	for _, tc := range cases {
		err := tc.v.Validate()
		if tc.want == "" {
			if err != nil {
				t.Fatalf("%s: Validate() = %v, want nil", tc.name, err)
			}
			continue
		}
		if err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Fatalf("%s: Validate() = %v, want error containing %q", tc.name, err, tc.want)
		}
	}
}

func TestDemoViewsValidate(t *testing.T) {
	d := demoApp{}
	for _, path := range []string{"", "item/1", "item/2", "form"} {
		v, err := d.View(context.Background(), Host{}, path)
		if err != nil {
			t.Fatalf("View(%q) error: %v", path, err)
		}
		if err := v.Validate(); err != nil {
			t.Fatalf("View(%q) invalid: %v", path, err)
		}
	}
	if _, err := d.View(context.Background(), Host{}, "nope"); err == nil {
		t.Fatalf("View(nope) = nil error, want failure")
	}
}

func TestDemoActions(t *testing.T) {
	d := demoApp{}
	ctx := context.Background()

	res, err := d.Action(ctx, Host{}, ActionRequest{Action: "toast", Args: map[string]string{"item": "1"}})
	if err != nil || res.Toast == "" {
		t.Fatalf("toast action = %+v, %v", res, err)
	}
	res, err = d.Action(ctx, Host{}, ActionRequest{Action: "reset", Args: map[string]string{"item": "1"}})
	if err != nil || res.View == nil {
		t.Fatalf("reset action = %+v, %v (want replacement view)", res, err)
	}
	if err := res.View.Validate(); err != nil {
		t.Fatalf("reset view invalid: %v", err)
	}
	res, err = d.Action(ctx, Host{}, ActionRequest{Action: "open-form"})
	if err != nil || res.Path != "form" {
		t.Fatalf("open-form = %+v, %v (want Path form)", res, err)
	}
	if _, err := d.Action(ctx, Host{}, ActionRequest{Action: "nope"}); err == nil {
		t.Fatalf("unknown action = nil error, want failure")
	}
}
