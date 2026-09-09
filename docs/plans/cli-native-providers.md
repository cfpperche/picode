# Native providers under Agent CLIs

Owner approved the migration on 2026-09-08. Move the existing Pi provider
surface into the app-owned Agent CLIs frame, with an explicit CLI capability
and machine scope. Preserve accounts, keys, OAuth, usage and verification.
The llama.cpp manager keeps its separate route. No credential migration,
new dependency or provider backend is part of this work.

The Cursor/t3code adaptation is contextual, reload-safe navigation from the
composer and model controls (see the 2026-08-24 benchmark study).

## Implementation

1. Add canonical/legacy route helpers and a Pi-only provider capability.
2. Embed each app's provider editor in `AgentClisFrame`; retain page geometry,
   add loading/error/retry and keep successful catalog data during refresh.
3. Move menu/search entries and update composer/model shortcuts and OAuth
   return URLs. Preserve the explicit desktop/mobile application path.
4. Validate the decision table in unit tests and an owned browser fixture;
   read dark/light desktop/mobile screenshots, then close and integrate.

## Decision table

| Conditions | Action | Validation |
| --- | --- | --- |
| Canonical Pi URL | Render machine providers and the Pi selector | Routes + browser |
| Legacy desktop/mobile list URL or omitted CLI | Replace with canonical Pi list | Routes + browser |
| Legacy/canonical new URL | Open Add provider; closing returns to canonical list | Routes + browser |
| Unknown CLI, malformed identity or extra path | Show blocked state; do not mount Pi editor | Routes + browser |
| Explicit agent/workspace/scope query | Block scoped editing; offer machine providers | Routes + browser |
| Old llama alias | Keep llama.cpp manager reachable | App route tests + browser |
| Catalog pending / initial failure | Skeleton / error and retry, never false empty list | Browser |
| Refresh failure after success | Keep roster and draft; display retry | Browser |
| Key save fails / succeeds | Keep draft and error / refresh roster and close | Browser with intercepted writes |
| OAuth starts in either app | Return to that app's canonical Pi provider list | Helper tests + browser |
| OAuth completes / fails | Refresh and close / keep visible failure | Browser with simulated OAuth |
| Editor closes/unmounts during OAuth | Ignore late UI completion and navigation | Browser with delayed response |
| Account actions, quotas and environment credentials | Retain existing native endpoints and action restrictions | Browser with synthetic catalog |
| Model/composer shortcut or menu search | Open the appropriate canonical provider route | App route tests + browser |
| Tabs at wide/narrow sizes | Keep width/alignment; selected tab remains reachable | Geometry + screenshots |

The native loopback callback still only accepts localhost/127.0.0.1 return
URLs; remote browser behavior retains that existing restriction.
Real vendor OAuth, real credential changes and physical-device acceptance
remain external to fixture validation.
