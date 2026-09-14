# 2026-09-14 — docs-getting-started: first run is a release, not make build

Shipped: `docs-site/guide/getting-started.md` is download + `picode install`
+ open the app (LibreChat bar 7). `make build` / Go / Node live on
`/guide/from-source`. README Quick start matches. Sidebar Start lists
From source after Getting started.

Verified: `make docs` + Vale (37 files); browser at
`http://127.0.0.1:4174/picode/guide/getting-started` — tip callout, three
steps, pager to From source. Screenshot
`var/screenshots/docs-getting-started.png` (not committed).

visual-review: n/a (docs site).

Not done: feature-page rhythm; config overview; MCP cookbook
(`docs/handoff/open/docs-site.md`). No Docker first-run — PiCode does not
ship a compose file for itself.

Merge: fast-forward ready.

## Next up

- Existing guides take the feature-page rhythm.
