# Documentation guidelines

Two layers. Do not mix them.

| Layer | Path | Audience | Published |
|---|---|---|---|
| Internal | `docs/` | agents, contributors | git only |
| Public | `www/` (Markdown → VitePress) | users | GitHub Pages |

Internal rules stay in [AGENTS.md](../AGENTS.md) (code and docs in the
same commit, handoff, changelog, **isolated git worktree per agent**).
This file is the **user-facing** contract.

Bars: [Documentation benchmarks](benchmarks.md#documentation-benchmarks)
(Stripe, Diátaxis, VitePress, pi). Do not invent a docs engine.

## UI rules that are enforced by tests

- **Dialogs (ADR-0045).** Every modal imports
  `web/src/components/ResponsiveDialog.jsx` — never `@radix-ui/react-dialog`,
  `@radix-ui/react-alert-dialog` or `vaul` directly. The primitive is a
  centred dialog at `min-width: 720px` and a bottom sheet below; content
  keeps the `dlg dlg-*` classes. Confirmations use its `Alert` set.
  `web/src/lib/dialogPolicy.test.js` fails the build on a raw import; the
  only allowlisted exceptions are desktop-only surfaces (`Palette`,
  `Hotkeys`). Anchored popovers are not modals and stay Radix Popover.

## Public site (`www/`)

- Markdown in `www/`. VitePress builds static HTML (`npm run build` in `www/`).
- Live: `https://cfpperche.github.io/picode/`
- Slash-menu hints open **a new tab** at `/commands#{id}` (`id` = `SLASH[].id`).
  No in-app docs route, no iframe.
- Command copy lives in `www/commands.md` as `## /name {#id}` headings.
- English. Short paragraphs. Tables for TUI vs PiCode.

## Related pi documentation (when applicable)

If a public heading documents something that exists in pi (slash command,
`settings.json`, session JSONL, trust, packages, RPC):

1. Link the **canonical pi doc** in
   [earendil-works/pi](https://github.com/earendil-works/pi)
   (`packages/coding-agent/docs/…`).
2. State **compatibility** in a table: same / PiCode-changed / TUI-only.
3. Do **not** paste pi docs. Summarize the delta; send readers to pi
   for the rest.
4. If parity is broken, say so and link the upstream issue
   (example: `/tree` click is fork until `navigate_tree`,
   [pi#8645](https://github.com/earendil-works/pi/issues/8645)).

A PiCode-only heading skips this section.
