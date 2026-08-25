# Documentation guidelines

Two layers. Do not mix them.

| Layer | Path | Audience | Published |
|---|---|---|---|
| Internal | `docs/` | agents, contributors | git only |
| Public | `www/` | users | GitHub Pages |

Internal rules stay in [AGENTS.md](../AGENTS.md) (code and docs in the
same commit, handoff, changelog). This file is the **user-facing** contract.

## Public site (`www/`)

- Source of truth for slash/command help. **Never** duplicate that copy
  into the React bundle (`commandDocs.js` is a URL helper only).
- GitHub Pages: `https://cfpperche.github.io/picode/`
- The app `#/docs/{cmd}` **iframes** `…/commands/{cmd}/`.
- Missing page → public `404.html`. Do not invent in-app prose.
- One command per folder: `www/commands/{id}/index.html` (`id` matches
  `SLASH[].id` in `web/src/lib/slash.js`).
- English. Short paragraphs. Tables for TUI vs PiCode.

## Related pi documentation (when applicable)

If a public PiCode page documents something that exists in pi (slash
command, `settings.json`, session JSONL, trust, packages, RPC):

1. Link the **canonical pi doc** in
   [earendil-works/pi](https://github.com/earendil-works/pi)
   (`packages/coding-agent/docs/…`).
2. State **compatibility** in a table: same / PiCode-changed / TUI-only.
3. Do **not** paste pi docs. Summarize the delta; send readers to pi
   for the rest.
4. If parity is broken, say so and link the upstream issue
   (example: `/tree` click is fork until `navigate_tree`,
   [pi#8645](https://github.com/earendil-works/pi/issues/8645)).

A page with no pi counterpart (PiCode-only chrome) skips this section.

## In-app iframe

- `#/docs/{cmd}` loads the public URL in an iframe. No second copy.
- “Open in browser” may sit next to Back; the public URL is canonical.
- Do not iframe `github.com` (X-Frame-Options). Iframe **our** Pages;
  pi links on those pages open in a new tab.
