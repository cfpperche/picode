# Model roles (ADR-0028, ADR-0033)

> Part of [PiCode's architecture](../architecture.md) (ADR-0105: one file per subsystem). Edit here; the index only links.

Opt-in pi package at `packages/pi-roles/` (MIT; the rest of this tree is
PolyForm Noncommercial). Users install it with `pi install -l` / `#/clis/pi/packages`;
a missing `<cwd>/.pi/roles.json` leaves it dormant. Three builtin behaviours
(`default`, `vision`, `plan`) plus named custom presets. `default` is the
switch-back target, not a startup override — per-agent `--model` stays with
ADR-0009. `/roles edit|add|remove` writes the workspace file, or
`<cwd>/.pi/roles/<id>.json` when `PI_ROLES_AGENT` is set (PiCode puts the
agent id in that env on RPC and TUI start). Overlay slots win; the rest is
inherited. Composer `/` lists those commands only while the agent is running
(ADR-0029). Since ADR-0099, `#/clis/pi/packages/config/pi-roles` views and edits both
layers (workspace file + agent overlay, effective merge, scoped reset) — the
files stay the only source of truth. ADR-0119 leaves this editor in place;
the descriptor system is additive and does not describe pi-roles.
