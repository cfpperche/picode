# 2026-09-13 — communication-consent-move: workspace consent does not transfer

Shipped: onboarding row 5 now has a direct store test for an actual owner `workspace_id` change (agent and terminal). Old credential, mint and stale preparation fail; the new workspace needs a fresh selection and a new capability. JS Off state covers a moved owner with an inactive connection. No product API was added — owners still have no move verb.
Verified: `TestPeerParticipationDoesNotTransferToAnotherWorkspace`; JS `peerParticipants.test.js`; `make close` ci-scoped PASS.
visual-review: n/a
Not done / debts: rows 9/12 still partial; activation acceptance and Codex resume `--` remain the next communication cuts.
Merge: fast-forward ready after this close.
