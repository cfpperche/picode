# 2026-09-21 — feat/harden-restarts: owner-grade restarts serialized behind one mutation lock

Shipped: commit c17104ed "install: serialize owner-grade restarts behind one mutation lock", plus
merge of main a30fa0d9; gates green (ci-scoped full, desktop-test, docs, vet). Context: 2026-09-21 ~11:20 local,
an agent ran a forced `PICODE_DEPLOY_FORCE=1 make deploy` concurrently with `make desktop-restart`; the cargo xwin
release build plus two deploys wedged WSL IO — the VM died (boot `b6e59a61`), all 21 tmux sessions lost, the
Windows resident stayed dead until the owner rebooted; the swap job died mid-build, installed exes never modified.

Guards: (1) a mutation lock `/tmp/picode-mutate.lock` serializes `make deploy`, `make desktop-restart`
(whole body, builds included) and direct `picode deploy` — Go-side flock; make passes `PICODE_MUTATION_LOCK_HELD=1` down
so a child never deadlocks on its parent; (2) desktop-swap.sh kills the shell only after probing a live
`wsl.exe … sleep infinity` keepalive process (the scheduler state lies across locales; `SWAP_FORCE=1` is the
deliberate one-off); (3) the swap ends with a bounded 30 s daemon-health verdict; (4) a drift test reads the
Makefile so the Go and make lock paths cannot diverge; (5) AGENTS.md rule: owner-grade restarts are serialized,
one at a time, verified — never `--force` while your own background jobs are in flight.

Verified: internal/install suite (3 new lock tests), go vet, `bash -n`, a full `DRY_RUN=1` swap walkthrough, the
keepalive probe proven both directions on the live machine, and a built CLI proven to WAIT on a held lock (timeout
exit 124, nothing deployed).

## Next up

- VM-death root cause is still unattributed: the updated debt in `docs/handoff/open/wsl-keepalive.md` asks for WSL's
  Operational event log on Windows (also covers the 2026-09-18 terminations).
- The guards take effect only after the owner deploys (deploy is the owner's call, ADR-0105).
- Sibling feat/desktop-ctrl-r (Ctrl+R desktop page reload) is uncommitted, parked to ship via the serialized path.
