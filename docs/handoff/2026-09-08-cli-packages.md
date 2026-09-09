# 2026-09-08 — cli-packages: native packages under Agent CLIs
Shipped: ADR-0102; independent desktop/mobile Packages views with Pi capability selector.
Canonical: `#/clis/packages/pi[/config/<package>]`; workspaceId/agentId/scope survive reloads and legacy redirects.
Preserved: native Pi APIs, installation commands, files and agent overrides; terminal inventory is independent.
Recovery: missing/mismatched targets block; transient failures retain drafts; changed file locations require confirmed reload.
Roles: independent layer drafts survive saves; scoped clear and malformed-file replacement work.
Verified: `make ci-scoped`; 24 browser scenario groups on owned fixtures, 32 geometry audits all ok.
Real fixture writes: agent package override and workspace/agent roles files; external install/update failures simulated.
visual-review: PASS — empty/blocked/error/dialog and narrow/wide screenshots read; overlayAudit ok.
visual-card: contained yes; readable yes; trigger usable yes; clip/double-scroll/dead hover no; next click obvious yes.
Evidence: `var/screenshots/cli-packages/`; closing/main CI results recorded there at integration.
Not done: real package downloads, physical-device acceptance, mobile-native config editor; no deploy or model turn.
Merge: fast-forward ready; code commit `5af1e57e`; close and full main CI are the integration gates.
