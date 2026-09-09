package main

import "testing"

// Before this guard, any unrecognised argument fell through to serve(). An
// older picode asked to `provision` by a newer PiCode Desktop therefore came
// up as a second server — as root, with its own /root/.picode and its own
// port — instead of failing. So did a plain typo.
func TestDispatchDoesNotClaimUnknownCommands(t *testing.T) {
	unknown := []string{
		"provisoin", // the typo that started a root server
		"serve",     // plausible, and not a command
		"doctor",    // belongs to picode-desktop, not here
		"",
	}
	for _, cmd := range unknown {
		if dispatch(cmd, nil) {
			t.Errorf("dispatch(%q) = true, want false so main can reject it", cmd)
		}
	}
}

// help is the one command safe to run in a test: it only prints.
func TestDispatchClaimsKnownCommands(t *testing.T) {
	for _, cmd := range []string{"help", "-h", "--help"} {
		if !dispatch(cmd, nil) {
			t.Errorf("dispatch(%q) = false, want it handled", cmd)
		}
	}
}

// A version probe must never reach serve(): ADR-0098's install check runs
// `picode --version` through a login shell, and the leading dash used to
// fall through to the server.
func TestDispatchClaimsVersionProbes(t *testing.T) {
	for _, cmd := range []string{"version", "--version", "-v"} {
		if !dispatch(cmd, nil) {
			t.Errorf("dispatch(%q) = false, want it handled", cmd)
		}
	}
}
