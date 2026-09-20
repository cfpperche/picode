# 2026-09-20 — agent-terminal-runtime-desktop

## Done

- Desktop Pi interactive panes now resolve `agents.terminalId` to the canonical Agent CLIs terminal record.
- Bound panes attach through terminal identity and use terminal-scoped file/cwd routes.
- Legacy unbound Pi panes retain the agent address until explicit restart.
- Added a Chat/Terminal toolbar visible in both Pi surfaces; removed the duplicate composer Open action.
- Updated CLI terminal architecture docs and added a changelog fragment.
- Added unit coverage for bound and legacy terminal resolution.
- `make ci-scoped` passed after the adversarial review.
- Scratch visual review passed: toolbar readable in Chat and Terminal, overlay audit `ok`, no console errors.

## Next up

- Land this branch with `make land BRANCH=feat/agent-terminal-runtime-desktop`, then run the full `make ci` on `main`.

## Debts

- A live authenticated Pi completion remains unverified on scratch because the fixture had no provider credentials.
- Hosted macOS TUI lifecycle acceptance remains open in `docs/handoff/open/pi-interactive-runtime.md`.
