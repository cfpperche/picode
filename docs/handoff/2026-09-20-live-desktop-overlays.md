# 2026-09-20 — live-desktop-overlays: live native page composition

Implemented: transparent `main-content` chrome above native pages, with a bounded, trusted `chrome_layers` geometry bridge (ADR-0161); existing React components retain their state and handlers.
Compatibility: the updated shell uses live composition; older shells retain the previous overlay path. No production deployment was performed by this session.
Verified: production Windows release build without the QA feature and isolated product QA release/LTO build passed; scoped CI passed with full scope; 20 focused JS tests (including serialized updates and pending disposal) and two Rust geometry tests passed.
Native evidence: composed screenshots covered the empty page, suggestions, menu, modal, reconnecting state, resized product viewport (852×658), and Ctrl+K palette in light and dark themes; page animation advanced under overlays, and native rejection/recovery and tab return were exercised.
Error recovery: native screenshot review passed the inline address-area error with a distinct Retry action, closed suggestions and no page blanking; a targeted CDP pointer click after fault removal cleared the error, restored the native-ready marker and returned an `ok` overlay audit.
visual-review: PASS for those screenshot-reviewed states at 150 percent scaling; this is not an all-surface or all-DPI input verdict.
Input: guarded physical typing and resize passed in the isolated prototype. Product QA used targeted CDP input and native resize; its guarded physical input attempt was refused before sending input because the product window was not foreground.
Acceptance scope and remaining input/OS combinations are recorded in `docs/handoff/open/live-desktop-overlays.md`; the decision table is in `docs/plans/live-desktop-overlays.md`.
Merge: this note records branch validation before integration; the landing commit and full-main CI are recorded by Git and the gate logs.
