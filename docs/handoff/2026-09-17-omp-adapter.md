# 2026-09-17 — omp-adapter: Omp is a full catalog row

Shipped: editable launch defaults, PATH wrapper with presence lease
(maintenance subcommands exec past it; protocol modes incl. `=` forms mark
non-TUI), Activity through the pi-shaped terminal-state extension injected
with `-e` (extension API measured live on 2026-09-17), pi-shaped lifecycle
carrying the real npm package (`@oh-my-pi/pi-coding-agent`,
`npmUpdateOnly` like claude-code). `PI_CODING_AGENT_DIR` is launcher-owned;
integration seed bumped v3→v4 (the muse v2→v3 precedent);
`normalizeTerminalCLI` (server) learned omp.
Adversarial pass, measured: `--trusted-extension` + injected `-e` = omp
refuses the run — preview names it and prepare refuses
(`ompTrustedExtensionConflict`, 5-row table); `--no-extensions` + `-e`
measured fine; `--export` renders kept a false lease (now exec past);
`--mode=rpc` escaped the `$1:$2` case (glob fix). Guard rows in
TestWrapperPresenceLease.
Verified: make close green (fmt,vet,go[21],docs); scratch `ompc` — Check
setup green (`omp/18.2.4`), Activity toggle ON via seed, Customize present,
lifecycle menu with a real npm update-check saved, overlayAudit ok.
visual-review: PASS (omp-adapter-card.png, omp-lifecycle-menu.png,
omp-update-check.png; card 5/5)
Not done / debts: live TUI Ready/Working acceptance needs omp auth in a
PiCode terminal (scratch has no omp credentials by design); Fatia 6
handoff (Reader/Prompter/Writer) still open.
Merge: fast-forward ready.

## Next up

- Omp Fatia 6 (handoff): Reader on the JSONL, Prompter via positional prompt, Writer after the minimum-viable-file spike — a real session now exists to clone. Plan: `docs/plans/omp-cli.md`.

## Debts

- Live omp TUI activity acceptance (Ready/Working in a real PiCode terminal) is external until the owner runs omp authenticated inside PiCode.
