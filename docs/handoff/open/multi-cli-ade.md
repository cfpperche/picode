# Multi-CLI ADE — Pi is one CLI among nine

Decision: ADR-0179. Plan and the 2026-09-22 inventory: `docs/plans/multi-cli-ade.md`.
All five planned branches landed on 2026-09-22; what remains is the debts below.

## Next

## Debts

- [ ] Windows installer without pi: scenario run on the `picode-test` VM (owner); `feat/runtime-without-pi` has landed.
- [x] GitHub repository description and topics still say "for Pi coding agents" — paid 2026-09-22: description names the nine CLIs; topics gain claude-code, codex, opencode (owner-approved `gh repo edit`).
- [x] Backup snapshots only `~/.pi`; the other CLIs' sessions and settings are not covered (`docs-site/guide/backup.md` says so since `feat/ade-copy-tail`; covering them is an owner decision). — paid by `feat/backup-all-clis`: settings always, login files with secrets, file-based sessions with sessions, for every CLI (ADR-0014 amendment); SQLite session stores stay out.
- [ ] What's new: check whether the 0.1.0 headline "Run real Pi agents from your browser" (`web/shared/data/whats-new.json:8`) renders on a fresh install; reword if it does.
- [x] `docs-site/guide/remote-server.md` (one caveat sentence) and `docs/architecture.md:52` still describe `pi` on PATH as a doctor step — paid by `feat/runtime-without-pi` (`clisStep`, informational).
- [x] `internal/server/packages_watch.go` scans `~/.pi/agent` and every workspace against npm every 30 min even on a machine with no Pi; harmless (no errors, just registry traffic), but a skip needs the watcher test to stop depending on a missing user dir. — paid by `feat/door-unattended`: the watcher skips when `~/.pi/agent` does not exist (`piAgentDirPresent` seam, tested).
- [ ] Mobile search aliases `Pi settings` / `pi-packages` / `pi-providers` (`web/mobile/src/lib/moreMenuModel.js`) open Pi's panes by design — mobile has no selected-agent context; revisit if mobile gains one.
- [x] The prompt door maps an `unverified` paste to a **done** run (`mapDoorOutcome`, `internal/server/automations_run.go`): on 2026-09-22 an automation pasted into Claude Code's login screen on a scratch instance and recorded `done · Sent to the terminal (unverified)`. A run should not say done when PiCode never saw the CLI take the prompt; the logged-in path is verified (2026-09-22, scratch with a copied Claude Code login: `done · Sent to the terminal.`, the prompt submitted in Claude Code's composer), so what remains is only the `unverified` → done mapping. — paid by `feat/door-unattended`: automations and the extension use `doorDeliverUnattended`, which refuses an unrecognized composer (`skipped · unrecognized`); a CLI without a reader still gets the paste and says "(unverified)".
- [x] Shared-server member containers (`internal/provision/container.go`) get the distro's nodejs/npm (12 on Ubuntu 22.04, 18 on 24.04) with a root-owned prefix: Agent CLIs' Install fails there (EACCES, and Pi needs node >=22.19). The desktop installer's fix (NodeSource 22 + `~/.local` prefix, `feat/npm-user-prefix`) is not applied to the container yet. — paid by `feat/backup-all-clis`: the member root gets NodeSource Node 22 and a container-only npmrc `prefix=${HOME}/.local`; older roots converge on the next provision.
- [x] The Chrome extension lists every agent (`/api/extension/agents`) but sends through managed start (`/api/extension/send` → `startManaged`), so a guest (non-Pi) agent can be picked and never receives the tab. — paid by `feat/door-unattended`: a guest agent gets the tab through its terminal (text only; screenshots and act stay Pi's).
- [x] The providers usage summary (`GET /api/providers/usage`, `internal/server/providers_usage.go`) builds its targets from Pi's catalog and answers 503 without pi; check whether guest rosters lose usage on a machine without Pi. — paid by `feat/door-unattended`: `catalog.LoadAccounts` builds the account list from the vault when pi is absent; the summary and the refresh loop use it.
- [ ] Dead code from the old Pi-only free-agent form: the `kind === "free"` branches of both `CreateForm.jsx`, `createSubmit.js` (both apps) and `createFreeAgentSchema`, which `schemas.test.js` still uses to test the model pick.
