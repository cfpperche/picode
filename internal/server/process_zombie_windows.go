//go:build windows

package server

import "os"

// processZombie on Windows, where this daemon does not run.
//
// PiCode's server runs on Linux and macOS, and inside WSL on Windows
// (ADR-0020); the Windows build exists so every package and test compiles
// there. syscall.Kill does not, which is how the !linux spelling of this
// function broke the Windows job — a reminder that "not Linux" is three
// platforms, not one.
//
// os.FindProcess always succeeds on Windows, so it proves nothing; a
// process this build cannot inspect is reported as not-a-zombie rather
// than declared dead, which is the direction that refuses to end something
// on a guess.
func processZombie(pid int) bool {
	if pid <= 0 {
		return true
	}
	_, err := os.FindProcess(pid)
	return err != nil
}
