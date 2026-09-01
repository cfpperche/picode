package apps

import (
	"context"
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
	if len(r.All()) != 2 {
		t.Fatalf("demo registry has %d apps, want 2 (inbox + demo)", len(r.All()))
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
	if len(prod) != 1 || prod[0].Manifest().ID != "inbox" {
		t.Fatalf("BuiltIns(false) = %v, want just inbox (demo must stay hidden)", prod)
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
