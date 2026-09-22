package oauth

import (
	"net"
	"testing"
	"time"
)

// An abandoned browser sign-in frees its callback port and the next login
// can start.
func TestLoopbackSigninTimesOut(t *testing.T) {
	old := loopbackTimeout
	loopbackTimeout = 50 * time.Millisecond
	t.Cleanup(func() { loopbackTimeout = old })
	if _, _, err := StartSink("anthropic", "", nil); err != nil {
		t.Skipf("callback port unavailable here: %v", err)
	}
	deadline := time.Now().Add(3 * time.Second)
	for {
		pending, done, msg := Status()
		if !pending && done {
			if msg == "" {
				t.Fatal("a timeout reported success")
			}
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("sign-in never timed out")
		}
		time.Sleep(20 * time.Millisecond)
	}
	for time.Now().Before(deadline) {
		if ln, err := net.Listen("tcp", "127.0.0.1:"+anthropicPort); err == nil {
			_ = ln.Close()
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("callback port still held after the timeout")
}
