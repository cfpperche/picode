# Mobile webapp v2

Status: research complete; implementation complete for the current mobile scope; integration gates below. The owner
requested a complete mobile redesign on 2026-09-07 after a web benchmark
study. Expansion beyond the current mobile product is a pending scope
clarification because ADR-0044/0072 explicitly exclude editor/tree/Git graph.

References: [fresh benchmark study](../benchmarks/2026-09-07-mobile-v2.md).
Existing boundaries: ADR-0072 (independent apps), ADR-0044 (mobile product),
ADR-0048 (event feed), ADR-0086 (scratch QA and batch deployment).

## Product direction

Give the active work the screen. Keep Now, Inbox, Work and More as stable
top-level destinations. A conversation, terminal or tool takes the whole
viewport with one compact contextual header. Secondary actions open a sheet.
Retain context and unsent work while navigating. Use the shared theme tokens,
mobile-owned presentation and existing Radix/Vaul/native controls.

## Complete current-surface inventory

| Area | v2 work and acceptance |
|---|---|
| Now | Decisions first, compact running/results lists, honest loading/error recovery and quick access to work. |
| Work | Searchable project/agent/terminal lists; reduce nested-card padding; preserve all creation/start/stop/remove actions. |
| Conversation | Compact multiline composer, secondary attachments/settings on demand, retained drafts, visible Stop and pending questions, useful scroll area. |
| Terminal | Full-height real xterm, compact contextual actions, retry attach errors, user-armed IME accessory, retained attachment path. |
| Inbox | Full-width list/detail, visible touch actions, filters/search, preserve reply/back behavior. |
| Changes and previews | Compact file context, readable patch/content, one horizontal scroll region when code requires it. |
| More | Grouped/searchable entry points and consistent mobile layout for every subpage. |
| Settings and service tools | Providers, Pi settings, preferences, packages, integrations/MCPs, Agent CLIs, llama.cpp, devices, system and notifications. Audit every nested route. |
| Sessions and Automations | Mobile-owned list/detail/editor flows, preserved scope, recoverable errors, existing server permissions and confirmations. |
| Apps | Existing app manifests, dynamic lists/details/forms/actions, including Docker; validate states and narrow layouts. |
| Access/PWA | Pairing, install, sharing, notifications, reconnect, theme and existing deep links. |

Sessions and Automations close existing server-workflow gaps in the mobile
tool catalog. Expansion candidate: Files/editor and Git history/actions. Reuse server endpoints and
mobile-owned components; do not interpret a CLI terminal as a managed agent.
Document any approved change of the mobile product boundary in a new ADR.

## Dependencies and ownership

| Layer | Boundary |
|---|---|
| `web/mobile` | Owns shell, route state, tool views, composer, sheets and mobile CSS. |
| `web/shared` | Existing headless API/feed/contracts/domain helpers and tokens; no React UI. |
| `web/desktop` | Independent responsive application; never imported by mobile. |
| Go server | Existing permissions, lifecycle and filesystem ownership remain authoritative. |
| Libraries | Prefer installed React, Radix, Vaul, xterm and native controls. Any extra dependency requires concrete justification. |

## Delivery sequence

1. Establish compact layout, context retention and loading/error primitives.
2. Redesign conversation/composer and terminal; measure content area.
3. Redesign Now/Work/More and cover all settings/app routes.
4. Close functional coverage gaps and any approved product expansion.
5. Exercise decision tables and browser journeys; inspect screenshots.
6. Run `make close`, integrate with current main, run `make ci`; leave
   device-only acceptance explicit. Deployment follows the existing batch process.

## Validation contract

- Same fixtures before/after at 320, 360, 390 and 430 CSS px; landscape and
  tablet widths. No page-wide horizontal overflow; overlays stay in view.
- Target at 390 x 844: empty composer no taller than 120 px; contextual
  header no taller than 56 px before safe-area padding. Measure actual
  unobscured content, not just the conversation element's bounding box.
- Capture populated, empty, blocked, loading and error states. Read the
  screenshots; run `window.__picodeOverlayAudit()` after overlays.
- Conversation tests: multiline input, attachments, submit failure, waiting
  and abort, switch away/back, two different agents, stale reconnect events.
- Terminal tests: attach/error/retry, start/stop, keyboard tap gate, resize,
  extra keys, copy/paste and attachment flow in a disposable session.
- Navigation tests: every route, nested CLIs/apps/settings, deep link, back,
  draft retention, focus restoration and reduced motion.
- Real iOS Safari/Home Screen and Android Chrome/PWA acceptance remains
  separate from emulation: IME, safe area, rotation, background/resume, push.

## Decision table and evidence

| Conditions | Expected action | Evidence |
|---|---|---|
| Switch screen or agent with unsent text/attachments | Restore each agent's own draft without sending. | `agentDrafts.test.js`; `qa-mobile-chat-v2.mjs`. |
| Submit accepted; with/without edits during request | Clear only acknowledged content; preserve subsequent edits. | `agentDrafts.test.js`; pending-send browser journey. |
| Submit fails, is refused or lacks acknowledgement | Keep content and offer Retry; do not claim queued delivery. | Draft unit table; intercepted browser failures and retries. |
| Agent streaming/waiting; prompt/steer/follow-up | Keep Stop and route the selected message kind correctly. | 12-row `submissionKind` table; socket/browser regression. |
| Agent stopped; a message is accepted | Use established start-on-message path. | Browser acceptance is mocked; no paid model turn in QA. |
| Terminal opens without user tap | Keep keyboard and accessory hidden. | Tools browser journey against a real disposable xterm. |
| Terminal focus and emulated viewport shrink | Keep accessory/prompt in view and refit xterm. | Tools screenshots 320–430 px and landscape; physical IME remains owner acceptance. |
| Missing resource vs failed initial source | Show gone only after required data; otherwise Retry. | Fleet source table and integrated deep-link error journey. |
| Back to Work after searching or scrolling | Restore query and scroll without focusing the keyboard. | `qa-mobile-navigation-v2.mjs`: 933 → 933 px at 320/390. |
| Create finishes while a preceding fleet read is pending | Await a subsequent read before opening the new resource. | `fleetReload.test.js`; intercepted create/navigation journey. |
| Partial fleet refresh fails after prior success | Retain successful sources and expose recovery. | Eight failure masks in `fleetReads.test.js`. |
| Changes initial/patch/refresh fails; owner changes | Retry, retain prior files/patch, discard old-owner results. | `qa-mobile-tools-v2.mjs` (11 rows, disposable real shell). |
| Sessions CLI changes while old request is pending | Discard the old reply; retain current list on refresh error. | `qa-mobile-workflows-v2.mjs`. |
| Automation list/detail/runs or handoff preview fails | Show error and explicit Retry; keep last successful data. | `qa-mobile-workflows-v2.mjs`. |
| Active-source handoff returns 409; forced request is pending or fails | Confirm once, keep controls locked while pending, show execution errors inside the sheet. | Workflow 409/cancel/confirm/pending/error table; no real native-session writes. |
| Toggle/run/delete/rotate while another action is pending | Block repeated mutations synchronously; retain progress through refresh; rollback rejected toggles. | `qa-mobile-automation-mutations.mjs`: duplicate rotation, rollback in both directions, cancel, success/failure. |
| Automation editor is dirty; save fails or navigation starts | Retain fields on failure; confirm discard before leaving. | Draft unit table and workflow browser journey. |
| Agent settings PATCH succeeds/fails | Apply saved config or restore previous values; keep conversation draft. | Existing `qa-mobile-settings.mjs` passes model/tools/checklist/rollback. |
| Catalog prints setup prose or valid capability columns | Ignore prose, accept yes/no combinations. | Eight Go parser cases in `TestParseListModelsCapabilityColumns`. |

## Verification notes

The empty composer is 114 px at 390 × 844 (baseline 191 px); the agent
header stays 52 px. Work's default toolbar is one row; Search expands only
when requested. Evidence lives in `var/screenshots/mobile-v2/` and remains
uncommitted. Browser scripts require a loopback synthetic fixture.

Physical iOS Safari/Home Screen and Android Chrome/PWA acceptance is still
open: real IME, safe areas, rotation, install, push and background/resume.
Mocked voice/dictation paths do not verify hardware microphone permission.
Native handoff writes and scheduled/webhook delivery reuse existing server
tests; mobile browser mutation scenarios intercept these requests.
Full zoom accessibility remains limited by the existing ADR-0044 viewport
policy. Files/editor and Git history/actions await the owner's scope answer;
this delivery does not silently change those documented exclusions.
