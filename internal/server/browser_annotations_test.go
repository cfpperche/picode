package server

import (
	"strings"
	"testing"
)

// TestAnnotationNoteIsTheWholeSendAsOneDocument pins the shape the agent
// reads: the count and the page, then one section per pin — the human's
// sentence as the heading, then where it is, then the evidence. The note is
// per Send, so any number of pins arrives as one context (ADR-0152
// amendment, owner 2026-09-19).
func TestAnnotationNoteIsTheWholeSendAsOneDocument(t *testing.T) {
	note := annotationNote("https://example.com/page", "Example", []annotationItem{
		{
			Selector: "button#save",
			Comment:  "Make this button primary",
			DOM:      `<button id="save">Save</button>`,
			CSS:      "color: rgb(0, 0, 0)\n/* proposed */\ncolor: #b00020",
			Shot:     "annotation-1-1.png",
		},
		{Selector: "h1", Comment: "This headline is too loud", Shot: "annotation-1-2.png"},
		{Selector: "footer", Comment: ""},
	})
	for _, want := range []string{
		"3 annotations on https://example.com/page",
		"Title: Example",
		"## 1. Make this button primary",
		"## 2. This headline is too loud",
		"## 3. Annotated element (no comment written).",
		"Element: button#save",
		"Screenshot: annotation-1-1.png (same folder)",
		"HTML:\n```html",
		"Styles:\n```css",
		"/* proposed */",
	} {
		if !strings.Contains(note, want) {
			t.Fatalf("the note lost %q:\n%s", want, note)
		}
	}
	if strings.Contains(note, "Computed styles") {
		t.Fatal("the block carries proposals now: the header must not promise computed-only values")
	}
	// Order: the count and the page first, then the pins in the order pinned.
	if strings.Index(note, "3 annotations") > strings.Index(note, "## 1.") {
		t.Fatal("the header comes first")
	}
	if strings.Index(note, "## 1.") > strings.Index(note, "## 3.") {
		t.Fatal("pins keep the order the human gave them")
	}
	// A pin with no crop gets no screenshot line at all.
	tail := note[strings.Index(note, "## 3."):]
	if strings.Contains(tail, "Screenshot:") {
		t.Fatal("a pin without a crop must not carry an empty screenshot line")
	}
}

// TestAnnotationNoteForOnePin keeps the singular honest: the common Send of a
// single note reads "1 annotation", not "1 annotations".
func TestAnnotationNoteForOnePin(t *testing.T) {
	note := annotationNote("https://example.com/", "", []annotationItem{{Selector: "a", Comment: "Fix this"}})
	if !strings.HasPrefix(note, "1 annotation on https://example.com/\n") {
		t.Fatalf("singular header expected:\n%s", note)
	}
	if strings.Contains(note, "Screenshot:") || strings.Contains(note, "Styles:") || strings.Contains(note, "HTML:") {
		t.Fatalf("empty fields must not leave empty blocks: %q", note)
	}
}

// TestAnnotationNoteClipsHugeDOM plants the cap's promise: one enormous page
// must not cost the whole annotation (nor its neighbours).
func TestAnnotationNoteClipsHugeDOM(t *testing.T) {
	note := annotationNote("u", "", []annotationItem{
		{Selector: "s", Comment: "c", DOM: strings.Repeat("x", annotationTextCap+500)},
		{Selector: "s2", Comment: "kept"},
	})
	if !strings.Contains(note, "…[truncated]") {
		t.Fatal("an over-cap DOM is clipped and says so")
	}
	if !strings.Contains(note, "## 2. kept") {
		t.Fatal("a clipped pin must not swallow the pins after it")
	}
	if len(note) > annotationTextCap+2000 {
		t.Fatalf("the note stays near the cap: %d bytes", len(note))
	}
}
