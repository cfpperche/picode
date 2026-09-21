# 2026-09-20 — test-isolation: the suite can no longer reach the live vault

Shipped: `scripts/go-test.sh` unsets `PICODE_DATA` for every package;
`internal/server`'s TestMain pins it to the temp sandbox and
`internal/catalog`'s removes it (catalog tests isolate by HOME), each with a
`TestSuiteIsVaultIsolated` guardrail. `docs/architecture/credentials.md` names
the rule and the incident. Also: the owner's vault was repaired by hand — 30
rows across 15 providers, all fixtures of `internal/server`,
`internal/usage`, `internal/credentials` and `internal/catalog` tests
(`sk-live`, `sk-one`, `kimi-key`, `secret`, `sk-ant-test-…`, `access-one`…),
written there in one 3-minute window today; one test-set identity
(`work@example.com`) was cleared from the migrated pi row. Three rows were kept
as real logins (anthropic + xAI oauth with long tokens, a 56-char
cheaperinference key).

Verified: the leak condition reproduced — with `PICODE_DATA=/home/goat/.picode`
exported, the vault-writing server tests left the real vault's mtime untouched.
`make ci-scoped` PASS (full). First attempt pinned PICODE_DATA process-wide in
catalog and failed `TestEnvKeyMakesAProviderSignedIn` (that package isolates by
HOME); fixed by removing the ambient value instead.

visual-review: n/a (no UI change)
Not done / debts: the mechanism is proved by construction, not by recovering
the exact process of that window — no run since reproduces the write, and the
three layers now make it impossible whichever env a gate is launched from.

## Next up

- Audit the other `PICODE_*` names the agent runtime exports
  (`internal/rpc/runtime.go`) for tests that read them process-wide.
