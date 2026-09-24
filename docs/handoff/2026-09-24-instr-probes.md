# 2026-09-24 — instr-probes

Sentinel runs for the resolver rules that came from docs (owner asked for
items 2 and 3 of the AGENTS.md debts). Results in the study's "Measured
2026-09-24" section.

- Muse Code 1.3.0: confirmed with no model call (`--provider echo`, the
  session record lists each loaded rules file).
- Omp 18.2.11: confirmed with an `-e` extension that dumps the system prompt
  and exits before the provider; one fix — `.agent/` beats `.agents/`.
- Antigravity 1.2.10: four runs, none loaded at start → cells now unknown.
  A headless run with `--dangerously-skip-permissions` was refused by the
  session's permission classifier; not worked around. The probe folder was
  trusted once through Antigravity's own prompt (`~/.gemini/antigravity-cli/settings.json`).
