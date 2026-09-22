# 2026-09-22 — feat/system-clis

Shipped: the first branch of `docs/handoff/open/multi-cli-ade.md`. Server: `/api/system` drops the fixed `pi` block and the npm warning (`internal/server/system.go`), the Pi update card and `POST /api/system/pi-update` are gone with `internal/pipkg/pi_update.go`, and `decideFire` gains `fireInput.TargetIsPi` with two new decision rows — a `message` to a guest agent never needs Pi, a `message` to a stopped Pi agent still does (`internal/server/automations_run.go`, rows in `automations_test.go`). Web, both twins: the System page lists tmux as the only requirement, mkcert and tailscale as optional, and an Agent CLIs section built from `/api/clis` with one row per CLI linking to its page; the Automations blocked banner is scoped by the shared rule in `web/shared/domain/automationsPi.js` (tests in `automationsPi.test.js`). `docs/architecture/agent-manager.md` and `automations.md` describe the gate; the QA fixture `scripts/qa-mobile-workflows-v2.mjs` stubs `/api/clis`.

Verified: `make ci-scoped` PASS — fmt, vet, hooks, go, test-js, build, docs. Live on a scratch instance with `pi` off the PATH: `/api/system` has no `pi` key and no warning, `/api/clis` reports `pi: false`; "Run now" of a message automation to a Claude Code agent recorded `skipped · closed` (the door receipt), a start automation recorded `failed · pi missing`.

Visual review: three passes on desktop 1440×900 and mobile 390×844 captures under `var/screenshots/system-clis/` — System with and without pi, top and bottom; Automations guest-only without the banner, blocked with it. Pass 2 failed on the "Install Pi" ghost button: its outline sat about 10 RGB units from the amber banner; fixed with a visible amber border and tint in both apps. Pass 3 PASS, `window.__picodeOverlayAudit()` ok, card 5/5.
visual-review: PASS (v3-automations-*-nopi-blocked.png + system captures; overlayAudit ok; card 5/5)

Known: CLI versions on System read "installed" until the Agent CLIs page has run a check for that CLI — the lifecycle diagnostic is what carries the version. Public captures are stale for the web inputs; `make deploy` recaptures them.

The remaining branches, their order and the debts live in `docs/handoff/open/multi-cli-ade.md`; nothing is repeated here.
