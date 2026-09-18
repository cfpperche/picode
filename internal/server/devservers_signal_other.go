//go:build !linux

package server

// Nothing to signal here: the owner walk needs /proc, so on this platform no
// row ever carries a pid and the panel offers no Stop to begin with. The
// refusal exists so a hand-made request gets a sentence, not a panic.
func signalDevServerProcess(pid int, force bool) error {
	return errDevServerStopUnsupported
}
