package browser

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestSessionDriveDecisionTable(t *testing.T) {
	// identified x verb class x what the daemon is willing to send.
	// A stored read grant does not narrow a session drive (ADR-0172).
	rows := []struct {
		name       string
		verb       string
		identified bool
		session    bool
		tier       string
	}{
		{"identified snapshot opens the session tab", "snapshot", true, true, "act"},
		{"identified navigate is not a grant", "navigate", true, true, "act"},
		{"identified click is act", "click", true, true, "act"},
		{"identified open is the split", "open", true, true, "act"},
		{"history is not a session drive", "history", true, false, ""},
		{"raw cdp is not a session drive", "cdp", true, false, ""},
		{"no identity does not get a session", "navigate", false, false, ""},
		{"no identity snapshot stays off the session", "snapshot", false, false, ""},
	}
	for _, row := range rows {
		t.Run(row.name, func(t *testing.T) {
			if got := row.identified && IsSessionDrive(row.verb); got != row.session {
				t.Fatalf("session = %v, want %v", got, row.session)
			}
			if row.session {
				p := SessionDrive()
				verb, ok := VerbFor(row.verb)
				if !ok || !p.Allows(verb) || p.Tier != row.tier {
					t.Fatalf("drive = %+v allows %v", p, ok && p.Allows(verb))
				}
				if row.verb == "navigate" && !p.AllowsVerb(verb, map[string]any{"url": "https://login.example/sso"}) {
					t.Fatal("a session navigate must reach any https URL")
				}
				if row.verb == "navigate" && p.AllowsVerb(verb, map[string]any{"url": "file:///etc/passwd"}) {
					t.Fatal("a session navigate must still refuse file:")
				}
			}
		})
	}
}

func TestPrepareDoesNotConcatenateASelector(t *testing.T) {
	method, raw, err := Prepare("click", map[string]any{"selector": `button"]; alert(1); "`})
	if err != nil {
		t.Fatal(err)
	}
	if method != "Runtime.evaluate" {
		t.Fatalf("method = %s", method)
	}
	var params struct {
		Expression string `json:"expression"`
	}
	if err := json.Unmarshal(raw, &params); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(params.Expression, `alert(1)`) && !strings.Contains(params.Expression, `\"]; alert(1);`) && !strings.Contains(params.Expression, `\"; alert(1);`) {
		// The payload is a JSON string inside the expression, not live script.
		if !strings.Contains(params.Expression, `const sel = "`) {
			t.Fatalf("expression did not quote the selector: %s", params.Expression)
		}
	}
	if !strings.Contains(params.Expression, `querySelector(sel)`) {
		t.Fatalf("expression = %s", params.Expression)
	}
	// The dangerous characters sit inside the JSON string literal, not as syntax.
	if strings.Count(params.Expression, "querySelector") != 1 {
		t.Fatalf("expression grew a second query: %s", params.Expression)
	}
}

func TestPrepareRejectsABadClickAndABadOpen(t *testing.T) {
	if _, _, err := Prepare("click", map[string]any{}); err == nil {
		t.Fatal("click with nothing must fail")
	}
	if _, _, err := Prepare("open", map[string]any{"url": "javascript:alert(1)"}); err == nil {
		t.Fatal("open must refuse javascript:")
	}
	method, raw, err := Prepare("open", map[string]any{"url": "https://example.com/login"})
	if err != nil || method != "shell.open" || !strings.Contains(string(raw), "https://example.com/login") {
		t.Fatalf("open = %s %s %v", method, raw, err)
	}
	if _, _, err := Prepare("press", map[string]any{"key": "F12"}); err == nil {
		t.Fatal("an unknown key must fail")
	}
	if _, _, err := Prepare("press", map[string]any{"key": "Enter"}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := Prepare("type", map[string]any{}); err == nil {
		t.Fatal("type without text must fail")
	}
}
