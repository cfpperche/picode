# Study: quick launch settings — the flags people actually type

- **Date:** 2026-09-17
- **Sources:** owner screenshot of the Launch tab (`#/clis/pi`, 2026-09-17
  17:17, 2559×1599); live docs fetched 2026-09-17 —
  [Vibe Kanban, "Agent Profiles & Configuration"](https://www.vibekanban.com/docs/settings/agent-configurations),
  [VS Code, "User and workspace settings"](https://code.visualstudio.com/docs/configure/settings)
  (already studied in [2026-09-12-cli-settings-ux.md](2026-09-12-cli-settings-ux.md)),
  [pi CLI reference](https://github.com/badlogic/pi-mono/blob/main/packages/coding-agent/README.md),
  [Claude Code CLI flags](https://code.claude.com/docs/en/cli),
  [Codex command line options](https://developers.openai.com/codex/cli/reference),
  [OpenCode CLI](https://opencode.ai/docs/cli/). Flags for the remaining
  CLIs verified against the installed binaries the same day — `--help`
  plus a `--flag --version` parse check per quick flag: grok 1.0.34
  (`--permission-mode`, `--always-approve`), Hermes Agent v0.21.3
  (`--model`, `--reasoning`, `--yolo`), omp 18.2.4 (`--model`), codex-cli
  0.154.0 (`--yolo` accepted); `--add-dir` verified the same way on
  claude (variadic), codex and omp 18.2.4 (space and joined forms).
  In-house bars: [benchmarks.md](../benchmarks.md) (progressive disclosure,
  one page width, UI copy rule), ADR-0069 (launch model: argument array,
  never a shell string), ADR-0070 (profiles are copies, not links).
- **Scope:** the launch editor (`LaunchFields`) and the Launch pane's
  profile section, desktop and mobile. The launch model — CLI defaults,
  terminal overrides, redacted diagnostics, explicit Save — stays as it is.

## The problem, measured

The Launch pane answers "is it configured?" well and "configure it" poorly.
Every common change — pick a model, loosen or tighten approvals — requires
knowing the exact flag and typing it into the raw argument textarea:

| Measured | Value | Why it hurts |
|---|---|---|
| Controls for common launch choices | **0** | Model/approvals are per-CLI flags typed by hand |
| Raw textareas exposed by default | 4 (executable, args, PATH, env) | Full surface shown before any need |
| Profiles | functional, collapsed `<details>` at the bottom | The reusable-settings feature is invisible |
| New-terminal editor | profile select only when profiles exist | Discovery depends on prior setup |

## What the sources do

| Pattern | Receipt | Why it matters here |
|---|---|---|
| **Named profiles, default preselected** | Vibe Kanban: "The default configuration appears pre-selected in the agent dropdown" | PiCode already stores profiles (`cli_profiles`, ADR-0070); they only need surface. |
| **Env-var presets per profile** | Vibe Kanban: profiles "point… at different providers — each profile fully isolated" | PiCode's env field covers it; the profile is the delivery vehicle, not a new store. |
| **Commonly Used group first** | VS Code: "There is a **Commonly Used** group at the top" | Frequent launch choices sit above the raw fields; advanced stays behind a reveal. |
| **Model as a first-class control** | Every CLI: pi `--model`/`--thinking`, Claude Code `--model` + `--permission-mode`, Codex `-m`/`--sandbox`/`--ask-for-approval`, OpenCode `--model`/`--auto` | These five cover the cross-CLI denominator: model, autonomy, effort, sandbox. |
| **Danger named, not hidden** | Claude Code docs: bypassPermissions "offers no protection against prompt injection"; Codex: "avoid `--dangerously-bypass-approvals-and-sandbox` unless… a dedicated sandbox VM" | A dangerous pick shows a one-line warning inline — honesty bar, no extra friction. |

## What we refuse

- **A new persistence concept.** Quick controls patch the same draft
  argument array; Save, preview, overrides and profiles keep their
  contracts (ADR-0069/0070). No backend change, no schema change.
- **Shell-string presets.** Arguments stay individual values, never an
  evaluated expression (ADR-0069).
- **Editing the CLI's own config files in the pane** (the 2026-09-12
  refusal again): quick controls only compose launch arguments.
- **Hard-blocking on CLI version drift.** Flags are verified against the
  vendor docs, not the installed binary; the launch preview and Check
  setup remain the truth. A flag a future CLI drops degrades to the
  advanced textarea, not to an error.
- **Covering CLIs without a verified or editable surface.** Muse Code and
  Antigravity have no adapter, so their launch screens are read-only by
  design — there is nothing to attach a control to. Grok's `--sandbox` and
  `--reasoning-effort` accept no documented value list in 1.0.34, so they
  stay advanced-only rather than offering guessed options. A wrong
  generated flag is worse than no control.
- **Conflicting flags coexisting.** Codex `--yolo` conflicts with
  `--sandbox` / `--ask-for-approval` (documented in the vendor CLI);
  picking one clears the others in the same exclusivity group instead of
  composing a launch the CLI refuses.

## What follows

`web/shared/domain/cliLaunchPresets.js` holds the per-CLI specs and pure
arg patching (replace generated flag, preserve everything the user typed);
`LaunchFields` renders the controls above an **Advanced** reveal, in all
three editors (defaults, terminal, profile) on both shells. Profiles open
by default once they exist. UI refinement — no ADR.