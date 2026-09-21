# 2026-09-21 — feat/packages-mut: the guest mutations answer from the driver (ADR-0176 slice 2b)
Shipped: `/api/cli-packages/{install,remove,update,toggle,marketplace,inspect}` answer from `pkgs.DriverFor(cli)`. The model gained the mutation half — `Target`, `MarketRequest`, `Command` (the vendor argv, the folder it runs in and the exact line a pane copies) and `Driver.Install/Remove/Update/Marketplace/Toggle/Inspect`. The verbs split by transport: install, remove, update and a marketplace fetch *build* the command the job lane runs (`internal/clijob` untouched — 202 + job id + the same payload, so the same job id and the same stored argv); toggle, inspect and a marketplace removal *run* and answer the command they ran. `internal/clipkgs` untouched: argv builders, rendered lines, the OpenCode file write and every refusal stay the engine's, so a verb's gate is its own declaration. Pi's verbs refuse with `ErrNoMutation`; no route reaches them (both families refuse Pi before a verb). `pkgs.GuestViewOf` now has one caller: the byte-equality test that compares the driver's mapping against the engine's.
Verified: `make ci-scoped` PASS (fmt,vet,hooks,go[5]; 6 path(s) vs main), then `make close` green after merging main. Byte equality at the mapper layer as a permanent test (`internal/pkgs/mutate_test.go`: argv+dir+line per verb, the removal's remaining sources, the inspection text) and over the real routes by a scratch test (deleted) that compared every mutation's raw body, status and command against copies of the handlers HEAD shipped — mutation-checked first, which caught a real drift (`inspect` refused with "opencode inspect" instead of the engine's "opencode inspects nothing"). `internal/clipkgs` and every `internal/server` test green unchanged; `git diff --stat main...HEAD -- web/` empty. Not run: the live harness (`PICODE_PKGS_LIVE=1` installs vendor plugins on this machine). `go test ./internal/...` shows one contention flake (`internal/llama`), green alone.
visual-review: n/a
Merge: fast-forward ready.

## Next up

- Slice 3: the Pi driver's own mutations onto these verbs, then one pane whose controls come from `Caps`.

## Debts

- OpenCode's removal is unreachable from the pane: its declaration says
  `Remove: true` (the write is clipkgs' own), while the route sends every
  removal to the job lane, which runs an argv. Owned by
  `docs/handoff/open/packages.md`.
