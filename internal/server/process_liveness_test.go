package server

import (
	"os"
	"testing"
)

// This process is alive on every platform, and the identity it reports is
// its own. Saying otherwise is what broke macOS: processZombie read
// /proc/<pid>/stat and treated an unreadable file as "gone", which is the
// right answer on Linux — where /proc always exists, so its absence for one
// pid means the pid does — and wrong anywhere without /proc, where the file
// is never there for anyone.
//
// The consequence was not cosmetic. devServerProcessGone feeds Stop and the
// hide sweep, so on macOS a running dev server answered `stopped: true` and
// every hide was pruned on the next read. No test asked this question
// without a /proc in the room, so nothing caught it for three releases.
func TestThisProcessIsNotReportedGone(t *testing.T) {
	pid := os.Getpid()
	token := processStartToken(pid)
	if token == "" {
		t.Skip("no process identity on this platform: nothing to compare")
	}
	if devServerProcessGone(pid, token) {
		t.Fatal("the running test process is reported gone — Stop would answer stopped=true for a live server, and the hide sweep would forget every row")
	}
	if processZombie(pid) {
		t.Fatal("the running test process is reported as a zombie")
	}
}

// And the other direction still works: an identity that does not match is
// gone, which is the guard that stops a signal reaching a reused pid.
func TestAChangedIdentityIsGone(t *testing.T) {
	pid := os.Getpid()
	if !devServerProcessGone(pid, "not-the-token-this-process-has") {
		t.Fatal("a start token that does not match was accepted as the same process")
	}
	if !devServerProcessGone(0, "") {
		t.Fatal("pid 0 is not a process")
	}
}
