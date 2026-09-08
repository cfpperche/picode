# Documentation guidelines

Two layers. Do not mix them.

| Layer | Path | Audience | Published |
|---|---|---|---|
| Internal | `docs/` | agents, contributors | git only |
| Public | `docs-site/` (Markdown → VitePress) | users | GitHub Pages |

Internal rules stay in [AGENTS.md](../AGENTS.md) (code and docs in the
same commit, handoff, changelog, **isolated git worktree per agent**).
This file is the **user-facing** contract.

Bars: [Documentation benchmarks](benchmarks.md#documentation-benchmarks)
(Stripe, Diátaxis, VitePress, pi). Do not invent a docs engine.

## UI rules that are enforced by tests

- **Dialogs (ADRs 0046/0072).** Desktop modals import
  `web/desktop/src/components/ResponsiveDialog.jsx`: centered at ≥720px,
  bottom sheets below. Mobile modals import
  `web/mobile/src/components/MobileSheet.jsx`: sheets at every width.
  Both retain `dlg dlg-*` classes and the `Alert` confirmation API.
  `web/tools/dialog-policy.test.mjs` rejects raw Radix dialog/alert or Vaul
  imports outside these primitives and the desktop Palette/Hotkeys exceptions.
  Anchored popovers stay Radix Popover. A sheet focuses a field only after
  the user taps or types; do not bypass the primitive's focus handling.

## Public site (`docs-site/`)

- Markdown in `docs-site/`. VitePress builds static HTML (`make docs`).
- Live: `https://cfpperche.github.io/picode/`
- Slash-menu hints open **a new tab** at `/commands#{id}` (`id` = `SLASH[].id`).
  No in-app docs route, no iframe.
- Command copy lives in `docs-site/commands.md` as `## /name {#id}` headings.
- English. Short paragraphs. Tables for TUI vs PiCode.
- Example local URLs (`https://localhost:8445`) are **inline code**, not
  markdown links. VitePress treats a bare `https://…` as a crawlable
  link and fails the build — that froze GitHub Pages from 2026-08-29.
  `ignoreDeadLinks` in `docs-site/.vitepress/config.mjs` also skips localhost
  and 127.0.0.1. `make docs` is in `make ci`; do not rely on the Pages
  workflow as the first gate.

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
