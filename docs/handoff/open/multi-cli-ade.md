# Multi-CLI ADE — Pi is one CLI among nine

Decision: ADR-0179. Plan and the 2026-09-22 inventory: `docs/plans/multi-cli-ade.md`.
All five planned branches landed on 2026-09-22; what remains is the debts below.

## Next

## Debts

- [ ] Windows installer without pi: scenario run on the `picode-test` VM (owner) once `feat/runtime-without-pi` lands.
- [ ] GitHub repository description and topics still say "for Pi coding agents" (owner: `gh repo edit cfpperche/picode --description …`).
- [ ] Backup snapshots only `~/.pi`; the other CLIs' sessions and settings are not covered (`docs-site/guide/backup.md` says so since `feat/ade-copy-tail`; covering them is an owner decision).
- [ ] What's new: check whether the 0.1.0 headline "Run real Pi agents from your browser" (`web/shared/data/whats-new.json:8`) renders on a fresh install; reword if it does.
- [x] `docs-site/guide/remote-server.md` (one caveat sentence) and `docs/architecture.md:52` still describe `pi` on PATH as a doctor step — paid by `feat/runtime-without-pi` (`clisStep`, informational).
- [ ] `internal/server/packages_watch.go` scans `~/.pi/agent` and every workspace against npm every 30 min even on a machine with no Pi; harmless (no errors, just registry traffic), but a skip needs the watcher test to stop depending on a missing user dir.
- [ ] Mobile search aliases `Pi settings` / `pi-packages` / `pi-providers` (`web/mobile/src/lib/moreMenuModel.js`) open Pi's panes by design — mobile has no selected-agent context; revisit if mobile gains one.
