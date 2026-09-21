# 2026-09-20 — feat/inspector-drawer: agent-screen header toggle opens Inspector as right drawer
Shipped: owner feedback rejected the glance line. Agent screen header now carries a PanelRight toggle (the desktop inspector toggle's analog) opening the Inspector as a right drawer (Vaul direction="right" via MobileSheet's new side prop) over the conversation; Back in the drawer header or the overlay returns to the agent — nothing under it remounts. Glance component + CSS deleted; ProjectToolsSheet lost its Inspect row (the button replaces it); App closes the drawer on any navigation away. Two bugs fixed in QA: agentTouched's getSnapshot re-parsed JSON per read, looping the commit (React #185) — store now returns stable array identity (test pins it); side prop reached Sheet.Root but not Sheet.Content so the drawer rendered unstyled — caught by DOM probe.
Verified: make ci-scoped PASS; suites green; visual-review PASS 5/5 (drawer-open, drawer-actions, agent-header). Blind spot: physical-device swipe/drag feel of the right drawer untested (headless QA only).
visual-review: PASS
Merge: fast-forward ready.

## Debts

- Right-drawer swipe/drag feel unverified on a physical device — headless QA only
