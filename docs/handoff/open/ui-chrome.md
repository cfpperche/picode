# App chrome

## Debts

- Fullscreen: browser chords never reach a CLI; a revealed top strip covers the Canvas pill; the right strip misses an embedded frame; icon-only controls have no hover label.
- App-wide: the toast covers a surface's Close; a grow-resize leaves an idle cursor; Inspector branch chips truncate to `· fe… ·…`; a panel's Run reads as a disabled chip.
- Native surfaces (ADR-0109): `host` has no `openTerminal`; a tab closing under a panel remounts the body with a fresh xterm.
- App surfaces ([parity plan](../../plans/app-surface-parity.md)): gone-entity views answer 500 instead of 404; the blocked card has no screenshot; **Inbox empty lines name no next action — paid 2026-09-21 for the case that has a destination** (`feat/inbox-empty-action`: an empty Active queue with history behind it carries "See the done item(s)"; screenshot + click verified on a scratch, `var/screenshots/inbox-empty-*.png`). Residual, with reason: a mailbox with **nothing at all** keeps the one line and no action — an app reaches the host only through ADR-0109's closed doors, and none of them is an honest "start something" from the Inbox, so inventing a button there would be theatre.
- ~~The CLI sub-tab strip truncates at narrow widths (seen on the providers pane at 600px).~~ **Paid 2026-09-21** (`feat/keyboard-ui`): `.cli-pane-tabs` now carries `min-width: 0` + `overflow-x: auto`, so the nine-tab strip scrolls inside its card instead of running past the card edge and the viewport — which is what `CliPaneTabs`' active-tab `scrollIntoView` was always written for. Measured at 1024px: nav right 967 ≤ card right 992, `scrollWidth > clientWidth`, last tab fully reachable, no page-level overflow. Re-confirmed by a second visual review of the Keyboard pane's 1024px capture.
- **The desktop `.btn-danger` was hover-only.** Every destructive confirm in the desktop app drew Remove and Cancel as the same grey button; the phone app has defined the base state all along, and two scoped desktop rules (`.ask-confirm`, `.dlg-hist`) had re-stated it. Paid 2026-09-21 (`feat/keyboard-ui`) with the base rule in `web/browser/src/styles/app.css`; the blast radius is every desktop `btn-danger` (30 call sites), seen in the `Reset all` confirm.
- **Four defects from the agent-exits visual review (2026-09-23, pixel-sampled on a scratch; the owner asked for them to be fixed during the next tasks).**
  - The removal toast's close × sits on the divider, against Undo's corner — `.notice-x` / `.notice-actions` (`web/browser/src/components/Notice.jsx`, `web/browser/src/styles/app.css` ~4240).
  - In the dark theme a ghost/secondary button's border is nearly invisible (`--border` #232329 on `--bg-elevated` #1c1c23, ~1.1:1), so Cancel beside Remove looks lopsided — `.btn` / `.btn-ghost` (`app.css` ~1539) and the tokens in `web/shared/tokens/theme.css`.
  - The Pi icon renders as a white square in the dark theme (sidebar and tab). Likely cause, not yet measured: `https://pi.dev/favicon.svg`, first in `CLI_FAVICONS` (`web/shared/domain/terminalCli.js`), paints its own opaque ground.
  - On the phone in the light theme, the Remove text is ~4.3:1 (`--danger` #cf3b54 on its own tint), under AA's 4.5:1 — `.btn-danger` (`web/mobile/src/styles/app.css` ~803).

## Notes

- Notices (2026-09-07): the finish card only fires for the agent whose socket is open.
