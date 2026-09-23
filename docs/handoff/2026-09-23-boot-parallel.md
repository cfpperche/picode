# 2026-09-23 — boot-parallel: the desktop boot no longer waits for Pi

Shipped: `web/browser/src/lib/bootReads.js` starts the boot's reads at once; the boot awaits only the fleet
(workspaces, free agents, terminals, apps — old failure rules kept, tested) and applies system, CLIs and the Pi
model catalog when they land. The catalog (`pi --list-models`, ~2.2 s per call) is asked after the fleet settles,
so it does not hold an HTTP/1.1 connection in front of the fast reads. One duplicate `/api/agents?free=1` read
removed. The Apps grid waits for the boot's `/api/apps` before saying "No apps yet." (it flashed over 4 apps).
Verified: `make ci-scoped` PASS; measured on scratch with a document-start recorder: skeleton 60–170 ms (was
2.2–4.4 s), fleet reads start within 1 ms of each other, catalog after them, tab restore and "gone" states intact,
no console errors.
Open: on plain HTTP/1.1, rapid reloads can still stall a page load for seconds (once 47 s) — aborted loads'
catalog requests keep connections busy; not measured over the production HTTPS/HTTP-2 listener. A server-side
catalog cache (`internal/catalog.Load` runs Pi on every call) would remove the long request altogether.

## Next up

- Cache `/api/catalog` server-side (invalidate on provider/model changes) — the 2.2 s Pi call is per request
