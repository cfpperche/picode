# ADR-0109: Native app surfaces — a first-party app's body may be a component compiled into the shell; the manifest names its surface

- **Status**: accepted (amends ADR-0036; first consumer: the Matrix app, `docs/plans/matrix-app.md` §4.1)
- **Date**: 2026-09-09
- **Boundary**: protocol — the apps manifest contract (`GET /api/apps`) gains an optional `surface` field, and a second surface kind that each shell negotiates before it renders a tile or a tab.

## Context

ADR-0036 made an app a manifest plus a **frozen** vocabulary of primitives
(list / detail / form / actions) rendered by the host from
`GET /api/apps/{id}/view`. Its 2026-08-31 amendment fixed the two body
surfaces for good: primitives for simple apps and for every sensitive
action, a sandboxed iframe for third-party bodies when the marketplace era
arrives. The same amendment says the pipeline "manifest → grid → tab →
badges → routes is surface-agnostic and carries over unchanged".

The Matrix (plan §2.1) needs a body neither surface can carry: a live grid
of xterm panes and conversation views. Both live in the host bundle and
hold host state — the `terms` registry (one xterm and one WebSocket attach
per terminal, `web/desktop/src/lib/terms.js`), the feed subscription, the
sockets. A primitives tree cannot name an xterm; an iframe on a separate
origin cannot reach the host's objects and would need a second attach per
pane, the tmux client the invariant forbids. What is missing is a third
surface kind for **first-party** bodies: a component compiled into the
shell, addressed by the same manifest, opened from the same tile, living on
the same tab strip and routes.

Facts that shape the design:

- Apps are versioned by `apiVersion` (ADR-0036, Zed's lesson). Two
  amendments (2026-09-01) already added optional fields without bumping it:
  the embedded UI ships in the same binary as the server, so both sides
  move together, and a client that ignores a field renders what it
  rendered before.
- Two shells consume the manifest today (desktop and mobile, ADR-0072) and
  they do not ship the same components. A surface kind has to be negotiated
  per shell, not per server.
- The desktop shows one surface at a time (`hidden` tabs stay mounted), and
  `ShellTerm` moves a terminal's pane into whichever host mounts it. Its
  `active` effect fitted and focused but did not re-append the pane, so a
  terminal tab revealed after its pane was shown elsewhere was empty (plan
  §2.2). `termSocket.js` had one stop switch, the one-way `closedByUser`; a
  panel scrolled out of view needs a reversible one (plan §4.5; phase 0
  measured the resume: the same xterm object, screen intact).
- The shell rewrites an unknown hash to `#/` at boot; `#/app/<id>` is a
  known route, so a native app tab survives a reload like any app tab
  (phase 0, result 7).

## Decision

`apps.Manifest` gains `Surface string` (`json:"surface,omitempty"`): `""`
is the primitives view every shell renders; `"native"`
(`apps.SurfaceNative`) means the body is a component compiled into a shell.
`APIVersion` stays 1 — the field is optional and additive, the precedent of
the 2026-09-01 amendments. A native app implements the same `App`
interface: `Manifest()` and `Badge()` as any app; `View()` answers one
`detail` block ("<Name> opens on the desktop.") so an older or foreign
client still renders something honest; `Action()` returns an error, which
the handler maps to 400.

Each shell owns a **registry** of the native surfaces it compiled in, keyed
by app id (`web/desktop/src/lib/nativeApps.js`; the mobile shell has none).
The contract's gate becomes `supportedApp(manifest, nativeSurfaces)`:
`apiVersion === 1` and (surface `""`, or surface `"native"` with the id in
the registry). Any other surface value is unsupported everywhere. The gate
decides the tile and the tab alike, so a tile is never enabled for an app
the shell could not open.

On the desktop, `App.jsx` mounts the registered component instead of
`AppSurface` for a native manifest — same `key`, kept mounted and `hidden`
while unselected like every surface — with one prop, **`host`**:
`{ fleet: { workspaces, freeAgents, terminals }, openTabs, openTab,
openInteractive, revealAgent, openFileTab, feed: subscribeFeed }`. `host` is
the client twin of Go's `apps.Host` and the **whole API a native app
gets**: the app never imports App state, and growth of the object is
reviewed the way `apps.Host` is. `openTabs` exists for one rule (plan §4.5):
a terminal with an open tab is owned by the tab. Server calls go through
the shared `api()` like any component. A shell without the surface renders
an honest line: the desktop tile is `app-tile-unsupported` with a title
saying it needs a newer PiCode (an unregistered id can only mean a stale
bundle), and a tab opened by deep link shows the same line inside the app
chrome; the phone lists a native app as **Desktop only** and does not
navigate, while `#/app/<id>` answers "<Name> is a desktop tool." with Back.

Two host behaviours travel with this. `ShellTerm`'s `active` effect
re-appends the pane when its host is not the current one — the pane
follows the visible host — and unmounting a host parks only a pane it still
holds. `termSocket.js` gains `suspendTermSocket(entry)` and
`isTermSocketSuspended(entry)`, a reversible stop beside the one-way
`closedByUser`: the socket closes silently (no "— detached —" line, no
retry), the xterm and its control block stay, `kickTermSocket` lifts it, and
`dropTermSocket` still wins.

The first native app is the hidden QA demo (`demo-native`, "Native demo",
icon `grid`) beside the primitives demo, present only with
`PICODE_DEMO_APP=1`: it renders the fleet's first terminal through
`TermSurface`, which is the pane hand-off case. The Matrix is the first
shipped consumer (phase 3).

**Refuse (carried from ADR-0036): third-party native code.** Native means
compiled into the binary and reviewed in this repository; the sandboxed
iframe stays the marketplace's first-class body for outsiders. Nothing here
loads JavaScript at runtime.

### Decision table (every row has a test)

| Manifest | Shell registry | Tile | Open |
|---|---|---|---|
| surface `""` (primitives) | any | enabled | `AppSurface` as today |
| surface `"native"`, id registered | desktop | enabled | the registered component with `host` |
| surface `"native"`, id not registered | desktop | unsupported, "needs a newer PiCode" | nothing (a deep-linked tab shows the same line) |
| surface `"native"` | mobile (no registry) | name + "Desktop only", no navigation | deep link → one line + Back |
| surface other value | any | unsupported | nothing |
| apiVersion ≠ 1 | any | unsupported (as today) | nothing |

Tests: `TestNativeDemoApp` and `TestNativeAppOnTheWire` (Go: the wire, the
view, the 400); `supportedApp gates on the surface a shell can draw`
(contract, one assertion per row — the mobile row is the gate with no
registry); `appTile: …` and `nativeSurfaceFor: …` (desktop registry, tile
and open columns). The mobile copy and the desktop mount are browser QA.

| Pane hand-off | Expected |
|---|---|
| terminal in its tab, then shown in the native app, then the tab revealed | the tab re-claims the pane (xterm rows inside the tab host) |
| app tab revealed again | the app re-claims it |
| `suspendTermSocket` then kick | same entry reconnects; no "— detached —" line was written while suspended |
| `dropTermSocket` after suspend | `closedByUser` wins; no reconnect ever |

Tests: the first two rows are browser QA on the scratch instance (the pane
is a DOM node; there is no unit harness for `ShellTerm`); the last two are
`termSocket.test.js`.

## Consequences

- The manifest is still the whole contract between an app and its shells;
  a new surface kind cost one optional field — no new host, no new route
  family. Everything ADR-0036 shipped — registry, tab family, grid, badges,
  routes — carries over unchanged, as its amendment promised.
- The `host` object is an API with a reviewed shape. It will be too small
  at first (the Matrix will ask for more — ADR-0036's dogfooding pattern);
  every addition is a line in `docs/architecture/routes.md`, never an
  import of App state.
- Two shells can disagree about an app, and the phone says so honestly. A
  future mobile Matrix registers the id on the phone — a registry entry,
  not a protocol change.
- Native apps are compiled in, so bundle growth is paid by every desktop
  user (phase 0: +24 KB gzip for the Matrix, below the lazy-import
  threshold). A heavier app lazy-imports its body behind the registered
  component; the registry shape does not change.
- If wrong: `surface` is one optional field; a shell that ignores it
  renders the app's one honest primitives view. Nothing on disk changes.

## Alternatives considered

- **A client-only surface flag** (the desktop decides by app id which
  component to mount; the server says nothing) — loses the phone's honest
  line and the version negotiation: a manifest that does not declare its
  surface cannot tell an older bundle it is missing something. The field is
  the smallest thing that lets every client be honest.
- **A separate route family outside the apps host** (`#/matrix`, its own
  sidebar entry) — the plan's §2.1 study: the apps host already owns the
  tile, the tab family, the badges and the routes; a parallel host would
  duplicate all of it and contradict ADR-0036's surface-agnostic pipeline.
- **Loading per-app JavaScript** (a bundle per app served by the binary) —
  the runtime-loaded-code line ADR-0036 refuses for good reason; even for
  first-party code it adds a loader, a cache story and a second build
  pipeline for no user-visible gain over compiling the component in.
- **Bumping `apiVersion` to 2** for the field — would refuse every app on
  a stale bundle instead of only the native one; the optional-field
  precedent of the 2026-09-01 amendments is exactly this case.

## Amendment 2026-09-11 — an app does not leak into PiCode's interface; the doors are a closed list

The owner's words, the day the Canvas's background control was found in
Preferences: *"canvas é um app standalone e não deve contaminar nenhuma GUI
no PiCode além do próprio app … isso deve ser uma diretriz de projeto: apps
não vazarem para interface do PiCode a menos que seja uma porta explícita
das diretrizes."*

The rule is **not** "an app may never appear outside itself" — the apps host
exists precisely so an app can be reached from the shell. The rule is that
**the ways an app reaches the host are declared by the host and are a closed
list**. An app names nothing about a host surface; the host names an app by
its **id** and by the fields of its manifest, and nothing else.

### The doors, as the code draws them today

| Door | Where it is | What the app supplies |
|---|---|---|
| **The tile on the Apps grid, and its badge** | `web/desktop/src/components/AppsGrid.jsx`, from `GET /api/apps`; the count/dot aggregates onto the sidebar tab icon | `Manifest.Name`, `Manifest.Icon`, `Badge()` — never markup |
| **One main tab** | `x:<id>` on the tab strip, peer of agents/terminals/editors; the host owns the strip, the ×, restore and the Inspector anchor | nothing: the host opens and closes it |
| **The app's own body, inside that tab** | `AppSurface` rendering `/api/apps/{id}/view`, or — `surface: "native"` — the component this shell registered (`web/desktop/src/lib/nativeApps.js`) | everything it draws, as long as it draws it **there** |
| **The manifest's `icon` key** | `web/desktop/src/components/AppIcon.jsx` — a fixed host map (`flask`, `inbox`, `grid`, `box`, `canvas`); an unknown key falls back to a letter tile | one string from that map. The host owns the glyph |
| **The `host` object** | `App.jsx`'s native mount: `{fleet, openTabs, openTab, openInteractive, revealAgent, openFileTab, feed}`, plus the `initialPath` / `onPathChange` pair | nothing — it is the whole API a native app gets, and it grows only by review, as `apps.Host` does |
| **The hash route the host owns** | `#/app/<id>[/<path>]` (`appHash` / `appRoute` / `appPath`, `web/desktop/src/lib/routes.js`); the phone answers the same route with its own honest line | a `path` string, which the host stamps into its own hash |

Anything else is a **leak**: a group in Preferences, a section inside another
surface, a row in a host list, a sidebar entry, a palette entry, a class in
the host's stylesheet, a key in the host's preferences that only the app
reads. A leak is not fixed by a commit — it needs a new door, which means an
amendment here saying what the door is and why the host owns it.

Direction is the test that settles most cases: **the host may name an app;
an app may never name a host surface.** `App.jsx` deciding not to auto-open
What's New while the Inbox's badge is blocking (`inboxNeedsYou`) is the host
reading a door it declared. The phone's home queue folding `/api/inbox` rows
into `needsYou` is the same direction and has its own decision (ADR-0044) —
which is exactly the form the rule asks for.

The 2026-08-31 amendment to ADR-0036 imagined primitives as "connective
tissue in host chrome … inbox items, sidebar rows, palette entries,
notifications". Nothing has ever shipped: no app contributes to host chrome
today. That paragraph is a direction, not a door. When contributed chrome is
built, it arrives as a declared door with its own decision.

### The two cases this amendment was written against

**Preferences → Appearance → Canvas background was a leak, and is gone.**
`Settings.jsx` imported `canvasPattern.js`, held the value in state and drew
a four-card radiogroup for a setting that means nothing outside one app; the
`⋯` menu's `Background…` item existed only to navigate to it. The control
now lives in the Canvas's own `⋯` menu as a **Background** submenu
(`DropdownMenu.Sub` with four `RadioItem`s, the current one ticked, the rows
staying open while you pick so the plane behind them is the preview), and
`PatternSwatch.jsx` moved into `components/canvas/` with it. **The value did
not move**: `web/shared/domain/canvasPattern.js` keeps its key
(`picode-canvas-pattern`), because the preference is per viewer and the app
owns it — only the *control* was in the wrong place. Preferences is back to
PiCode's own chrome, and its Appearance section is the theme and nothing
else.

**The Messages audit list is a door, and stays — reworded as the host's.**
ADR-0116 required a non-spatial place to audit link grants, because a
security control drawn with a mouse must be auditable somewhere that is not
the canvas. The honest reading is that **the grant is the host's**: it lives
in `peer_connections`, it changes who the mailbox lets talk, and a canvas is
only where a human happened to draw one. A host surface listing the host's
own grants is host business. So it is declared a door, with the conditions
that make it one:

- it is **Messages' section**, worded as contacts granted by a link drawn on
  a canvas — not "the Canvas, over here" (`CanvasLinks.jsx` →
  `web/desktop/src/components/GrantedContacts.jsx`, heading **Granted
  contacts**);
- it **imports nothing from `components/canvas/`**. Its dependencies are the
  shared domain modules (`canvas.js`, `canvasGrants.js`) and the host's own
  `appHash`, the same ones any host surface may use;
- it revokes through the same `DELETE …/edges/{id}` the canvas calls, so the
  two can never disagree.

Had those failed, it would have been a leak.

### The mechanical half

`web/tools/app-boundary.test.mjs` (`make test-js`) asserts the direction for
the first native app: nothing under `web/desktop/src/` outside
`components/canvas/` imports that tree or `canvasPattern.js`, except the
host's own native-surface mount in `App.jsx`; and `Settings.jsx` contains no
mention of a canvas at all. A second native app adds a row. The test cannot
see a *route* or an *id*, which is the point — those are the doors.

### Consequences

- A control that belongs to an app is found where the app is. The Canvas's
  ground is now changed on the plane, which is also where it is noticed.
- Preferences stops growing one group per app. That was the real cost: four
  cards for one app's texture set the precedent that every app may add a
  group, and eleven apps later Appearance is a junk drawer.
- A leak now has a name and a test, so it fails in CI rather than in review.
- **If wrong**: the list is descriptive, not prophetic — a genuine need for
  contributed host chrome is an amendment adding a door, not a reason to
  stop declaring them.
