# Multi-CLI ADE — Pi is one CLI among nine

Decision: ADR-0179. Plan and the 2026-09-22 inventory: `docs/plans/multi-cli-ade.md`.
One branch per session, in this order; each carries its decision table and tests.

## Next

- `feat/system-clis`: `/api/system` drops the fixed `pi` block and the npm warning; the System page lists tmux as the only requirement plus one row per agent CLI from `/api/clis`; automations gate on the target agent's CLI (`fireInput.TargetIsPi`, two new decision rows; fixture `scripts/qa-mobile-workflows-v2.mjs:84,284`). Plan: docs/plans/multi-cli-ade.md §2.
- `feat/runtime-without-pi`: `piStep` becomes the informational `clisStep`; the Windows installer's runtime drops `pi` and pins Node 22 by constant; `internal/provision/container.go` stops npm-installing pi into member containers. Plan: docs/plans/multi-cli-ade.md §3.
- `feat/free-agent-cli`: `POST /api/agents` takes `cli`; the sidebar **New agent** opens the CLI picker (`NewCliPrincipal` in free mode); `catalogForAgent` stops forcing Pi as installed. Plan: docs/plans/multi-cli-ade.md §4.
- `feat/ade-copy-tail`: the configure-cluster docs, the "pi correlation" writing rule generalized to every vendor, UI labels and jargon, dead `errAgentCmdMissing`. Plan: docs/plans/multi-cli-ade.md §5.

## Debts

- [ ] Windows installer without pi: scenario run on the `picode-test` VM (owner) once `feat/runtime-without-pi` lands.
- [ ] GitHub repository description and topics still say "for Pi coding agents" (owner: `gh repo edit cfpperche/picode --description …`).
- [ ] Backup snapshots only `~/.pi`; the other CLIs' sessions and settings are not covered and `docs-site/guide/backup.md:23,39` does not say so.
- [ ] What's new: check whether the 0.1.0 headline "Run real Pi agents from your browser" (`web/shared/data/whats-new.json:8`) renders on a fresh install; reword if it does.
- [ ] `docs-site/guide/remote-server.md` and `docs/architecture.md:52` still describe `pi` on PATH as a doctor step — true until `feat/runtime-without-pi`; drop the caveat sentence there when it lands.
