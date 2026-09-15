# 2026-09-14 — grants-editor

Slice 4 increment 4.3: the grants editor — the last open piece of slice 4.

## Done

- Go: `GET /api/browser/policies` (every agent + its effective grant,
  `saved` flag) and `POST /api/browser/policy` (existence check, tier via
  `browser.Save`, domain normalization: trim/lowercase/drop empties).
  Table-tested: defaults, round trip, bad tier 400, unknown agent 404.
- UI: BrowserPage gained the grants card — skeleton on first load, empty
  state (one line + Create an agent), per-agent rows (name, tier select,
  domains line visible only for Act/Full, Zod-gated Save with inline
  error), `custom`/`default` tags, feed-driven refetch (no polling).
- Zod: `browserGrantSchema` strips scheme/path/port from pasted entries.
- Visual review on the scratch (:8473 seeded, :8474 empty): light + dark,
  error state, save round trip proven through the API, state survives
  reload, overlay audit ok. Screenshots in var/screenshots/grants-*.png.

## Notes

- The domain rule matches bare hosts; a saved `https://x.com` entry would
  silently never match, which is why the editor normalizes instead of
  trusting the field.
