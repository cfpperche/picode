# 2026-09-23 — skills-inventory: read-only Skills tab (ADR-0196 slice 1)

Shipped: `#/clis/<cli>/skills` in both apps. `internal/skills` declares where each of the nine
CLIs reads skills and the precedence order, measured per CLI; the reader reports status
(loaded/shadowed/needs-trust/if-trusted/invalid) with provenance from `skills-lock.json`,
`~/.agents/.skill-lock.json` and the Hermes hub lock; Digest matches the skills CLI's own
computedHash. `GET /api/skills/report`. Packages menu copy now "Plugins, extensions, updates"
plus a Skills search shortcut.
Verified: go tests for `internal/skills` and the route; `TestLocaleOrderMatchesNode` and
`TestDigestMatchesTheSkillsCLI` against Node; live `TestLiveMuseParity`
(`PICODE_SKILLS_LIVE=1`, sandboxed HOME) agrees with real `muse skills list --json`; manual
comparison with `grok inspect --json` and `hermes skills list` (differences are vendor rules,
recorded as debts in `docs/handoff/open/skills.md`). `make ci-scoped` PASS; `make test-js` green.
visual-review: PASS on a scratch instance — desktop Muse/Pi/Codex/agy and 390px Muse captures
read in a subagent. Round 1 FAIL (clipped status chips, truncated token column, off-centre
scope labels, Muse trust line with no action); fixed and recaptured. `__picodeOverlayAudit()` ok.
Blind spot: Grok and Hermes rows include skills those CLIs hide by vendor rules; trust state
is readable only for Pi; bundled/plugin skills are not shown as rows.
Merge: fast-forward ready.
