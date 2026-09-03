package store

import (
	"errors"
	"strings"
	"testing"
)

func TestValidateChecklistItems(t *testing.T) {
	t.Run("normalizes whitespace, case, and default status", func(t *testing.T) {
		out, err := ValidateChecklistItems([]ChecklistItem{
			{Text: "read   the code", Status: ""},
			{Text: "edit", Status: "IN-Progress"},
		})
		if err != nil {
			t.Fatal(err)
		}
		if out[0].Text != "read the code" || out[0].Status != "pending" {
			t.Fatalf("first = %+v", out[0])
		}
		if out[1].Status != "in-progress" {
			t.Fatalf("second = %+v", out[1])
		}
	})
	t.Run("rejects over the cap and empty text and unknown status", func(t *testing.T) {
		many := make([]ChecklistItem, maxChecklistItems+1)
		for i := range many {
			many[i] = ChecklistItem{Text: "x"}
		}
		if _, err := ValidateChecklistItems(many); err == nil {
			t.Fatal("over-cap list accepted")
		}
		if _, err := ValidateChecklistItems([]ChecklistItem{{Text: "  "}}); err == nil {
			t.Fatal("empty text accepted")
		}
		if _, err := ValidateChecklistItems([]ChecklistItem{{Text: "x", Status: "done"}}); err == nil {
			t.Fatal("unknown status accepted")
		}
		if _, err := ValidateChecklistItems(nil); err != nil {
			t.Fatalf("nil items (absent marker) rejected: %v", err)
		}
	})
	t.Run("truncates by code points, not bytes", func(t *testing.T) {
		text := strings.Repeat("a", maxChecklistText-1) + "🎉" // cuts inside the emoji if bytes
		out, err := ValidateChecklistItems([]ChecklistItem{{Text: text}})
		if err != nil {
			t.Fatal(err)
		}
		got := []rune(out[0].Text)
		if len(got) != maxChecklistText {
			t.Fatalf("truncated to %d runes, want %d", len(got), maxChecklistText)
		}
		if got[len(got)-1] == '\uFFFD' {
			t.Fatal("truncation split a UTF-8 rune")
		}
	})
}

func TestChecklistLevel(t *testing.T) {
	cases := []struct {
		name string
		op   *string
		raw  string
		want string
	}{
		{"default", nil, "", ChecklistChanges},
		{"explicit changes", nil, "changes", ChecklistChanges},
		{"always", nil, "always", ChecklistAlways},
		{"never", nil, "never", ChecklistNever},
		{"garbage falls back to changes", nil, "sometimes", ChecklistChanges},
		{"read-only forces never", strptr(OpModeReadonly), "always", ChecklistNever},
		{"read-only forces never even for garbage", strptr(OpModeReadonly), "junk", ChecklistNever},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			a := Agent{OpMode: tc.op, Checklist: tc.raw}
			if got := a.ChecklistLevel(); got != tc.want {
				t.Fatalf("ChecklistLevel() = %q, want %q", got, tc.want)
			}
		})
	}
}

func strptr(s string) *string { return &s }

func TestSetAndClearChecklist(t *testing.T) {
	s := openTest(t)
	a, err := s.AddAgent(FreeWorkspaceID, "planner", "")
	if err != nil {
		t.Fatal(err)
	}

	if _, err := s.SetChecklist(a.ID, "s1", []ChecklistItem{{Text: "read", Status: "completed"}, {Text: "edit"}}, false); err != nil {
		t.Fatal(err)
	}
	c, err := s.GetChecklist(a.ID)
	if err != nil {
		t.Fatal(err)
	}
	if c.SessionID != "s1" || len(c.Items) != 2 || c.Absent {
		t.Fatalf("set = %+v", c)
	}

	// An absent marker is a row of its own (the shells render "No checklist").
	if _, err := s.SetChecklist(a.ID, "s1", nil, true); err != nil {
		t.Fatal(err)
	}
	if c, _ = s.GetChecklist(a.ID); !c.Absent || len(c.Items) != 0 {
		t.Fatalf("absent = %+v", c)
	}

	// Clear drops the row; the event data is the empty state shells already
	// render as silence.
	cleared, err := s.ClearChecklist(a.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(cleared.Items) != 0 || cleared.Absent {
		t.Fatalf("cleared = %+v", cleared)
	}
	if _, err := s.GetChecklist(a.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("get after clear = %v, want ErrNotFound", err)
	}
	// Idempotent.
	if _, err := s.ClearChecklist(a.ID); err != nil {
		t.Fatalf("second clear: %v", err)
	}
	// Unknown agent stays an error.
	if _, err := s.ClearChecklist("nope"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("clear unknown = %v, want ErrNotFound", err)
	}

	list, err := s.ListChecklists()
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 0 {
		t.Fatalf("list after clear = %+v", list)
	}
}
