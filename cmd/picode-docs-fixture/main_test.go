package main

import "testing"

// The directory name must be stable for a given address (captures compare
// pixel for pixel) and a single safe path element whatever the flag holds.
func TestPortSuffix(t *testing.T) {
	cases := []struct{ addr, want string }{
		{"127.0.0.1:18740", "18740"},
		{":18741", "18741"},
		{"[::1]:18742", "18742"},
		{"localhost", "localhost"},
		{"", "default"},
		{"host:odd chars", "odd_chars"},                // the port part wins when the address parses
		{"host/with odd chars", "host_with_odd_chars"}, // no port: the whole address, reduced
	}
	for _, tc := range cases {
		if got := portSuffix(tc.addr); got != tc.want {
			t.Errorf("portSuffix(%q) = %q, want %q", tc.addr, got, tc.want)
		}
	}
}
