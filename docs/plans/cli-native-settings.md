# Native settings under Agent CLIs

Status: implemented and validated (2026-09-08). ADR-0101.
Local `make ci-scoped`, `make close`, native Pi API tests and both browser
regression scripts passed. Integration CI is recorded in the session handoff.

## Scope

Move Pi Settings into Agent CLIs on desktop and mobile, preserving native
files, APIs, trust, agent runtime semantics and the mobile quick settings sheet.
Register Pi as the first supported native settings editor. No other CLI editor,
provider/package migration or runtime migration is included.

## Work

1. Add canonical routes, legacy redirects and explicit agent context.
2. Add Settings navigation and app-owned embedded editors.
3. Preserve persistence; surface loading/save failures and scope clearly.
4. Run route, API and browser regressions; review screenshots in scratch.
5. Update documentation, changelog and handoff; close the isolated branch.

## Decision table

| Conditions | Action / acceptance |
|---|---|
| Old desktop/mobile URL, context available | Replace with canonical Pi URL and preserve agent |
| Canonical URL, no agent | Global + Keys only |
| Explicit valid agent | Read/write that agent and its actual workspace; reload preserves target |
| Explicit missing agent | Error and recovery action; no fallback write |
| Trusted workspace | Project layer editable with existing inheritance |
| Untrusted workspace | Project write blocked; link to agent for Trust |
| Unsupported or malformed CLI | Unavailable state; never render Pi fields |
| Terminal inventory/jobs unavailable | Native settings still load independently |
| Initial read fails | Error with retry; no endless skeleton |
| Save fails | Visible error, rollback, retain pending pattern text |
| Scoped-models shortcut | Reveal section after loading |
| Mobile quick settings and return | Same conversation, draft and attachments retained |
| Changed agent tool mode | Preserve existing explicit runtime restart behavior |

## Validation

- `cliSettings.test.js`: canonical/legacy routes, explicit IDs, malformed and
  unsupported CLIs, free/bound agent context, authoritative runtime mode,
  missing identities and failed reads.
- `application-routes.test.mjs` and existing route tests cover both shells.
- `scripts/qa-cli-settings.mjs` on the synthetic fixture covers global and
  agent saves, legacy/reload identity, untrusted/trusted project controls,
  unsupported/missing targets, save rollback and retained pattern text,
  read retry, independent loading, focus, mobile Back, feed updates/deletion.
- `scripts/qa-mobile-settings.mjs` verifies retained draft/attachment, model,
  tools/checklist writes and save-error rollback on the quick sheet.
- Existing `TestPiSettings*`, `TestPiKeys*` and `agentConfig.test.js` cover
  native persistence, trust rejection, key bindings and restart decisions.
- Screenshots read in `var/screenshots/cli-native-settings/`: desktop,
  narrow desktop, mobile, empty/blocked/error, light/dark and quick overlays.

Physical iPhone/PWA/IME acceptance and a real running-agent tool-mode restart
are not exercised in this task. Existing restart semantics have automated
coverage; the browser fixtures keep every agent stopped and send no model turn.

