# ADR-0045: One modal primitive — dialog on the desktop, sheet on the phone

- **Status**: accepted
- **Date**: 2026-09-01

## Context

The owner, testing the mobile shell (ADR-0044): "New agent abre como um
drawer de baixo pra cima, o dialog Choose Folder já não abre assim".
Sixteen files imported `@radix-ui/react-dialog` directly and rendered a
centred `.dlg` card at every width; one (`CreateForm`) had grown its own
`useMedia` branch into a Vaul drawer. Two modal shapes on one phone
screen — sometimes nested (Choose folder opens from inside New agent).

## Decision

`web/src/components/ResponsiveDialog.jsx` is the only modal primitive.
It exposes the Radix Dialog API (`Root / Portal / Overlay / Content /
Title / Description / Close`, plus an `Alert` set for confirmations), so
a file switches by changing one import line. At `min-width: 720px` it is
the Radix dialog; below, the same tree is a Vaul bottom sheet — handle,
swipe to dismiss, safe-area padding, 44px actions, 16px inputs. The
visual switch is one CSS rule, `.dlg.dlg-sheet`, which beats every
`.dlg-*` width, so the existing class vocabulary keeps meaning "this
dialog's content" on both shapes.

The rule is enforced, not remembered: `web/src/lib/dialogPolicy.test.js`
fails `make test-js` when any file outside the primitive imports
`@radix-ui/react-dialog`, `@radix-ui/react-alert-dialog` or `vaul`. Two
allowlisted exceptions, both desktop-only surfaces the mobile shell never
mounts: `Palette.jsx` (cmdk inside a dialog) and `Hotkeys.jsx`.

Anchored popovers (`ContextMenu`, `GitGraphBranches`, `SearchCombo`,
`SessionBar`) are not modals and stay Radix Popover; they are a
separate question if they ever reach the phone.

## Consequences

- **Easier**: every modal — create, choose folder, confirm, prompt,
  usage, MCP add, providers add, share, session info, llama — behaves the
  same on a phone without each author thinking about it; a new dialog
  gets the right shape by importing the primitive.
- **Harder**: the sheet is a Vaul component, so Radix-specific props the
  sheet does not know are silently ignored below 720px (none in use
  today beyond `onOpenAutoFocus` / `onCloseAutoFocus` / `onEscapeKeyDown`,
  which Vaul forwards to its inner Radix dialog).
- **Accepted cost**: an `Alert` dialog loses "no dismiss on overlay
  click" on the phone — a swipe down is Cancel. Every confirm resolves
  `false` on dismiss already, so nothing destructive can happen from it.
- **If wrong**: the primitive is one file and the imports are one line
  each; reverting is mechanical.

## Alternatives considered

| Alternative | Why not |
|---|---|
| A `@media (max-width: 719px) .dlg { bottom: 0 … }` rule on the Radix dialog | Gets the position, not the behaviour: no handle, no swipe, no drag physics, and the Radix dialog still animates from the centre. |
| Keep `useMedia` branches per component (the `CreateForm` way) | Sixteen copies of the same fork; the drift that caused this ADR. |
| Make every dialog a sheet on the desktop too | A sheet at 1440px is a mobile pattern out of place; the desktop bar (docs/benchmarks.md) is Linear/Stripe-style centred dialogs. |
