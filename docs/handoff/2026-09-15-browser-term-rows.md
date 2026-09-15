# 2026-09-15 — browser-term-rows

ADR-0143 step 3: the grants API knows terminals. Step 4 (the rows) is open.

## Done

- `GET /api/browser/policies` appends one entry per terminal from the store
  (`kind: "terminal"`, `termId`, name, workspace) with its effective grant —
  `browser.Resolve(TerminalPrefix+id)` — and the same `saved` flag agents use.
- `POST /api/browser/policy` takes `term` beside `agent`: exactly one of them,
  the terminal must exist (`GetTerminal`), and the key becomes `term:<id>`.
  Sending both is a 400; an unknown terminal is a 404.
- Agent entries now carry `kind: "agent"` (the UI can tell the two apart).
- Green: `go build ./...`, the resolver's six decision rows.

## Not built yet

- **The UI rows** (step 4): the section still renders agents only, so the
  visible half is open. Needs the kind field, a terminal row that posts
  `{term, tier, domains}`, and the line that a `pi` started outside PiCode
  stays read-only with no row at all.
- The endpoint's wire rows have no test (the resolver's do) — named in the
  topic's debts, to land with the UI slice.
- Applied in the wrong tree once (the root checkout) and moved into the
  worktree before any commit; the root is clean (verified).
