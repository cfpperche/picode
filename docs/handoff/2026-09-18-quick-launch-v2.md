# 2026-09-18 — quick-launch-v2

Phase 2 of the quick launch settings (`feat/quick-launch-v2`, landed on main).

## Done
- Repeatable-flag machinery in `cliLaunchPresets.js`: `readQuickList` /
  `applyQuickList` (block replaced in place, quoted values round-trip) and
  the `QuickListField` textarea (local text so trailing newlines survive).
- Additional folders (`--add-dir`) on Claude Code, Codex and Omp; flags
  verified against the installed binaries (claude variadic, codex DIR,
  omp 18.2.4 space+joined). 18 preset tests green (36 total in file).
- Docs travel: benchmark receipts, architecture sentence, public guide row,
  own changelog fragment.

## Gates
- `ci-scoped: PASS`; visual-review: PASS (v2-claude-preview.png shows the
  preview with both `--add-dir` pairs; first round FAIL was framing only).
