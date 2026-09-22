# 2026-09-22 — feat/guest-automations

Origin: an adversarial review of the ADR-0179 work found that message automations never reached a guest (non-Pi) agent. `mode()` resolves a guest's launch-terminal session, so a live guest was skipped as "agent in terminal" and a stopped one got the door's 409 "closed". The feat/system-clis note had reported `skipped · closed` as the door's receipt. That result was the bug, and its changelog line is corrected here.

Server: the `decideFire` guest branch now sends an interactive guest ahead to the door (doorRun), and skips a stopped one with `terminal closed`, a new reason with its own Inbox body. Rows renamed or added in `automations_test.go`. `TestFireMessageToGuestAgent` drives `Fire` against a real tmux session: closed gives `terminal closed`, open reaches the door.

Web: `ownerOfTerminal` in `web/shared/domain/tree.js` (tested). App routes the user menu and the palette through the agent that owns the selected terminal tab. Undo of a removed free agent posts cli/provider/model/thinking.

Docs corrected: `docs/changelog.d/system-clis.md`, the ADR-0179 table and consequences, `docs/architecture/automations.md`, `docs/plans/multi-cli-ade.md`, `docs-site/guide/automations.md` (new "Skipped · terminal closed" row).

Verified: `make ci-scoped` PASS. Live on a scratch instance with a Claude Code agent: terminal closed gave `skipped · terminal closed`, and terminal open gave `done · Sent to the terminal (unverified)`.
Limit: the scratch had no Claude login, so the paste landed on Claude Code's login screen. The door still reported done because it maps `unverified` to done. This is recorded as a debt in `docs/handoff/open/multi-cli-ade.md`. Full end-to-end verification needs a logged-in CLI, which is the owner's to run.

No visual review: the change is routing and data only, so no UI renders differently.
