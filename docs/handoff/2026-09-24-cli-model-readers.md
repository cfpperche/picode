# 2026-09-24 — cli-model-readers: Codex, OpenCode and Muse list their models in PiCode

Shipped: climodels readers `codex` (`codex debug models`), `opencode` (`opencode models --verbose`,
`OPENCODE_DISABLE_MODELS_FETCH=1`, folder-dependent) and `muse` (`muse serve` JSON-RPC `model/list`,
`MUSE_NO_AUTO_UPDATE=1`), each with measured inputs and a trimmed real fixture; the native settings Model field
for those three keeps its text input and gains **Choose…** (two-line rows: name, then id · context).
Measured and left without a reader: Claude Code (rewrites ~/.claude.json per call), Grok (rewrites ~/.grok/docs),
Antigravity (network + OAuth refresh), Hermes (no public listing). The measurement itself refreshed the owner's
Antigravity token and Hermes auth.json once — reported to the owner.
Verified: `make ci-scoped` PASS; readers live once (7 / 105 / 4 models); visual-review PASS on scratch after five
rounds (D1 "No matches" over the empty note, a narrow field, a 320px popover cap, cut ids, a mobile regression and
indent — all fixed); overlayAudit ok.

## Next up

- Claude Code / Grok / Antigravity readers only on an explicit Refresh, if the owner wants them (they write files)
