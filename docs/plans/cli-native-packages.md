# Packages under Agent CLIs

Status: implemented; scoped gates and fixture validation passed.

Move Pi package management to `#/clis/packages/pi` and known configuration
editors to `#/clis/packages/pi/config/<package>`. Each app owns its view;
shared code owns routes, capabilities and context validation. Preserve native
package APIs, files, install/remove commands and agent package overrides.
Adapt the Cursor/t3code study's contextual, reload-safe navigation
(`docs/benchmarks/2026-08-24-adopt-t3code-paseo-cursor.md`).

## Work plan

1. Add canonical routes, compatibility redirects and explicit target validation.
2. Embed the existing package views in Agent CLIs; migrate links, update badges
   and the desktop configuration editor. Keep mobile configuration's existing
   desktop boundary visible with an action that preserves the same URL.
3. Guard refresh failures, asynchronous target changes and mutation recovery.
4. Exercise routes and native scopes with unit/browser tests on owned fixtures;
   inspect empty, blocked, error, dialog and narrow/wide screenshots.
5. Update living docs; run scoped gates, close, integrate and run full main CI.

## Decision table

| Conditions | Action | Validation |
|---|---|---|
| Legacy packages/config link | Replace URL; retain explicit or legacy pane context | Route tests + browser |
| Canonical URL without context | Machine packages only; no ambient fallback | Context tests + browser |
| Workspace or agent URL | Resolve identities; retain through reload/config/back | Context tests + browser |
| Free agent | Machine + agent; no workspace scope | Context tests + browser |
| Missing/mismatched identity, invalid scope/URL | Block editing; offer machine packages | Context tests + browser |
| Unsupported CLI/config adapter | Explicit unavailable state; no Pi API mutation | Route tests + browser |
| Context refresh fails transiently | Retain drafts/results; block writes until retry | Browser |
| Resolved workspace/agent file location changes | Retain draft, block writes, require confirmed reload | Context-key tests + browser |
| List read fails | Show retry, never a fabricated empty list | Browser |
| Install/update/remove fails or is pending | Visible progress/error; no success until response | Browser |
| Scope/config navigation | Encode scope and context; old draft never changes target | Browser |
| Desktop roles config | Preserve workspace/agent layers, save/reset/conflicts | Existing Go tests + browser |
| Mobile config deep link | Offer desktop layout with exact URL preserved | Browser |
| Packages active | No dependency on terminal inventory or CLI installation jobs | Browser |
| Narrow menu/dialog | Selected tab reachable; overlays contained | Screenshot + overlay audit |

Real package downloads and physical-device acceptance are separate from the
fixture validation. No production service restart or model turn is needed.

## Validation evidence

`make ci-scoped` passed. `scripts/qa-cli-packages.mjs` passed 24 scenario
groups across desktop/mobile with 32 successful geometry audits. Screenshots
were read for empty, blocked, error, dialog and narrow/wide states. Real
fixture writes covered agent package overrides and workspace/agent roles,
independent drafts, scoped clear and malformed-file replacement. External
package-manager responses were simulated; no real downloads or model turns.
Evidence: `var/screenshots/cli-packages/`.
