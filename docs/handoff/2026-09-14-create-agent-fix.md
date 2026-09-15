# 2026-09-14 — create-agent-fix

Owner report: clicking Create an agent in the grants empty state only
closed the settings page.

## Done

- Root cause: `go("agents")` falls through routes.js to the default
  `#/` — the Agents tab is a side tab, not a route. The button now
  selects the Agents side tab, goes home and opens the New agent form
  (`setFormKind("free") + setShowForm(true)`), verified live on the
  scratch (form.create-form visible, overlay audit ok).
- Empty-state copy now distinguishes managed agents from sidebar
  sessions (the production agents table is empty — the owner's items
  are terminal sessions, which resolve to the default read policy).
