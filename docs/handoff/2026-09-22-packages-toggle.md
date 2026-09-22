# 2026-09-22 — feat/packages-toggle: an extension row's toggle writes what Omp reads
Shipped: `clipkgs.OmpExtensionToggle` (ADR-0176 slice 5's last debt), asked by
`guestDriver.Toggle` before it asks for the vendor's verb, exactly as `Remove`
does. Enable/Disable on one of Omp's own `extensions` now writes
`disabledExtensions` instead of running `omp plugin disable`, which cannot know
a configured extension. Workspace layer: `<ws>/.omp/settings.json` gets the
CLI's id (`extension-module:<name>`) spliced in (or out, for the enable
direction, and the key created when the file has none) with every other byte
preserved and the result re-parsed and compared; a repeat is a no-op. User
layer: the CLI's own `omp config set disabledExtensions '<json array>'` in the
user's directory. `insertArrayElement` is the shared splice's insert half;
`Removal` became `Mutation`. A plugin row, every other CLI and the pane are
untouched — the control already rendered and the route already answered the
CLI's fresh list.
Verified: `make ci-scoped` PASS — `(fmt,vet,hooks,go[6]; 9 path(s) vs main)`;
`make close` green. Five new tests: the workspace fixture (id added, id already
present, the enable direction, an unrelated entry and an unrelated key
untouched), the created key, the user-layer command the driver runs (a stub
`omp` on PATH, a temp HOME — nothing touched the real `~/.omp`), the driver's
mapping, and the route both ways.
visual-review: n/a
Not done / debts: none. `TestCLIPackagesOmpExtensions` was re-tabled on purpose
(server fixture extracted; assertions unchanged).
Merge: fast-forward ready
