# 2026-09-15 — browser-term-ui

ADR-0143 step 4: the rows. This closes the ADR.

## Done

- `BrowserPage.jsx` keys every principal by `keyOf(row)` (`term:<id>` or the
  agent id — the daemon's own namespace), so drafts, flash and save all travel
  with the right principal; the save posts `{term, …}` for a terminal and
  `{agent, …}` for an agent.
- Copy: the section says "agents and terminals", the empty state is "No agents
  or terminals yet / Grants are given per principal…", and a closing line
  states that a `pi` started outside PiCode has no identity and always reads
  the tab on screen.
- Visual read (`var/screenshots/term-rows-section.png`): section, new copy,
  empty state with its one action, no clipping. Green: `make web`, 343 JS
  tests, `ci-scoped`, `close`.
- API verified live on the deployed instance: 7 principals, the six terminals
  by name (canvas, canvas2, codex, desktop, messages, muse agy), each `read`
  by default.

## Not verified

- The **rows-with-terminals** state was not on screen: a scratch is a fresh
  data dir with no terminals, so it renders the empty state. The data path is
  proven (listing + save on the live API); the row rendering for a terminal
  entry is the same `GrantRow` that already ships for agents, but the pixels
  of that state have not been read. First glance on the deployed instance is
  the owner's.
- The endpoint's wire rows still have no test (named in the topic's debts).
