# 2026-09-09 — peer-native-validation: verify native conversation messaging
Shipped: documentation records real native resume acceptance for Claude Code 2.1.266, Codex 0.153.4 and OpenCode 1.18.29; no runtime or UI change.
Verified: three native session/connection bindings; Claude ↔ Codex and OpenCode ↔ Claude roundtrips, four messages with valid reply links and acknowledgments; one earlier retired fixture message also acknowledged.
Verified: OpenCode native metadata confirms zai/glm-5.3-flash, variant max; explicit prompts initiated each turn and disposable settings preapproved only communication tools.
Evidence: root var/screenshots/peer-native-validation-20260909/ contains launch snapshots, native tool/model records, receipts and assertions.
visual-review: n/a (documentation of runtime validation only).
Not done / debts: Grok/Hermes automatic setup, physical-device acceptance and orphan private-file cleanup remain open; native session discovery is best effort and inline OpenCode configuration requires JSON objects.
Merge: fast-forward ready at close-summary; full make ci is required on main at integration. No deploy.
