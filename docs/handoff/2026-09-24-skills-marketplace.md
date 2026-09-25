# 2026-09-24 — skills-marketplace: a Marketplace for every CLI (ADR-0196 slice 5)

Shipped: `internal/skillcatalog` (the mcpcatalog pattern). Seeds anthropics/skills, openai/skills, vercel-labs/agent-skills
(owner's call 2026-09-24; github/awesome-copilot offered, not taken); the person's sources in setting `skills.sources` (GitHub
repo/folder/#ref, or an https site with a .well-known index; a local folder is refused); skills.sh only with `skills.skillssh`
on and a query of ≥ 2 chars (undocumented `/api/search`, no token, no descriptions, no audits, so the card links to its page).
A GitHub source is one unauthenticated tree call plus raw SKILL.md headers (64 KiB, 8 workers, ≤ 300 per source; hidden
folders count, openai keeps `skills/.curated`). Cache `<data>/skills-catalog.json` per source, read in the background when
missing or a day old; a failure keeps the last list and retries after 30 min; a read's end is an ephemeral `skills.catalog`
feed event (no polling). Routes: GET `/api/skills/catalog?q=`, POST/DELETE `/api/skills/sources`, PUT `/api/skills/skillssh`.
UI (both apps): Skills pane tabs Installed | Marketplace; cards in the Packages style, search debounced 250 ms, Sources section
(Built in, counts, add, remove), Radix switch for skills.sh with the "sent to skills.sh (Vercel)" note; Install opens
AddSkillDialog on the card's source with the skill preselected and scrolled into view; scope chips on both dialog steps;
Installed = same name AND the lock's source; an empty Installed tab offers "Browse the Marketplace". Owner's call: a skill
whose name differs from its folder installs under the name (5 of 73 seeds; `Candidate.Folder`, preview note); installed
folders still report the mismatch; the plan's decision table says so. Copy: skipped-items note ("Left out N item(s) that
are not regular files…"), license chip only for a short name, "N scan warning(s)".
Verified: skillcatalog httptest suite (+race), a server route test against a fake GitHub, the live test (`PICODE_SKILLS_LIVE=1`:
20+44+9 = 73 skills in 3.8 s); scratch on the real network: list, search, install pdf and a renamed Vercel skill, add a source
(40 skills), local folder refused, skills.sh search and a skills.sh card preselected among 20. Blind spot: the "1 scan
warning" badge tooltip is in no capture.
visual-review: PASS, card 5/5 (one FAIL round: LICENSE chip, hidden preselected row, "1 critical", plural/jargon note).
Gotcha: a scratch stage inside the worktree (`var/qa/…/skills/stage`) holding Go files breaks `make fmt-check` (go list picks them up); clear it.
Not done / debts: not deployed; slice 6 next and three new debts in `docs/handoff/open/skills.md`.
Merge: main moved; merge main, rerun `make close`, then fast-forward.
