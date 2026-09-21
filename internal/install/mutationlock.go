package install

// MutationLockPath is the one lock every owner-grade restart serializes on:
// `picode deploy`, `make deploy`, `make desktop-restart`. A contract with the
// Makefile's MUTATION_LOCK — mutationlock_unix_test.go reads the Makefile so
// the two cannot drift apart silently.
const MutationLockPath = "/tmp/picode-mutate.lock"
