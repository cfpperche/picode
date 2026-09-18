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
	}
}
