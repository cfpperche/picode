# Missions evidence review — 2026-09-24

Branch: `feat/missions-evidence-review` (`bf4eaf2d6` at close-summary).

## Delivered

- Desktop and mobile Mission review now place **Read evidence** beside the review actions and lead to the criterion reports.
- Current reports are readable in the criteria; older evidence entries in History expand to show their recorded details.
- File names in evidence notes are labeled as references, since PiCode does not retain or open the cited files as artifacts.
- Updated the Missions guide, architecture note, changelog fragment, and the durable Missions debt in the same branch.

## Evidence and next action

- Scratch visual review: PASS at desktop 1280 px and mobile 390 px; `window.__picodeOverlayAudit()` returned `ok`.
- `make ci-scoped`: PASS. `make close-summary` reported that `main` advanced; merge `main`, rerun `make close`, then land from the root after review.
- Durable artifact retention and physical-phone validation remain open in `docs/handoff/open/missions.md`; the owner must not treat file references as viewable attachments.
