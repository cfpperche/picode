# 2026-09-25 — feat/dead-code-sweep: remove code nothing reaches
Commits: cfdfe75d6, 93b00fd71, a9e3dd708, ec5e1ffc0, aab535157, 62bb81449.
JS: never-imported files (UsageDialog in both apps, desktop createSubmit, mobile AgentPageBar/GitAssetPreview/KindChip,
browserDomains + its test); 17 dead exports (listed in cfdfe75d6); 21 icons and their lucide imports.
Go: 24 functions unreachable even from tests (`deadcode -test` on Linux and GOOS=windows now report nothing), U1000
unused test types and never-set fields (probe.bearer, fake auth); errDevServerStopUnsupported kept (the other-platform
build tag uses it). usage.Identity's doc comment moved to the method that remains.
CSS: 124 classes that no JS/JSX/HTML/Go/Rust source names (hyphen-aware grep incl. desktop-shell/ui, management.html),
rules removed with postcss (a first regex pass corrupted comments and was reverted before commit).
Kept on purpose after reading the ADRs: web/browser/src/lib/shellVersion.js (ADR-0216's forward helper shellSupports)
and the :root[data-picode-frame] rules (ADR-0121 "permanent convention"; no shell sets the attribute today).
Method: inventory by grep + deadcode + staticcheck; blind spot: class or export names built at runtime from strings
(the uncertain ones were left alone), and callers outside this repo (HTTP routes, scripts run by hand).
Verified: `make ci-scoped` PASS (one run failed TestResolveInstalledCLIRetriesThroughAnUpdateWindow, a timing test
unrelated to the diff; 3/3 alone and on rerun). Scratch overlay audit ok on desktop /, /clis/pi, packages, connectors,
inbox, preferences, canvas, automations and mobile /, /work, /more, /inbox. Not run inside the Windows shell.
visual-review: PASS (dcs-desktop-home.png, dcs-mobile-work.png; the mobile select's font matched production).
Not done: everything deferred to the owner is in docs/handoff/open/dead-code.md.
Merge: fast-forward ready.

## Debts

- Dead-code candidates left for the owner (routes, Tauri commands, scripts, test-only Go, frame mode, a flake): docs/handoff/open/dead-code.md
