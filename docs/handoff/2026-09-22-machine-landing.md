# 2026-09-22 — machine-landing: Preferences → Landing work

Shipped: **Preferences** tab `#/preferences/landing` (components/LandingWork.jsx): the machine layer of the ADR-0182 integration declaration (fast-forward, up to 8 checks, help that machine checks run for every project) and a "Who follows these rules" list (followsRows in `web/shared/domain/workspaceSettings.js`, decision table tested) with Edit opening the workspace's Settings dialog via a window event (focus returns to the Edit button). New read GET `/api/delivery/integrations` (machine + workspaces map), `store.ListIntegrationSettings`. Rules editor extracted to components/LandingRulesFields.jsx, shared with WorkspaceSettings.

Found in review and fixed: a feed refresh swapped the base version under an edited draft, so a save racing another write overwrote it without 409 — the page now keeps the base layer its draft came from; an untouched form follows the feed. Mono font ligatures rendered `--check` as a dash in command inputs → `font-variant-ligatures: none`.

Verified: `make ci-scoped` PASS; visual-review PASS on scratch after two rounds (desktop, 390px, dark; conflict, feed-follow, cold load, focus).

Blind spots / pre-existing, not fixed: the sidebar shows "No workspaces yet." for ~2.7s on cold load before the fleet arrives; the Preferences pane draws a top divider on every tab but Appearance; dark-mode btn-primary contrast ~3:1. Mobile app has no Landing work tab. No ADR (read-only list route beside ADR-0182's doors).
