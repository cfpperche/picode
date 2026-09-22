# Omp OAuth parity — browser flow driven from the GUI

Opened: 2026-09-22 (owner asked when omp gets pi's login UX: provider page
opens, user authorizes, redirect back, subscription stored and used).

Today pi does exactly that from the Providers pane (`POST /api/oauth/start`
opens the authorize page, `/api/oauth/status` polls, credential lands in
`auth.json`). Guests with a login binary get the terminal-guided version
(`codex login`, `grok login`, `opencode auth login`, `hermes auth add`) and
PiCode imports what their store holds (Native readers). Omp is the gap:
- no non-TUI login entrypoint — the browser OAuth lives inside `/login` in
  its TUI, so PiCode can only open a terminal and hint;
- credentials land in `~/.omp/agent/agent.db` (SQLite) that PiCode
  deliberately does not read, so the roster cannot show or import the
  resulting subscription (note in `clicreds`).

## Paths to close it

- [ ] Upstream omp: publish a non-TUI login entrypoint (`omp login
      <provider>` or equivalent over the auth-broker). PiCode side is then a
      per-provider `Login` declaration in `internal/clicreds` — the same
      pattern as `codex login`. Blocked on upstream, not on PiCode.
- [ ] PiCode reads `agent.db` (a Native reader like codex/grok's) so native
      `/login` results import + display. ADR required first: security model
      + persistence (reading another tool's credential store; schema is
      omp-internal and versioned).
- Refused path: PiCode performing the vendor OAuth itself (ADR-0168).
