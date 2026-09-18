# Connectors parity (ADR-0150) — durable debts

## Debts

- [x] Guest codecs are strict: OpenCode itself writes trailing commas (JSONC-ish) into opencode.json and our parse error blanks the whole pane. Fix: tolerate/normalize trailing commas where the vendor does, and degrade malformed user files to a blocked state naming the file (one line + action) instead of a pane-wide error. Repro 2026-09-18: owner real file had `,}` after schema-only config.
  Paid 2026-09-18 (feat/connectors-codec-robustness): lenient JSONC reads (strict writes), per-layer `error` degradation on GET /api/mcp for every guest codec (JSON, TOML, YAML), pane blocked line + Open/Retry, `/api/mcp/reveal`.
