package llamaservice

import (
	"os"
	"sync"
)

// Cleanup and the cache list verify multi-gigabyte files by SHA-256, and they
// did it holding the service lock: one cleanup of large GGUFs stalled status,
// Configure and every model operation for minutes (2026-09-25 review). The
// hash is now taken before the lock (warmHashes) and remembered by the file's
// identity; under the lock a file whose identity is unchanged — same file
// (device and inode), size and modification time, the identity models.go
// already trusts for downloads — answers from memory, and any other file is
// hashed there as before. Nothing is removed on a hash PiCode did not take.

type hashed struct {
	info os.FileInfo
	sha  string
	size int64
}

// hashMemoCap bounds the memory; past it the whole memo is dropped (the next
// pass hashes again), which is simpler than an eviction order and rare.
const hashMemoCap = 512

var (
	hashMu   sync.Mutex
	hashMemo = map[string]hashed{}
	// hashFile is hashRegular; tests replace it to hold a hash in flight.
	hashFile = hashRegular
)

// hashKnown is hashRegular answered from memory while the file is the one
// that was hashed.
func hashKnown(path string) (string, int64, error) {
	before, statErr := os.Lstat(path)
	if statErr == nil {
		hashMu.Lock()
		m, ok := hashMemo[path]
		hashMu.Unlock()
		if ok && sameIdentity(m.info, before) {
			return m.sha, m.size, nil
		}
	}
	sha, size, err := hashFile(path)
	if err != nil || statErr != nil {
		return sha, size, err
	}
	// Remember it only if the file did not move while it was read.
	if after, e := os.Lstat(path); e == nil && sameIdentity(before, after) {
		hashMu.Lock()
		if len(hashMemo) >= hashMemoCap {
			hashMemo = map[string]hashed{}
		}
		hashMemo[path] = hashed{info: before, sha: sha, size: size}
		hashMu.Unlock()
	}
	return sha, size, err
}

func sameIdentity(a, b os.FileInfo) bool {
	return os.SameFile(a, b) && a.Size() == b.Size() && a.ModTime().Equal(b.ModTime()) && a.Mode() == b.Mode()
}

// warmHashes hashes paths without the service lock, so the checks that
// follow under it answer from memory. Errors are left for those checks.
func warmHashes(paths []string) {
	for _, p := range paths {
		_, _, _ = hashKnown(p)
	}
}
