# 2026-09-22 — feat/packages-pane: one pane renders every CLI (ADR-0176, slice 3)
Shipped: the two panes are one. `GuestPackages.jsx` is gone from both apps and `CliPackages.jsx` no
longer picks a component by CLI name; `Packages.jsx` asks `GET /api/packages/report` and draws what
the driver declares — `GalleryBody` where the catalog is PiCode's gallery, `VendorBody` elsewhere —
every control gated by `Caps`/`Catalog`, the copy each surface spoke kept verbatim in `PANE_WORDS`
("Install to" / "Plugins go to"). The hardcoded `CLI_PACKAGES`/`GUEST_PACKAGES` lists are gone from
`web/shared/domain/cliPackages.js`: the engine is the source. The report grew
`WorkspaceName`/`AgentName`, `Isolated` and `Caps.Async` (a direct call that answers the new list vs.
a reserved job); `handlePackageUpdates` takes the `vendor` word of the layer in view;
`piDriver.scopesForContext` declares only the radios this read can honour. `TestCLIListMatchesJS` is
deleted — with the JS list gone the seam it guarded no longer exists.
Verified: `make ci-scoped` PASS (fmt,vet,hooks,go[6],test-js,build; 15 path(s) vs main), 54 JS tests,
the pane's assertions re-tabled onto `packagesApi`/`packagesNotes`/`packagesSurface` rather than
dropped. Visual pass on a scratch instance of this worktree: 8 captures read (Pi at agent scope, Omp
with its `browser` extension row, the scope pills at native scale, Claude Code's empty list with its
refusal note, the Omp row card, mobile list, mobile pane body, the isolation switch on),
`__picodeOverlayAudit()` ok in all 8. The switch measured end to end: `PATCH /api/agents/<id>
{packagesIsolated:true}` → report `"isolated":true` → box ticked (reverted) — the debt it closes.
Not run: `PICODE_PKGS_LIVE=1`.
visual-review: PASS (pane3-pi-agent.png, pane3-omp-row.png, pane3-claude-empty.png,
pane3-mobile-body.png, pane3-isolation-on.png; card 5/5)
Merge: main had moved. The `Packages.jsx` conflicts (both apps) resolved to this branch's merged pane;
the two `GuestPackages.jsx` stayed deleted, their copy folded into `PANE_WORDS`. Main's scope rename
rides along: the chip is the driver's `ScopeRow.Label` ("Global"), `layerLabel` answers `global`, and
the vendor fallback reads "Use global".

## Debts

- Omp's extension row still draws Enable/Disable and the CLI's disable verb does not know extensions —
  unchanged by this slice, owned by `docs/handoff/open/packages.md`.
