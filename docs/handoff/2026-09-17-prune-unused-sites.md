# 2026-09-17 — prune-unused-sites

v2a from the browser scope: forgetting the permission entries for sites the
human no longer uses.

## The rule (in the code and in a test)

`PruneBrowserPermissions(since)` forgets a site entry when the history has no
visit for its host inside the window. One row per condition:

| the entry | the site's visits | result |
|---|---|---|
| the every-site policy (`*`) | anything | kept — it is policy, not a site |
| a site, visited since | inside the window | kept |
| a site, last visit before | only older | forgotten |
| a site, never visited | none | forgotten |
| nothing matches | — | nothing forgotten, **no event** |

One mutation, one event (ADR-0048, added to the invariant test). The answer
carries what was forgotten so the page can also clear the **live shell's**
copy — a standing that survived only there would come back on the next report,
the same rule the single Reset follows.

`POST /api/browser/permissions/prune` — `days` defaults to 90 and floors at 7:
"forget everything I saw today" is not a cleanup, and its accidental version
is unrecoverable.

## Verified

- Store table test: the four rows plus a second prune answering nothing.
- Live on a scratch: seeded `*` + a visited site + a never-visited one → prune
  → `removed: 1`, `forgotten: [esquecido.test]`, the other two left; `days: 1`
  → applied 7; no body → 90.
- Visual: "Forget unused" in the Recent decisions header, rows behind it.
- Green: store, server, 383 web tests, `ci`.

## Noted

A wrong-base-URL slip put three permission rows and a visit into another
scratch (port 8471) before I caught it; removed by id, that instance is back
to `{"permissions":[]}`. The lesson is the one this repo keeps teaching:
read the port the script printed, not the one from the last run.
