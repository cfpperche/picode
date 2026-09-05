# Mobile decoupling implementation and acceptance

Owner-approved scope, 2026-09-05. Architecture:
[ADR-0072](../decisions/0072-independent-web-applications.md).
Baseline: `522844a2`, one statically imported bundle for both shells.

## Ownership and migration inventory

| Surface | Desktop | Mobile starting implementation |
|---|---|---|
| Entry, routes, app state, styles | Own responsive app | Own app, Now / Inbox / Work / More |
| Sidebar, workspace tabs, dashboard, sessions | Retained | Absent; mobile work/fleet screens retained |
| Conversation, composer, ask cards | Own components | Copied components; settings link replaces unconnected configuration chips |
| Terminals and extra keys | Own desktop view | Own terminal view and touch key bar |
| File editor, file tree, Git graph, Pin Studio | Retained | Absent; no CodeMirror or Tiptap dependency |
| Read-only changes and attached file previews | Own components | Copied because mobile opens them |
| Attachment sketch, Mermaid, 3D preview | Lazy where applicable | Retained lazy; they render inside mobile conversations/attachments |
| Apps and Inbox | Desktop split/resizable presentation | Own stacked presentation and pushed detail screens; no mouse split sizer |
| More, CLI manager, providers, settings, devices | Own pages | Copied pages loaded by section; duplicate page headers removed |
| Dialogs | Own responsive Radix/Vaul primitive | Own Vaul sheet at every width |
| Pairing, PWA, notifications | Same server contracts | Own pairing UI; same origin, manifest identity and worker registration |
| Schemas, API/feed, pure domain helpers, theme tokens | Explicit shared imports | Same contracts; no shared React tree |

All UI imports originating from the mobile entry were followed, including
literal dynamic imports. Desktop-only roots were excluded; source-unreachable
mobile files were removed after adaptation. The old global stylesheet was
split into shared token declarations and app-owned rules; desktop-only
selectors were removed from the mobile copy. This is an initial copied
implementation, ready for screen-by-screen redesign.

## Development and delivery

Run `npm ci` in `web/`. `make ui` starts the launcher on 5173, desktop on
5174 and mobile on 5175, proxying the existing Go API. For a single app:
`npm run dev:desktop` or `npm run dev:mobile` in `web/`.

`npm run build:desktop` and `npm run build:mobile` build just that application
and preserve sibling artifacts. `make build` builds the launcher and both
apps, then embeds their complete asset set in the Go binary. Full CI tests
both frontends and shared contracts. No backend schema migration or new
third-party dependency is needed. Each new mobile UI increment can stay
inside `web/mobile`; shared contract changes must validate both clients.

## Decision table

| Conditions | Expected action | Evidence |
|---|---|---|
| `/desktop/`, any width/query/saved choice | Mount desktop | `shared/client/shell.test.js`, narrow browser review |
| `/mobile/`, any width/query/saved choice | Mount mobile | Shell tests, wide sheet browser review |
| `/`, legacy desktop query (including both queries) | Desktop, persist choice if storage permits | Shell tests |
| `/`, legacy mobile query | Mobile, preserve hash and other query | Shell tests, legacy URL browser check |
| `/`, valid saved preference, no override | Saved application | Shell tests |
| `/`, no valid preference | Mobile at ≤767px, desktop otherwise | Shell tests |
| Storage denied | Launcher selection/switch remains available | Shell tests; other app preferences retain existing browser-storage requirements |
| Resize after mounting | Same app and presence identity; responsive layout only | Presence tests, browser root-identity check |
| Mobile secondary chunk fails | Visible error and Try again reload | Browser request abort + retry |
| API/WS, non-GET or other origin | Worker cache bypass | `tools/service-worker.test.mjs` |
| Root/app HTML, manifest or worker | Fetch fresh | Worker and Go cache tests |
| Hashed asset hit/miss | Application cache hit or fetch/cache successful response | Worker tests |
| Asset response fails | Do not cache it | Worker tests |
| Worker activates after upgrade | Remove obsolete owned caches; retain active/unrelated caches | Worker tests, browser update check |
| Notification + open mobile | Navigate and focus mobile | Worker tests |
| Notification + only desktop/legacy app window | Navigate and focus existing app | Worker tests |
| Notification + no eligible window | Open `/mobile/` with app hash | Worker tests |
| Invalid notification target | Fall back to `#/` | Worker tests |
| Manifest start URL changes | Preserve old implicit identity and root scope | Manifest test; physical-device acceptance remains open |
| Import crosses app boundaries, including through shared/CSS | Build/check fails | Boundary tests and resolved Vite module check |
| Build one app | Preserve sibling output | Build artifact hash comparison |
| Style changes in one app | Only its docs captures become stale | `scripts/docs-surfaces.test.mjs` |

## Validation record

`make ci` passed on the implementation (603 frontend tests plus Go and
package suites, production build, docs parity and Vale). Embedded UI/server
tests passed. Both scoped builds preserved every sibling output hash. Both
Vite entries mounted through the development launcher without browser errors.
The concurrent Rename icon and Windows task changes were incorporated from
`main`; the final combined gates are recorded in `docs/handoff.md`.

| Initial JavaScript entry | Raw kB | gzip estimate kB |
|---|---:|---:|
| Baseline (`522844a2`) | 2,557.19 | 787.75 |
| Desktop | 2,495.93 | 770.52 |
| Mobile | 561.83 | 175.55 |

Sizes are decimal kB, measured from production output. The gzip column is a
local compression estimate, not a network-transfer claim. Mobile's initial
entry is about 78% smaller than the baseline. Secondary screens, optional
preview/sketch modules, CSS and fonts are excluded from this entry comparison;
those dependencies still load when used. Entering through `/` also downloads
the small launcher. The complete binary still carries both apps.

Synthetic Chromium QA covered light/dark, 390px desktop/mobile and a 1280px
mobile sheet. Empty Inbox/Changes, unavailable Docker, validation errors and
chunk failure/retry were captured and read. Overlay audits passed. Resizing
preserved the mounted root; explicit and legacy links retained app/deep-link
identity. A same-origin upgrade of the old worker retained registration scope,
stored preference and agent hash while loading `/mobile/` and its new caches.
Curated evidence is in `docs/screenshots/split-*.png`; public captures were
regenerated from the final embedded fixture.

No model turns were started and this increment was not deployed or pushed.
Physical Safari/Android installation and real push delivery remain separate
owner-device acceptance. The strict video freshness audit reports three old
captures stale after source relocation; their integrity still passes, and
no old input hashes were rewritten to imply a fresh render.

## Adversarial review corrections

The composer opens an agent-only Settings sheet above the mounted conversation;
closing it preserves text, attachments and the connection. More keeps the full
settings page. Both paths persist agent configuration through the existing
PATCH endpoint and refresh the fleet. Controls update optimistically, prevent
concurrent edits and report failures inline. As on desktop, changing tool mode
restarts an active agent in its original managed/interactive mode; other edits
do not restart it. A failed restart is reported as a saved setting with failed
application, rather than pretending the save was rolled back.

Asset cache reads and writes are optional: storage failure must not turn an
HTTP success into an app load failure. Writes run under worker `waitUntil`
without delaying delivery. Boundary validation parses JavaScript/TypeScript
imports using Vite's existing parser/transformer and checks transitive shared
imports in the resolved graph. No new dependency was added.

| Conditions | Action | Regression evidence |
|---|---|---|
| Cache open/read/write rejects | Return successful network response | Worker tests, three injected storage failures |
| Cache hit / failed HTTP response | Use hit / do not persist failure | Existing worker tests |
| Tool/checklist selector inside Settings | Above sheet, within viewport | Browser overlay audit and screenshots |
| Composer text and image, open/close Settings | Same composer and unsent content | Browser acceptance |
| Agent configuration save succeeds | PATCH exact agent, refresh and retain values | Configuration tests and browser acceptance |
| Save fails | No restart; rollback optimistic controls and show error | Configuration tests and browser acceptance |
| Tool mode unchanged or agent stopped | Save without lifecycle calls | Configuration table tests |
| Tool mode changes on managed/interactive agent | Stop/start same runtime; close interactive view | Configuration table tests |
| Stop/start fails after save | Stop sequence; report saved configuration and failed restart | Configuration failure tests |
| No agent | No mutation | Configuration test; missing-agent browser state |
| Shared presentation import in JS/MJS/CJS/TS/MTS/CTS/JSX/TSX | Reject | Boundary tests |
| Transitive shared dependency imports React | Reject resolved graph | Graph regression test |
| Import syntax appears only in a string | Allow | Parser false-positive regression test |
