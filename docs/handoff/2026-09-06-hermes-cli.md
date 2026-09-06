# 2026-09-06 — feat/hermes-cli: Hermes Agent terminals and activity hooks

Two commits on this branch, not merged, not deployed.

**PR 1 (`fe34ae5f`):** catalog id `hermes` / command `hermes`; lists cli/tui
sessions from `~/.hermes/state.db` (active home only, folder + messages);
resume `hermes --resume <id>` (verified Hermes Agent v0.18.2 `--help`);
presence lease; Check setup clips `--version` to the first line. No
`HERMES_HOME` overlay, no profile scan, no cost/size on guest rows.

**PR 2 (this session):** session `PYTHONPATH` sitecustomize after unwrapping
the official trampoline (`unset PYTHONPATH`). Patches
`agent.shell_hooks.register_from_config` with a deepcopy so `load_config` /
`save_config` never see PiCode entries and `~/.hermes/config.yaml` is not
written. `--accept-hooks` + `HERMES_ACCEPT_HOOKS=1`. Maintenance subcommands
(`setup`, `model`, `auth`, …), including after `-p`/`--profile`, skip the
patch. `pre_tool_call` is not registered (gate, not a status signal).

Hook map: `pre_llm_call` / `post_approval_response` → working;
`on_session_start` / `on_session_end` / `on_session_reset` /
`on_session_finalize` / `post_llm_call` / `subagent_stop` → idle;
`pre_approval_request` → needs-you.

Verified: sitecustomize against the real v0.18.2 venv with `HERMES_HOME` in
`/tmp` — nine events registered, `load_config` clean, user `config.yaml`
mtime unchanged; allowlist written only in the temp home. Wrapper decision
table covers `--tui`, `--resume`, `chat`, prompt args, `setup`,
`-p NAME setup`, `version`. visual-review PASS (scratch
`http://127.0.0.1:8462`, overlayAudit ok: activity-on summary, launch
details, empty terminals, mobile 390px).

Not done: live Hermes TUI turn (working / needs-you); live `--resume` in a
PiCode terminal; `cli-v1-*` / `cli-v2-desktop-defaults` screenshot refresh;
profile scan with `-p`. Hermes may still write
`~/.hermes/shell-hooks-allowlist.json` when auto-accepting.
