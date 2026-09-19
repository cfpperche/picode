package server

import (
	"strings"
	"testing"
)

// TestAnnotationNoteIsTheHumanSentenceFirst pins the shape the agent reads:
// the sentence, then where it is, then the evidence. The styles block is
// labelled "Styles" — not "computed styles" — because after the style
// inspector it carries what the page had AND, behind a /* proposed */ marker,
// what the human wants instead (v2c step 5).
func TestAnnotationNoteIsTheHumanSentenceFirst(t *testing.T) {
	note := annotationNote(
		"https://example.com/page",
		"Example",
		"button#save",
		"Make this button primary",
		`<button id="save">Save</button>`,
		"color: rgb(0, 0, 0)\n/* proposed */\ncolor: #b00020",
		"annotation-1.png",
	)
	for _, want := range []string{
		"Make this button primary",
		"Page: https://example.com/page",
		"Title: Example",
		"Element: button#save",
		"Screenshot: annotation-1.png (same folder)",
		"HTML:\n```html",
		"Styles:\n```css",
		"/* proposed */",
	} {
		if !strings.Contains(note, want) {
			t.Fatalf("the note lost %q:\n%s", want, note)
		}
	}
	if strings.Index(note, "Make this button primary") > strings.Index(note, "Page:") {
		t.Fatal("the human's sentence must come before the page metadata")
	}
	if strings.Contains(note, "Computed styles") {
		t.Fatal("the block carries proposals now: the header must not promise computed-only values")
	}
}

// TestAnnotationNoteSaysWhenNothingWasWritten: an empty comment is not a
// blank note, and no crop means no dangling screenshot line.
func TestAnnotationNoteSaysWhenNothingWasWritten(t *testing.T) {
	note := annotationNote("https://example.com/", "", "", "", "", "", "")
	if !strings.Contains(note, "no comment written") {
		t.Fatalf("an empty comment must say so: %q", note)
	}
	if strings.Contains(note, "Screenshot:") || strings.Contains(note, "Styles:") || strings.Contains(note, "HTML:") {
		t.Fatalf("empty fields must not leave empty blocks: %q", note)
	}
}

// TestAnnotationNoteClipsHugeDOM plants the cap's promise: one enormous page
// must not cost the whole annotation.
func TestAnnotationNoteClipsHugeDOM(t *testing.T) {
	note := annotationNote("u", "", "s", "c", strings.Repeat("x", annotationTextCap+500), "", "")
	if !strings.Contains(note, "…[truncated]") {
		t.Fatal("an over-cap DOM is clipped and says so")
	}
	if len(note) > annotationTextCap+2000 {
		t.Fatalf("the note stays near the cap: %d bytes", len(note))
	}
}
