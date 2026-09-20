# 2026-09-20 — desktop-overlay-resize: refresh full-window chrome geometry

Implemented: native layer cache invalidation includes viewport width, height and device pixel ratio, including windows with no native page or overlay. The trusted IPC payload is unchanged.
Verified: three regression tests fail against the previous cache key and pass with the fix; all 11 native-layer JS tests pass. Scoped CI passed formatting, vet, hooks, JS tests and build.
Native QA: scratch port 23481 was verified against this worktree's server metadata and process 349977; isolated Windows QA process 11512 used that instance. Native chrome client and region sizes matched at 2040×1320, maximized 2560×1528, and restored 2040×1320; the bottom-right point stayed covered in each state.
visual-review: PASS for maximized/restored empty chrome and final modal composition on the verified scratch instance, including the corrected fixture charset and all five screenshot review questions. The live page advanced from frame 366 to 487 beneath the modal, remained visible, and reported native layers ready; overlay audits passed at maximize and restore.
Native QA isolation: the initial port-8471 attempt reached another scratch instance and was discarded; it provides no acceptance evidence for this change.

| Conditions | Action | Evidence |
|---|---|---|
| Identical viewport, scale and layer geometry | Suppress duplicate IPC | All three new tests |
| Maximize or restore; empty page/overlay lists | Resend native region update | Maximize/restore tests, native region measurements and screenshots |
| Device pixel ratio changes; empty page/overlay lists | Resend native region update | DPI-change test |

Scope: physical multi-monitor, IME and the broader overlay acceptance matrix remain tracked in `docs/handoff/open/live-desktop-overlays.md`; unit tests are not physical DPI acceptance.
Deployment: owner-authorized forced deploy and supported desktop restart completed at 11:04 local time. Production serves 0.3.1+b6c9f19 with healthy boot c84109a6995ea49c; Windows shell process 20352 responds and its installed executable matches the build hash.
Merge: b6c9f190 fast-forwarded to main and full `make ci` passed. Task-owned scratch, fixture and Windows QA processes were stopped; the merged worktree was removed. Evidence is retained under `var/screenshots/desktop-overlay-resize/` and `var/qa/desktop-overlay-resize-results/`.
