# ADR-0072: Independent desktop and mobile web applications

- **Status**: accepted (owner approved the decoupling plan on 2026-09-05)
- **Date**: 2026-09-05
- **Supersedes**: ADR-0008's single frontend entry, ADR-0044's reuse of desktop
  presentation inside mobile, and ADR-0046's single shared modal primitive.
  Their framework, mobile product scope and interaction guidance still apply.

## Context

The old `web/src/main.jsx` statically imported both applications before
choosing a shell. Separate folders did not produce separate bundles. Global
CSS and copied-through imports made a desktop change affect the phone.
The owner wants desktop to remain responsive while mobile gets its own
implementation, initially copied from only the UI reachable on mobile.

The [reference study](../benchmarks/2026-09-05-mobile-decoupling.md) favors
sharing protocol and client behavior through explicit APIs while applications
own presentation. The existing Go server, same-origin authentication and
installed PWA identity do not need to change to obtain that boundary.

## Decision

Use npm workspaces with independently built `web/desktop` and `web/mobile`
applications. Each owns its entry, React tree, dialogs, styles, presentation
helpers and dependencies. `web/shared` exports individual contracts, client
adapters, domain helpers, release data and theme tokens. It has no React
components, presentation dependencies, wildcard export or root barrel.
Application imports may reach shared only through its explicit exports.

```
web/launcher ──────> shell selection helper
web/desktop ──┐
              ├───> web/shared ───> existing /api and /ws contracts
web/mobile ───┘
```

The root launcher has no React dependency. `/desktop/` and `/mobile/` are
explicit application choices at every width. Legacy query parameters,
then saved preference, then the 767px breakpoint choose the application
when entering `/`. The launcher preserves other query parameters and the
fragment. Only explicit navigation switches applications; rotation preserves
the mounted tree and its connections. Desktop offers a navigation disclosure
at narrow widths; mobile owns an always-sheet modal primitive and loads
secondary screens on demand, with retry when a screen chunk cannot load.

Both builds remain inside one Go release: `internal/web/public/desktop/`
and `mobile/`, plus the launcher and root PWA files. Each application build
cleans only its own output. The complete release build assembles all three.
The Vite resolved-module check and source boundary tests reject cross-app
imports and presentation leaking through shared. Tailwind scans only the
current application's sources. A single npm lockfile pins dependencies;
this change introduces no third-party package.

Keep `/sw.js`, registration scope `/`, cookies and existing subscriptions.
Manifest `id: "/?mobile=1"` preserves the old implicit identity while
`start_url` moves to `/mobile/`. The worker separates launcher/desktop/mobile
asset caches, removes obsolete owned caches, and never caches API responses
or HTML. Notification taps prefer an existing mobile window, then an existing
app window, otherwise open `/mobile/` with the original app fragment.
Presence reports the mounted application, rather than reinterpreting width.

## Consequences

Desktop and mobile presentation can evolve independently. Mobile contains
no desktop shell, sidebar, tab workspace, file editor/tree, Git graph or
Pin Studio. Conversation, attachment preview/sketch, terminal, changes,
Apps and settings remain because they are reachable mobile workflows.
The migration inventory and [acceptance matrix](../plans/mobile-decoupling.md)
define that initial scope; this is the foundation for subsequent mobile UX
increments, not a claim that a new phone product design has shipped.

Copied UI and application state adapters now require deliberate fixes in both
apps when applicable. Shared contracts get common tests; each app retains
its presentation behavior tests. Large optional sketch/preview dependencies
remain lazy. The binary includes duplicated presentation/vendor assets and
can grow despite the smaller initial mobile download. Releases remain
atomic; independent deployment or a second backend is not part of this ADR.

A rollback ships the previous complete binary/UI set and reloads open clients.
The old launcher query and manifest identity remain compatible. Do not mix
HTML from one build with assets from another or unregister the worker to
perform the migration. Browser evidence cannot certify physical iOS/Android
installation and delivery through their push services; that remains an
explicit device acceptance item.

## Alternatives considered

| Alternative | Why it was not selected |
|---|---|
| Dynamic-import only the two old shells | Smaller entry, but shared presentation/CSS still couples their development. |
| One Vite build with multiple HTML inputs | Valid output separation, but both app dependency graphs remain part of one build configuration. |
| Fully duplicate contracts and transport | UI independence does not justify diverging auth, feed, schemas or data semantics. |
| Separate repositories, origins or native framework | Adds releases, authentication and installation migration without helping this first web separation. |
