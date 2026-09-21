//go:build unix

package install

import (
	"fmt"
	"os"
	"syscall"
)

// MutationLock serializes every owner-grade restart on one machine:
// `picode deploy` (here), `make deploy` and `make desktop-restart` (through
// the same path in the Makefile). Two of them interleaving is not survivable
// in the general case — 2026-09-21: a forced deploy raced a desktop-restart's
// cargo build, WSL's IO wedged and the whole VM went down with all 21
// sessions, leaving the Windows resident with no daemon to talk to until a
// machine reboot. The make targets hold the lock for their whole body and
// hand it down through PICODE_MUTATION_LOCK_HELD=1 so the child here never
// flocks a file its own ancestor already holds — that would deadlock.
//
// The path is a contract with the Makefile's MUTATION_LOCK; a test reads the
// Makefile so the two cannot drift apart silently.
func MutationLock(path string, held bool) (release func(), err error) {
	if held {
		return func() {}, nil
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return nil, fmt.Errorf("mutation lock %s: %w", path, err)
	}
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		fmt.Printf("Waiting for %s — another deploy/desktop-restart is in flight…\n", path)
		if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX); err != nil {
			_ = f.Close()
			return nil, fmt.Errorf("mutation lock %s: %w", path, err)
		}
	}
	return func() { _ = f.Close() }, nil
}
