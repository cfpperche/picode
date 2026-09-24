# 2026-09-24 · feat/skills-toggles

ADR-0196 slice 3: per-skill switches on the Skills tab (desktop and mobile).

- **What:** each CLI's own switch, written and read back by `internal/skills/toggle.go`:
  - Claude Code `skillOverrides` (Global in `settings.json`, workspace in `settings.local.json`);
  - Codex `[[skills.config]]` by `SKILL.md` path, user file only;
  - OpenCode `permission.skill`;
  - Omp `skills.ignoredSkills` (the workspace list replaces the machine list, so the first write copies it);
  - Grok and Hermes `disabled`, machine only;
  - Muse's own `muse skills disable|enable`.

  Pi and Antigravity say they have no switch. Route `POST /api/skills/toggle` rewrites only rows the reader found; it answers 409 on a stale file.
- **Primitives in `internal/clisettings`:** `SetScalar`, `Keys`, `Empty`, `Text`, `AppendArrayTable`, `RemoveArrayTables`.
- **Formats measured live** in a throwaway HOME. The Codex, Omp, Grok and Hermes rows in `docs/plans/skills.md` are corrected.
- **Adversarial review** found two bugs that corrupted files, both fixed with regressions in `TestSwitchReviewRegressions`:
  - a TOML header with a trailing comment was not seen as a header, so a toggle could delete `config.toml`;
  - a PyYAML indentless list, when removed, left `skills` as a list.

  Also fixed: Codex path+name semantics, a no-op deleting a `{}` file, OpenCode merge order and walk-up. The rest are debts in `docs/handoff/open/skills.md`.
- **Gates:**
  - `make ci-scoped` PASS;
  - live `PICODE_SKILLS_LIVE=1 TestSwitchLiveVendorReadBack` PASS for all 7 CLIs (the `~/.picode/bin` wrappers are dropped from PATH).
- **visual-review:** PASS (`var/screenshots/skills-toggles/`, overlayAudit ok, card 5/5). Files were byte-identical after off→on.
- **Not deployed.** Next: slice 5, the marketplace (the seed sources are the owner's call).
