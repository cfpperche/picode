### Added
- **Terminals are listed as principals in the browser grants API** (ADR-0143):
  `GET /api/browser/policies` returns each terminal with its effective grant,
  and `POST /api/browser/policy` accepts `term` beside `agent` — the key
  carries the namespace (`term:<id>`), so a terminal can never widen an
  agent's grant or the other way round. The settings rows that show them are
  the next slice.
