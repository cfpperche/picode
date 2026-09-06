# 2026-09-06 — feat/hermes-cli: Hermes Agent terminals and activity hooks

Shipped: catalog id `hermes`; sessions from `~/.hermes/state.db` (cli/tui,
active home, `--resume <id>`); PYTHONPATH sitecustomize after unwrapping
the official trampoline; `register_from_config` deepcopy so `config.yaml`
is never written; no `HERMES_HOME` overlay. Hook map: LLM start/end and
approvals. `setup`/`model`/`auth` (including after `-p`) skip the patch.

Verified: `make close` then `make ci` on main; venv dogfood (v0.18.2, nine
events, user config mtime unchanged); visual-review PASS (scratch :8462).
Live catalog lists Hermes Agent installed. Deploy `0.1.0+405fed1` left 21/21
tmux sessions alive. systemd active, health ok.

visual-review: PASS (scratch; overlayAudit ok).

Owner dogfood 2026-09-06: Activity reporting on, resume of
`20260906_202219_54a6a2`, sidebar Working then Ready.

Not done: needs-you; `cli-v1-*` screenshots; profile scan. Hermes may
write `shell-hooks-allowlist.json`.

Merge: fast-forwarded main to `405fed15`; deployed `0.1.0+405fed1`.
