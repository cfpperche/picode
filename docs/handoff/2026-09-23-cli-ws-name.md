# 2026-09-23 — feat/cli-ws-name: Agent CLI panes name the bound workspace

Shipped: Agent CLI panes now name the bound workspace instead of the generic "This workspace" — layer switchers in Settings/Models/Memory, the Checks card "Comes from" column, Packages scope radios, setting-row select options (omp's role save-to), connector-package tags in Connectors. Pi's pane and the connectors/MCP pane already named the workspace; guest panes follow the same rule via new shared helpers `namedScope`/`namedLayers` (web/shared/domain/cliNative.js) and `sourceLabel(source, workspaceName)` (web/shared/domain/cliDoctor.js). Server: internal/pkgs/guest.go `scopesForGuestContext` prefix-replaces "This workspace" keeping qualifiers, so Claude Code's local layer reads "<workspace> (local)" instead of a second bare "<workspace>" radio (new test TestGuestWorkspaceScopeKeepsItsQualifier). Fragment docs/changelog.d/cli-ws-name.md.

Verified: make ci-scoped PASS (fmt,vet,hooks,go[5],test-js,build,living-docs) before and after merging main. Scratch instance (scripts/qa-scratch.sh clis-wsname): Omp Settings switcher "Global | QA", Models "Global | QA", Grok Memory "Global | QA"; Omp Settings bound to fixture workspace "Delivery" shows doctor row tools.approvalMode → "Comes from: Delivery"; Claude Code Packages radios "Global / QA / QA (local)" after the guest.go fix; __picodeOverlayAudit() ok. Screenshots in var/screenshots/clis-ws-name/ (not committed). Blind spot: mobile panes share the components but were verified only in the desktop DOM, not with a phone-width screenshot.

visual-review: PASS — scratch instance (scripts/qa-scratch.sh clis-wsname) confirmed every named surface renders workspace names; 5-question card: all yes/5.

Merge: fast-forward ready.

## Debts

- Mobile panes share the renamed components but were verified only in the desktop DOM, not with a phone-width screenshot (docs/handoff/open/clis-workspace-names.md).
- Pi settings deep link (#/clis/<cli>/settings?workspaceId=…) drops workspaceId (pre-existing parser behavior — context arrives from the sidebar selection), so a deep link without a selected agent still shows the generic label (docs/handoff/open/clis-workspace-names.md).
