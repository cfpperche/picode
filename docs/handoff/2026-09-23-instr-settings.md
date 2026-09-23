# 2026-09-23 — feat/instr-settings: the settings that change what a CLI reads

Step 1 after the Instructions tab (owner's go-ahead: "pode seguir com o plano").

**Rows.** An Instructions group on four CLI Settings pages, each key checked on the installed version: Claude Code's Project instructions (`pluginConfigs.agents-md@builtin.options.instructionFiles`, Global layer only — the first field to use `Field.Scopes`, and `TestEveryDeclaredFieldRoundTrips` now checks only the layers a field allows), Codex's `project_doc_fallback_filenames` and `project_doc_max_bytes`, Hermes's `context_file_max_chars`, OpenCode's `instructions`. A finding whose fix is a setting opens that page: the CLAUDE.local.md finding opens Claude Code's, and a size cut by Hermes or Codex opens theirs.

**A defect from the previous slice, fixed here.** `isAgentTab` read `i:<workspace>` as an agent id, so any CLI page opened while the Instructions tab was selected got `?agentId=i:<workspace>` in its address (found while verifying the settings link on the scratch instance).

Verified: `make ci-scoped` and `make close`. Visual review on scratch `instrset` (screenshots read in subagents): the first pass FAILED on two defects, both fixed. The table header "Antigravity" ran into "Omp" under a 64px header cap left from the previous slice, now removed. Claude Code's select cut its option mid-word; the options are shorter and the default is spelled out in the help line. The second pass was PASS. Nothing deployed.
