# 2026-09-14 — docs-sidebar: Start / Use / Run / Configure / Reference

Shipped: VitePress sidebar + top nav follow the LibreChat audience split
(`docs/benchmarks/2026-09-14-librechat-docs.md`). Guides is gone. Settings
and Providers moved to Configure. `/guide/` is the Use catalog (tables).
`scripts/docs-llms.mjs` CURATED matches. Fragment
`docs/changelog.d/docs-sidebar.md`.

Verified: `make docs` + Vale (36 files); browser on
`http://127.0.0.1:4173/picode/` — nav Start/Use/Commands, catalog tables
link through (Agent CLIs, Settings). Screenshot
`var/screenshots/docs-sidebar-use.png` (not committed).

visual-review: n/a (docs site, not app chrome).

Not done: getting-started still leads with Go/Node/`make build`; feature
rhythm; config overview; MCP cookbook (`docs/handoff/open/docs-site.md`).

Merge: fast-forward ready.

## Next up

- Getting started is the user first-run; from-source demoted.
