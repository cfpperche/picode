//go:build !unix

package install

// MutationLock is a no-op where flock does not exist. The documented paths
// (`make deploy`, `make desktop-restart`) still serialize through the
// Makefile's flock; this platform has no shell swap to protect anyway.
func MutationLock(path string, held bool) (release func(), err error) {
	return func() {}, nil
}
