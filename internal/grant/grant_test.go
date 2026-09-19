package grant

import "testing"

func TestKeyIsTheHouseIdentityRule(t *testing.T) {
	rows := []struct {
		agent, term, want string
	}{
		{"agent-1", "", "agent-1"},
		{"agent-1", "t-9", "agent-1"}, // the agent id wins over a terminal id
		{"", "t-9", "term:t-9"},
		{"  ", "t-9", "term:t-9"}, // whitespace is no identity
		{"", "", ""},
		{" ", " ", ""},
	}
	for _, r := range rows {
		if got := Key(r.agent, r.term); got != r.want {
			t.Errorf("Key(%q, %q) = %q, want %q", r.agent, r.term, got, r.want)
		}
		p := FromIDs(r.agent, r.term)
		if p.Key() != r.want {
			t.Errorf("FromIDs(%q, %q).Key() = %q, want %q", r.agent, r.term, p.Key(), r.want)
		}
		if ParseKey(r.want).Key() != r.want {
			t.Errorf("ParseKey(%q).Key() = %q", r.want, ParseKey(r.want).Key())
		}
	}
}

func TestPrincipalKinds(t *testing.T) {
	a := FromIDs("ag-1", "t-9")
	if a.Kind != KindAgent || a.ID != "ag-1" {
		t.Fatalf("agent wins: %+v", a)
	}
	term := FromIDs("", "t-9")
	if term.Kind != KindTerminal || term.ID != "t-9" || term.Key() != "term:t-9" {
		t.Fatalf("terminal: %+v key %q", term, term.Key())
	}
	if ParseKey("term:").Kind != KindNone {
		t.Fatalf("empty terminal id after prefix is unmanaged")
	}
}
