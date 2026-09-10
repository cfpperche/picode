//go:build !unix

// Non-unix half of the browser-stream rendezvous checks (ADR-0114).
// The socket root is unsupported on windows ("" from browserSocketRoot),
// so these paths only exist to compile; discovery returns nothing there.

package rpc

import "os"

func streamRootTrust(os.FileInfo) bool { return false }

func rendezvousFileOwned(os.FileInfo) bool { return false }

func openRendezvousFile(string) (*os.File, bool) { return nil, false }

func pidAlive(int) bool { return false }
