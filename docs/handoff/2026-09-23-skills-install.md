# 2026-09-23 — skills-install: install/remove/update skills (ADR-0196 slice 2)

Shipped: `internal/skills`: `source.go` (GitHub codeload tarball, `.well-known` index with
mandatory sha256 digests, local folder; public-only HTTP client; extraction keeps regular
files, refuses traversal/absolute paths; limits 50 MiB/200 MiB/20,000 files), `scan.go`
(advisory findings), `install.go` (preview stage 30 min; canonical `.agents/skills` + links
for Claude Code/Hermes/Antigravity where installed; `skills-lock.json` v1 and
`~/.agents/.skill-lock.json` v3 written preserving other entries, re-read before atomic
write, never migrated; Conflict codes exists/update/critical/modified/unlocked/invalid/gone/
stale/lock-version). Routes `POST /api/skills/preview`, `POST`/`DELETE /api/skills`,
`POST /api/skills/update`, `GET /api/skills/updates`; ephemeral `skills.changed`. UI: Add
skill dialog (both apps), Check for updates, Update, Remove with inline confirmation.
Verified: decision-table tests in `internal/skills/install_test.go` (install/adopt/replace/
critical/invalid/expired/lock version/stale/machine links/remove rows/github check+update/
unreachable/well-known digests/extraction escapes/SSRF refusal/scan rules); route test; live
`TestLiveSkillsCLIHash`: Digest equals the real `npx skills` computedHash (mixed-case files);
`make ci-scoped` PASS; `make test-js` green.
visual-review: PASS (scratch; round 1 FAIL — clipped conflict button, legend overlapping the
list border, meaningless truncated local source, cut placeholder on phone, duplicate Remove —
fixed and recaptured; overlayAudit ok).
Blind spot: global entries PiCode writes have an empty skillFolderHash; Check downloads the
whole repository tarball; Remove cannot yet name agents that use the skill.
Merge: fast-forward ready.
