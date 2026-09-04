package apps

import "fmt"

// The primitive vocabulary (ADR-0036): a View is a JSON tree the PiCode
// UI renders with its own components. Form fields reuse rpc.UIDialog's
// field names and method enum (select|confirm|input|editor) so app forms
// and pi extension dialogs stay one language.

// View is one screen of an app. Layout is a hint, not a container: the
// host decides how to arrange the blocks it gets.
type View struct {
	APIVersion int     `json:"apiVersion"`
	Title      string  `json:"title"`
	Layout     string  `json:"layout,omitempty"` // "" (stacked) | "split"
	Empty      string  `json:"empty,omitempty"`  // blankslate line when Blocks is empty
	Tabs       []Tab   `json:"tabs,omitempty"`   // optional segmented nav/filter strip, host-rendered
	Blocks     []Block `json:"blocks"`
}

// Tab is a navigation choice, not a block: clicking it navigates like
// ListItem.Path does. Deliberately its own type rather than reusing
// Action, which carries Confirm/Danger/Primary — fields that mean
// nothing for "go look at this other view."
type Tab struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	Path  string `json:"path"`            // navigates here on click; "" is the root view
	Badge string `json:"badge,omitempty"` // short count, e.g. "37"
}

// Block is one vertical section of a view. Type picks which field is
// meaningful; the rest stay empty.
type Block struct {
	Type     string     `json:"type"`               // "list" | "detail" | "form" | "actions"
	Title    string     `json:"title,omitempty"`    // section label above the block
	Meta     []string   `json:"meta,omitempty"`     // header meta strip, beside Title
	At       string     `json:"at,omitempty"`       // RFC3339, formatted by the host
	Pane     string     `json:"pane,omitempty"`     // "" | "list" | "detail" — split layout only
	Items    []ListItem `json:"items,omitempty"`    // list
	Markdown string     `json:"markdown,omitempty"` // detail
	Form     *Form      `json:"form,omitempty"`     // form
	Actions  []Action   `json:"actions,omitempty"`  // actions
}

// ListItem is one row. Path, when set, navigates the view there on click.
// Meta renders as a separated strip; At is RFC3339 the host formats
// itself (relative in the row, absolute on hover) — never pre-format.
type ListItem struct {
	ID       string   `json:"id"`
	Title    string   `json:"title"`
	Subtitle string   `json:"subtitle,omitempty"`
	Meta     []string `json:"meta,omitempty"`
	At       string   `json:"at,omitempty"`
	Icon     string   `json:"icon,omitempty"`
	Badge    string   `json:"badge,omitempty"` // short pill text on the row
	Tone     string   `json:"tone,omitempty"`  // "" | "info" | "ok" | "warn" | "danger"
	Unread   bool     `json:"unread,omitempty"`
	Path     string   `json:"path,omitempty"`
	Actions  []Action `json:"actions,omitempty"`
}

// Form posts its field values as the args of one action.
type Form struct {
	ID     string  `json:"id"`               // action id fired on submit
	Submit string  `json:"submit,omitempty"` // submit button label
	Fields []Field `json:"fields"`
	// Burst marks a TUI reply that runs through ADR-0059's temporary
	// control channel while the shell keeps the terminal tab in place.
	Burst bool `json:"burst,omitempty"`
}

// Field mirrors rpc.UIDialog.
type Field struct {
	Name        string   `json:"name"`
	Method      string   `json:"method"` // select | confirm | input | editor
	Title       string   `json:"title,omitempty"`
	Message     string   `json:"message,omitempty"`
	Options     []string `json:"options,omitempty"`
	Placeholder string   `json:"placeholder,omitempty"`
	Prefill     string   `json:"prefill,omitempty"`
}

// Action is a button. Confirm, when set, prompts before firing; Danger
// styles it destructive; Icon lets a row action render as a quiet glyph
// (the label stays as its accessible name).
type Action struct {
	ID      string            `json:"id"`
	Label   string            `json:"label"`
	Icon    string            `json:"icon,omitempty"`
	Primary bool              `json:"primary,omitempty"` // the decision, not merely the first button
	Confirm string            `json:"confirm,omitempty"`
	Danger  bool              `json:"danger,omitempty"`
	Args    map[string]string `json:"args,omitempty"`
}

// ActionRequest is the body of POST /api/apps/{id}/action.
type ActionRequest struct {
	Action string            `json:"action"`
	Path   string            `json:"path,omitempty"` // view path it was fired from
	Args   map[string]string `json:"args,omitempty"` // includes form field values
}

// ActionResult tells the client what to do next: show a toast, replace
// the current view, and/or navigate to a path (refetch).
type ActionResult struct {
	Toast string `json:"toast,omitempty"`
	View  *View  `json:"view,omitempty"`
	Path  string `json:"path,omitempty"`
	// Goto asks the shell to leave the app and navigate somewhere it
	// owns. "agent:<id>" opens the docked TUI; "agentburst:<id>:<gen>"
	// opens the same tab under its transient reply state — never chat.
	// Each shell resolves the directive to its own terminal surface.
	Goto string `json:"goto,omitempty"`
}

func validMethod(m string) bool {
	switch m {
	case "select", "confirm", "input", "editor":
		return true
	}
	return false
}

func validLayout(l string) bool {
	return l == "" || l == "split"
}

func validPane(p string) bool {
	return p == "" || p == "list" || p == "detail"
}

func validTone(t string) bool {
	switch t {
	case "", "info", "ok", "warn", "danger":
		return true
	}
	return false
}

// Validate enforces the vocabulary so a bad tree fails in tests, not in
// the renderer.
func (v View) Validate() error {
	if v.APIVersion != APIVersion {
		return fmt.Errorf("view: apiVersion %d, want %d", v.APIVersion, APIVersion)
	}
	if !validLayout(v.Layout) {
		return fmt.Errorf("view: layout %q unknown", v.Layout)
	}
	for i, t := range v.Tabs {
		if t.ID == "" || t.Label == "" {
			return fmt.Errorf("view: tab %d needs id and label", i)
		}
	}
	for i, b := range v.Blocks {
		if !validPane(b.Pane) {
			return fmt.Errorf("view: block %d pane %q unknown", i, b.Pane)
		}
		switch b.Type {
		case "list":
			for j, it := range b.Items {
				if it.ID == "" || it.Title == "" {
					return fmt.Errorf("view: block %d item %d needs id and title", i, j)
				}
				if !validTone(it.Tone) {
					return fmt.Errorf("view: block %d item %d tone %q unknown", i, j, it.Tone)
				}
				if err := validActions(it.Actions); err != nil {
					return fmt.Errorf("view: block %d item %d: %w", i, j, err)
				}
			}
		case "detail":
			if b.Markdown == "" {
				return fmt.Errorf("view: block %d detail needs markdown", i)
			}
		case "form":
			if b.Form == nil || b.Form.ID == "" {
				return fmt.Errorf("view: block %d form needs an id", i)
			}
			for j, f := range b.Form.Fields {
				if f.Name == "" {
					return fmt.Errorf("view: block %d field %d needs a name", i, j)
				}
				if !validMethod(f.Method) {
					return fmt.Errorf("view: block %d field %d method %q unknown", i, j, f.Method)
				}
			}
		case "actions":
			if err := validActions(b.Actions); err != nil {
				return fmt.Errorf("view: block %d: %w", i, err)
			}
		default:
			return fmt.Errorf("view: block %d type %q unknown", i, b.Type)
		}
	}
	return nil
}

func validActions(list []Action) error {
	for _, a := range list {
		if a.ID == "" || a.Label == "" {
			return fmt.Errorf("action needs id and label")
		}
	}
	return nil
}
