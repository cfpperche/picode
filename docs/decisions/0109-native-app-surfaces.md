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
