# 2026-09-08 — feat/cli-layout: stable Agent CLIs page geometry
Shipped: app-owned `AgentClisFrame` shared by CLIs, Terminals, Sessions, Settings and Packages.
Desktop uses one 1240px maximum, an unpadded card and a stable scrollbar gutter;
mobile keeps full page width. Narrow Sessions and CLI setup actions wrap.
No API, native CLI settings or package behavior changed; no dependency added.
Verified: `make ci-scoped` PASS (fmt, vet, hooks, frontend tests, both app builds and embedded binary).
Scratch browser QA: desktop 1920/1365/740/560 and mobile 390/320, dark/light;
48 tab/config/blocked/error states and 54 overlay/control audits PASS.
At 1920px, switching to Packages changed card width by 160px before, 0px after.
Actual tab navigation, narrow horizontal tab scrolling and package picker exercised.
Screenshots read, including empty/blocked/error states and four refreshed public captures.
visual-review: PASS — contained overlays, readable text, usable controls,
no clipping/double scroll/dead hover, clear next action (5/5).
Evidence: `var/screenshots/cli-layout/` (ignored; retained in the primary checkout).
Not done / debts: physical-device acceptance remains external; deployment is a separate batch.
Merge: fast-forward ready; full `make ci` runs on main at integration.
