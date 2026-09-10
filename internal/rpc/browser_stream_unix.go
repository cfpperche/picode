//go:build unix

// Unix half of the browser-stream rendezvous checks (ADR-0114): uid trust,
// O_NOFOLLOW opens and pid liveness. Mirrors the pi-browser-capture sidecar.

package rpc

import (
	"os"
	"syscall"
)

// streamRootTrust requires the socket root to be owned by this uid with
// mode 0700 (Node: st.uid === uid && (st.mode & 0o777) === 0o700).
func streamRootTrust(info os.FileInfo) bool {
	st, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return false
	}
	if int(st.Uid) != os.Getuid() {
		return false
	}
	return info.Mode().Perm() == 0o700
}

// rendezvousFileOwned requires the rendezvous file to be owned by this uid.
func rendezvousFileOwned(info os.FileInfo) bool {
	st, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return false
	}
	return int(st.Uid) == os.Getuid()
}

// openRendezvousFile opens with O_NOFOLLOW so a planted symlink is refused.
func openRendezvousFile(path string) (*os.File, bool) {
	fd, err := syscall.Open(path, syscall.O_RDONLY|syscall.O_NOFOLLOW, 0)
	if err != nil {
		return nil, false
	}
	return os.NewFile(uintptr(fd), path), true
}

// pidAlive mirrors Node's process.kill(pid, 0): only an error-free signal-0
// probe counts as alive (EPERM — someone else's process — is a refusal,
// matching the sidecar's throw-and-skip).
func pidAlive(pid int) bool {
	return syscall.Kill(pid, 0) == nil
}
