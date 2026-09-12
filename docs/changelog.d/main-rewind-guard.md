### Added
- Git guard: `main` can no longer be rewound — the reference-transaction
  hook refuses any move of `main` that is not a fast-forward, including a
  stale-ref fast-forward onto a divergent tip that silently erased merged
  work once. A deliberate rollback requires `PICODE_ALLOW_MAIN_REWIND=1`.
  The guard's functional tests live in the hooks selftest.
