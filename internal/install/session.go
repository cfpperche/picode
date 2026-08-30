package install

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
)

// `systemctl --user` talks to the caller's own service manager over a socket
// whose location arrives in the environment. A login shell has it; a cron job,
// a script, or an agent's shell does not — and there systemctl fails with a
// message about a bus address that says nothing about what to do.
//
// Install and Deploy copy the binary *before* they restart anything, so that
// failure used to leave the new binary on disk with the old one still running:
// an update that looked done and was not. Two guards: fill the variables in
// when we can, and refuse before copying when we cannot.

const (
	runtimeDirEnv = "XDG_RUNTIME_DIR"
	busEnv        = "DBUS_SESSION_BUS_ADDRESS"
)

// ErrNoUserSession says the caller cannot reach its own service manager.
var ErrNoUserSession = errors.New("no systemd user session")

// sessionEnv returns the variables to add so `systemctl --user` can connect,
// or nil when the environment already carries them or nothing can be filled
// in. uid is the caller's, so `/run/user/<uid>` is by definition its own
// runtime directory — this never reaches for another account's socket.
func sessionEnv(getenv func(string) string, uid int, exists func(string) bool) []string {
	dir := getenv(runtimeDirEnv)
	var out []string
	if dir == "" {
		candidate := filepath.Join("/run/user", strconv.Itoa(uid))
		if !exists(candidate) {
			return nil // nothing to point at; the caller reports why
		}
		dir = candidate
		out = append(out, runtimeDirEnv+"="+dir)
	}
	if getenv(busEnv) == "" {
		bus := filepath.Join(dir, "bus")
		if exists(bus) {
			out = append(out, busEnv+"=unix:path="+bus)
		}
	}
	return out
}

// EnsureUserSession fails when `systemctl --user` has no chance of working, so
// callers can stop before copying a binary they cannot activate. A var, like
// Run, so tests can drive both outcomes.
var EnsureUserSession = func() error {
	return userSession(os.Getenv, os.Getuid(), pathExists, currentUser())
}

func userSession(getenv func(string) string, uid int, exists func(string) bool, user string) error {
	if getenv(runtimeDirEnv) != "" {
		return nil
	}
	dir := filepath.Join("/run/user", strconv.Itoa(uid))
	if exists(dir) {
		return nil // sessionEnv will fill it in
	}
	return fmt.Errorf("%w: %s is unset and %s does not exist.\n"+
		"Run this from your own login shell, or enable a session that outlives "+
		"login with:\n  loginctl enable-linger %s",
		ErrNoUserSession, runtimeDirEnv, dir, user)
}

func pathExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

func currentUser() string {
	if u := os.Getenv("USER"); u != "" {
		return u
	}
	return strconv.Itoa(os.Getuid())
}
