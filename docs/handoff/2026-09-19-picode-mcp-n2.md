# 2026-09-19 — feat/picode-mcp-n2 (ADR-0154, N2: measured, documented)

Question: can the other guest CLIs take PiCode's tool servers at launch?
Measured against the installed binaries (`--help`) and the vendor docs:
Grok, Hermes Agent, Muse Code and Antigravity have no flag or environment
variable that adds MCP servers for one run — only their config files
(Grok `~/.grok/config.toml` + `.grok/config.toml`; Hermes
`~/.hermes/config.yaml`; Muse `~/.config/muse/settings.json`; Antigravity
`~/.gemini/config/mcp_config.json`, docs also name `.agents/mcp_config.json`).
`GROK_HOME`, `HERMES_HOME`, `--profile` swap the whole configuration,
auth included, so they are not a tool switch. Omp is a pi fork: no MCP
flag (`--config` overlays `config.yml`, which has no MCP block), but `-e`
loads an extension per launch, so pi's own packages are its natural path;
the typebox shim is unverified live and the path is not built.

Product: nothing to change — `hasToolLaunchMechanism` already limits the
Launch settings group to Claude Code, Codex and OpenCode and the PUT guards
name the Connectors pane. Guide and plan now state the measured truth.

## Next up
- Owner's call: build Omp `-e` injection (embed the pi packages in the daemon) only if Omp gets real use.
