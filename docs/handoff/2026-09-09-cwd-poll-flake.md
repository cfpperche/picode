# 2026-09-09 — the live-cwd poll was too tight (`feat/cwd-poll-flake`)

**Shipped.** `TestTerminalLiveCwd` waits up to 20 s (was 2 s) for the pane's
shell to run the `cd` it was sent. The loop still leaves the moment tmux
reports the new path, so a healthy run is as fast as before.

**Why.** `make ci` on `main` failed here on 2026-09-09 while a second full
suite ran on the same machine: the shell was late, not wrong
(`live cwd=".../002" want ".../003"`). The test passed 3/3 in isolation
immediately afterwards, and `make ci` was green on a quiet machine.

**Verified.** The test alone, then `make close`.

**Debts.** Three sibling polls keep a 1–2 s ceiling
(`automations_test.go:269`, `mcp_test.go:302`, `peer_attention_test.go:92`).
They wait on this daemon's own goroutines rather than on a foreign process,
so they were left alone; if one of them ever flakes under load, it wants the
same treatment.

**Merge.** `git merge --ff-only feat/cwd-poll-flake && make ci`.
