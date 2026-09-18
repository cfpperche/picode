# 2026-09-18 — feat/connectors-codec-robustness

Paid the connectors-parity codec debt (ADR-0150 read vs write bars).

- Guest codecs read leniently, write strictly: strict JSON first, then a
  JSONC second chance (trailing commas before closers, `//` and `/* */`
  comments; strings copied verbatim). Every save still lands strict JSON.
- A config file that exists but parses under no form blocks only its own
  layer: `Layer.Error` (JSON, TOML, YAML alike), no servers from it, healthy
  layers keep listing — `GET /api/mcp` answers 200.
- Pane shows one blocked line naming the file with **Open** (guest;
  `POST /api/mcp/reveal`, path re-derived server-side) and **Retry**;
  browser + mobile share `blockedLayers()`.
- Tests: codec table, per-driver JSONC tolerance + strict rewrite,
  per-codec degradation, blocked-user/healthy-project split, HTTP degrade +
  reveal (client path refused). Fixtures root HOME/XDG in t.TempDir only.

Gates: go build, go test ./internal/... -count=1, web npm test (820),
make web, ci-scoped PASS.

visual-review: PASS (var/screenshots/codeccr-blocked.png read; owner repro
listed clean, malformed showed the one-line blocked row; overlayAudit ok,
rows 36px aligned; card 5/5).

Decision table (conditions → action) covered by tests:
- file missing → empty layer, no error
- strict JSON → parsed
- JSONC-ish (commas/comments) → parsed leniently, 200, no blocked row
- malformed → 200, layer exists+reason, no servers, other layers list
- write on malformed → refused (unchanged)
