
## Amendment 2026-09-14 — the dashboard's attention line

**The door.** The dashboard (`DashboardView`) may carry at most one line above
its KPI row that says what is blocked on the reader, and its single action may
name the Inbox app: the count is the `Badge.Count` the tile already draws from
`GET /api/apps`, and the action opens the app's own route (`#/app/inbox`
through the host's `openApp`). The host supplies the line, the words, the
placement and the button; the app supplies a number and a route it already
has. Nothing imports and no app module reaches the shell, so
`web/tools/app-boundary.test.mjs` passes unchanged.

**Why it is a door and not a leak.** The direction rule above settles it — the
host names an app — and the form is the one this ADR asks for: **the phone
already does exactly this**, its home queue folding Inbox rows into
`needsYou` under ADR-0044. The desktop had no equivalent, so the only place a
desktop reader could learn that a question is waiting was the sidebar tab's
badge. The line does not render app content, does not read an app preference
and does not add a second source of truth: it is the badge, in words, with a
way to act on it.

**It is not the app's chrome.** The line counts two things, only one of which
is the app's: questions in the Inbox, and agent-CLI terminals whose own hooks
reported `needs-you` (ADR-0062) — that half is the fleet's and stays if the
Inbox is uninstalled. The selected agent's `waiting` is deliberately not
added, because an agent asks through the Inbox: counting both would report one
block twice.

**Conditions.** The line exists only while something is blocked (no zero-state
chrome), it offers exactly one action, and the action goes to the app's route
rather than to a surface the app does not own.

**If wrong**: delete the two props (`inboxWaiting`, `onOpenApp`) from
`App.jsx`'s `<DashboardView>` call and the line goes with them, leaving the
fleet half's data untouched.
