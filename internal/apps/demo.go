package apps

import (
	"context"
	"fmt"
	"strings"
)

// demoApp is the hidden QA app (PICODE_DEMO_APP=1). It exists to exercise
// every primitive end to end — grid badge, list, detail, form, actions —
// and is the manual-QA walk for the host pipeline. Never in BuiltIns(false).
type demoApp struct{}

func (demoApp) Manifest() Manifest {
	return Manifest{ID: "demo", Name: "Demo", Icon: "flask", APIVersion: APIVersion}
}

func (demoApp) Badge(context.Context, Host) (Badge, error) {
	return Badge{Count: 3}, nil
}

func (demoApp) View(_ context.Context, _ Host, path string) (View, error) {
	switch {
	case path == "":
		return demoRoot(), nil
	case strings.HasPrefix(path, "item/"):
		return demoItem(strings.TrimPrefix(path, "item/")), nil
	case path == "form":
		return demoForm(), nil
	}
	return View{}, fmt.Errorf("demo: no view at %q", path)
}

func demoRoot() View {
	return View{
		APIVersion: APIVersion,
		Title:      "Demo",
		Blocks: []Block{
			{Type: "detail", Markdown: "Every primitive the host renders, in one app. Rows navigate; buttons act."},
			{Type: "list", Items: []ListItem{
				{ID: "1", Title: "First item", Subtitle: "navigates to a detail view", Icon: "flask", Badge: "new", Path: "item/1"},
				{ID: "2", Title: "Second item", Subtitle: "has a row action", Path: "item/2", Actions: []Action{
					{ID: "wave", Label: "Wave", Args: map[string]string{"item": "2"}},
				}},
				{ID: "3", Title: "Third item", Subtitle: "plain row", Path: "item/3"},
			}},
			{Type: "actions", Actions: []Action{
				{ID: "open-form", Label: "Open form"},
			}},
		},
	}
}

func demoItem(n string) View {
	return View{
		APIVersion: APIVersion,
		Title:      "Item " + n,
		Blocks: []Block{
			{Type: "detail", Markdown: "## Item " + n + "\n\nMarkdown **detail** block with a [link](https://example.com) and `code`."},
			{Type: "actions", Actions: []Action{
				{ID: "toast", Label: "Toast me", Args: map[string]string{"item": n}},
				{ID: "reset", Label: "Reset item", Confirm: "Reset item " + n + "? This is the demo danger flow.", Danger: true, Args: map[string]string{"item": n}},
			}},
		},
	}
}

func demoForm() View {
	return View{
		APIVersion: APIVersion,
		Title:      "Demo form",
		Blocks: []Block{
			{Type: "form", Form: &Form{
				ID:     "submit-form",
				Submit: "Submit",
				Fields: []Field{
					{Name: "flavor", Method: "select", Title: "Flavor", Options: []string{"vanilla", "chocolate", "pistachio"}},
					{Name: "sure", Method: "confirm", Title: "Are you sure?"},
					{Name: "name", Method: "input", Title: "Name", Placeholder: "type a name"},
					{Name: "notes", Method: "editor", Title: "Notes", Prefill: "multi\nline"},
				},
			}},
		},
	}
}

func (d demoApp) Action(_ context.Context, _ Host, req ActionRequest) (ActionResult, error) {
	switch req.Action {
	case "wave":
		return ActionResult{Toast: "Item " + req.Args["item"] + " waves back"}, nil
	case "toast":
		return ActionResult{Toast: "Toast from item " + req.Args["item"]}, nil
	case "reset":
		v := demoItem(req.Args["item"])
		v.Title += " (reset)"
		return ActionResult{Toast: "Item reset", View: &v}, nil
	case "open-form":
		return ActionResult{Path: "form"}, nil
	case "submit-form":
		v := demoRoot()
		return ActionResult{Toast: "Form: " + req.Args["flavor"] + " for " + req.Args["name"], View: &v}, nil
	}
	return ActionResult{}, fmt.Errorf("demo: unknown action %q", req.Action)
}
