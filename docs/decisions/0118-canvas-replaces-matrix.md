# ADR-0118: Canvas replaces Matrix — one engine, one name

- **Status**: accepted (the owner's call, 2026-09-11: "quero remover já o grid
  e renomear a funcionalidade de matrix para canvas")
- **Date**: 2026-09-11
- **Boundary**: persistence **and** protocol. Three tables, their columns, the
  route family, every event name, the app id and its hash all change name;
  grid coordinates are converted once and the mode column is dropped.
  Supersedes ADR-0113 (the two modes) and renames what ADR-0108 and ADR-0116
  describe.

## Context

The Matrix shipped with two layout engines by design (plan
`docs/plans/matrix-canvas.md` §3): grid mode on react-grid-layout, kept
untouched while canvas mode was built on React Flow, with the end state left
to be decided "after the canvas has been used, with evidence rather than
argument". The evidence now exists, and it is one-sided:

- The only matrix in the owner's instance is in **canvas** mode.
- The C0 spike measured what grid mode does that the canvas does not:
  vertical compaction, collision resolution and reflow to container width.
  None of the three has been asked for since the canvas shipped.
- The cost is not symmetric. Canvas mode is lazy-imported, so only a reader
  who opens it pays its 49 KB. Grid mode rides in the desktop's **main
  chunk** — every PiCode load carries react-grid-layout and react-resizable,
  including readers who never open the app.
- The recurring cost is worse than the bundle: every later panel feature has
  to work in two hosts with two drag/resize/selection models. C3's node kinds
  and phase 4's chat body each paid that tax twice.

The name followed the same drift. "Matrix" named a grid of panels; what
shipped is a plane you pan and zoom, and the owner asks for the product name
to say so.

## Decision

**Canvas is the only layout engine and the only name.**

1. **Grid mode is removed**: `MatrixGrid`, the mode switch, the per-mode
   validation, the grid→canvas transform's reverse direction, and the
   `react-grid-layout` and `react-resizable` dependencies.
2. **Coordinates convert once, in the migration**, with the transform
   ADR-0113 already defined and tested (a grid cell is 8 canvas units wide and
   3 tall). Every panel of a grid-mode matrix is rewritten in canvas units in
   the same transaction that drops `matrices.mode`. A matrix already in canvas
   mode is untouched.
3. **The rename goes all the way down**: tables `matrices` → `canvases`,
   `matrix_panels` → `canvas_panels`, `matrix_edges` → `canvas_edges`; routes
   `/api/matrices…` → `/api/canvases…`; events `matrix.*` → `canvas.*`; the app
   id `matrix` → `canvas`; the hash `#/app/matrix/<id>` → `#/app/canvas/<id>`;
   the Go and JS module names with them. A half-renamed system — a "Canvas"
   whose API says matrix — is the outcome this refuses.
4. **Old deep links keep working**: `#/app/matrix[/<id>]` redirects to the
   canvas route by replacement, the way ADR-0101/0102/0103 redirected their
   moved surfaces. An `x:matrix` tab id restored from `localStorage` opens the
   canvas app. No compatibility layer is kept on the **API**: the routes are
   consumed only by this repo's own clients, which ship in the same binary.
5. **The word "matrix" leaves the product surface**, including the guide and
   the What's New already published for 0.2.0 — that entry is history and is
   not rewritten; the next release's notes carry the rename.

## Consequences

- The desktop's main chunk loses react-grid-layout and react-resizable; the
  surface itself becomes lazy, so a reader who never opens Canvas pays for
  none of it.
- One host, one drag model, one set of panel behaviours. The next node kind is
  written once.
- **Two accepted ADRs are renamed, not rewritten.** 0108 and 0116 keep their
  text; this ADR is the pointer that says what their nouns are called now.
  0113's two-mode decision is superseded outright.
- A migration that rewrites coordinates is irreversible in practice: going
  back means restoring the column and packing panels into 12 columns again.
  The transform is tested in both directions and the old direction stays in
  the domain module as the documented inverse, unused by any caller.
- Anyone holding an old bookmark keeps working; anyone holding an old API
  client does not. There is no such client outside this repository.

## Alternatives considered

- **Keep both engines.** Rejected by the owner with the evidence above; the
  tax is paid on every future panel feature, and the bundle cost falls on
  readers who never open the app.
- **Rename only the product surface, keep `matrix` in the schema and API.**
  Rejected: the drift is permanent and every future reader pays it in
  confusion. Pre-alpha, one instance and one in-repo client is exactly when a
  full rename is cheap.
- **Keep grid mode as a "tidy" preset of the canvas.** Deferred, not refused:
  the Tidy action already packs panels in reading order, and vertical
  compaction could become a canvas behaviour later without a second engine.

## Amendment 2026-09-25 — old addresses retired (owner)

The owner retired the compatibility addresses this ADR kept. §4 is retired: `#/app/matrix[/<id>]`, a saved `x:matrix` tab and the `picode-matrix-last` / `picode-matrix-view:<id>` keys are no longer read. An old bookmark now lands where any unknown address does.
