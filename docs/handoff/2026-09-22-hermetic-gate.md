# 2026-09-22 — feat/hermetic-gate: the gate that would have caught the three, and the race it found

`ci-scoped`'s Go stage now runs the scoped packages with `PATH=<toolchain>:/usr/bin:/bin`. No agent CLI exists there, so a test that reaches a vendor-locating path without stubbing fails while iterating — the three that landed on 2026-09-22 (`opencode`, `omp`, `pi`) would have been caught before the push instead of on a runner hours later.
Measured before adopting it: the whole suite under that PATH failed exactly one test, and it was *not* a CLI dependency — `TestTerminalRefusesAShellThatExitsAtOnce`, the guard against a phantom terminal.
That test was right and the product was wrong: a pane that dies at once takes its session with it while tmux still answers "alive" for a moment, so `ensureShell` could keep a terminal that never lived, every later call on it answering "no server running". It now asks `HasSession` a second time after a 50 ms beat; measured under the gate's PATH, 5 ms and 20 ms still said alive and 50 ms told the truth five runs out of five. Fifty milliseconds on a creation is invisible; a phantom terminal is not.
Verified: `make ci-scoped` PASS (full) — 78 packages, `internal/server` in 4 shards at 104.7 s, the whole Go stage under the CLI-free PATH.
Fragment: `docs/changelog.d/hermetic-gate.md` (`### Fixed` — the phantom terminal is user-visible). The debt this pays is `docs/handoff/open/process.md`.
Blind spot: the beat was measured on this machine; a slower reaper would want more, and the number is in the comment beside the call.
