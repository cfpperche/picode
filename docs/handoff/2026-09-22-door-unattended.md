# 2026-09-22 — feat/door-unattended

Origin: on 2026-09-22 a live automation pasted into Claude Code's login screen and recorded `done (unverified)`. This branch pays four debts from `docs/handoff/open/multi-cli-ade.md`, which are already flipped there.

Door: `doorDeliverUnattended` in `internal/server/term_prompt.go` covers CLIs with a measured reader. If it does not recognize the composer, it refuses with 409 `unrecognized`. If it cannot read the pane, it refuses with `unobservable`. It types nothing in either case. Automations and the extension use this unattended door. The attended door is unchanged: that covers people at the terminal (the prompt door, Inbox reply and snippets).

Extension: `/api/extension/send` to an agent other than Pi now goes through the unattended door. It sends text only, because screenshots and act are Pi's. It names a closed or missing terminal. Before this it called managed start, and that failed for every guest.

Usage: when `pi --list-models` fails, `catalog.LoadAccounts` falls back to the login set, custom definitions and the vault. `/api/providers/usage` uses it (it returned 503 without pi), and so does the refresh loop. `/api/catalog` still fails without pi, because it is Pi's. The Packages watch skips when `~/.pi/agent` is absent (the `piAgentDirPresent` seam).

Tests: `TestDoorUnattendedRefusesAnUnrecognizedComposer` uses a real tmux pane: unattended gets 409 unrecognized, attended gets 200 unverified. `TestFireMessageToGuestAgent` now expects skipped·unrecognized for a shell pane under a claude-code launch. New: `TestExtensionSendToGuestAgent`, `TestLoadAccountsWithoutPi` and `TestStartPackageUpdatesWatchSkipsWithoutPi`.

Verified: `make ci-scoped` PASS. No UI changed, so there was no visual review.

Not re-run live: the logged-in Claude Code path. It was verified earlier today as `done · Sent to the terminal.`. This change does not touch it, because a recognized composer takes the same branch.
