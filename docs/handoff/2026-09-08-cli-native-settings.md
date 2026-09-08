# 2026-09-08 — cli-native-settings: native settings under Agent CLIs

Shipped: ADR-0101; `#/clis/settings/pi` on desktop/mobile, legacy redirects,
explicit agent URLs, capability registry and app-owned embedded Pi editors.
Global/project/agent/key APIs and native files retained. Context validates the
agent, follows feed updates/deletion and reads its runtime mode before writes.
Mobile quick settings retains the conversation, draft and attachment.

Verified: `make ci-scoped` and `make close` PASS; native `TestPiSettings*`
and `TestPiKeys*` PASS. Route/context tests and both browser regression scripts
PASS on a synthetic fixture; no model turns or running-agent mutations.
visual-review: PASS — desktop/narrow/mobile, dark/light, empty/blocked/error,
quick sheet and tool selector; overlay audit ok. Card: yes/yes/yes/no/yes.
Evidence: `var/screenshots/cli-native-settings/` (screenshots read, logs/results).

Not done: other CLI editors; physical iPhone/PWA/IME acceptance; real running
agent tool-mode restart (existing restart decision matrix tested).
Merge: fast-forward ready at 599e7195; main integration CI follows this note.
Deploy: not requested; normal batch policy applies.
