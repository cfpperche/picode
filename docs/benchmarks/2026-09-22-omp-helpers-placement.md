# Study: omp-specific helpers inside a multi-CLI host — where they live, and what they show

- **Date:** 2026-09-22
- **Asked (owner, 2026-09-22):** omp is the Agent CLI that surprises them
  most. They want visual features in PiCode that help people who use PiCode
  *with omp* configure omp well in their projects, and do not know where
  omp-specific work belongs — core, extension, app, plugin. Two screenshots
  framed the ask: omp's **Models** browser (a provider list; kind chips
  `chat · tiny · image · tts · stt · search · judge · embedding · rerank ·
  video`; columns IQ · speed · latency · context · price) and its **Model
  roles** screen (DEFAULT, SMOL, SLOW, VISION, PLAN, COMMIT, TINY, MEMORY,
  TASK, ADVISOR, IMAGE, WEB, SPEECH, DICTATION, JUDGE; `auto → provider/model`
  on unset roles; `+ New role…`, `+ New fallback…`; `loop 1/2/3` beside three
  rows).
- **Sources:** read-only probe of the installed omp **18.2.8** (`omp --help`
  for all 43 subcommands, `omp config list --json`, `omp models --json`,
  `~/.omp/**`, `~/picode/.omp/`, the package's shipped `src/config/*.ts` and
  the 133 docs it embeds in `dist/docs-index.generated.txt`); the omp
  repository docs (`docs/settings.md`, `models.md`, `config-usage.md`,
  `rpc.md`) and npm metadata; vendor docs of some thirty products fetched
  live on 2026-09-22 (§4); PiCode's own decisions — ADR-0036/0109 (apps),
  0069/0160 (CLI runtimes), 0091 (protocol hold), 0099/0119 (package
  config), 0150/0163/0167/0174/0175 (per-CLI drivers), 0154 (picode-mcp) —
  and [../plans/omp-cli.md](../plans/omp-cli.md). Three research passes
  (omp inventory; product config UIs; host placement patterns) with receipts
  inline. **UNVERIFIED** marks what no primary source confirmed.
- **Scope:** the placement decision and a ranked menu of helpers with their
  gates. No plan, no code, no ADR — the two decisions it surfaces are the
  owner's (§8).

## 1. The answer in one paragraph

The question "core, extension, app or plugin" has been answered four times
this month for other omp surfaces — settings (ADR-0163), packages
(ADR-0167), key maps (ADR-0174), custom providers (ADR-0175) — and every
time the same way: **an omp declaration behind a CLI-neutral engine in the
core, rendered by the pane every CLI shares.** omp is named in some thirty
Go and JS files today and none of them is a `switch cli == "omp"` inside a
shared component; that is the shape the strongest platforms call *bundled,
not privileged* (VS Code's TypeScript extension, JetBrains' Java plugin,
Zed's `extensions/`, Obsidian's core plugins, Grafana's Prometheus library —
§4.1). So the default door for omp helpers is **the omp page under Agent
CLIs**, one declaration per helper. A first-party **app** tile is the door
only for a surface that is not "this CLI's page" (the Docker and tmux apps
are the precedent for vendor-named tiles); an **omp-side package** only for
behaviour that must run inside omp's process (PiCode already injects one —
the activity extension — with `-e`); and a **plugin runtime for host UI**
does not exist and is not needed (ADR-0036 refuses third-party JS in the
page). The two screenshots map to two helpers, each needing one widening of
an engine: a **roles matrix with fallback chains** (the settings engine
writes scalars; chains are string lists, which the key-map engine already
writes) and a **models catalog** (fed by `omp models --json`; the IQ column
is not in that JSON). The helper the benchmarks rate highest, and PiCode
lacks for every CLI, is an **effective-config and project doctor** view —
and omp is the one CLI in the roster that publishes its own schema through
its binary (`omp config list --json`: 495 keys with type and description),
which turns "mirror omp's settings" from a transcription job into a runtime
read.

## 2. What omp exposes (inventory, measured on 18.2.8)

omp is a hard fork of pi (`@oh-my-pi/pi-coding-agent`, MIT, ~32.7k stars,
Bun runtime), not a pi package; it reads `~/.omp` and `.omp/`, never `.pi/`
— this repository's `.pi/settings.json` is invisible to it, and its
`pi-browser` extension reaches omp only through the **legacy**
`.omp/settings.json`, which omp still honours. `~/.omp/agent/config.yml` on
this machine carries six keys (`modelRoles.default:
deepseek/deepseek-flash:max`, `symbolPreset`, `composer.shape`,
`theme.dark`, `setupVersion`, `defaultThinkingLevel: auto`).

| Item | Scope | Location | Format | Machine-readable schema | How a host reads / writes it safely |
|---|---|---|---|---|---|
| Settings | global | `~/.omp/agent/config.yml` (`settings.json` auto-migrated once, `.bak` kept) | YAML, nested | internal `SETTINGS_SCHEMA` (`src/config/settings-schema.ts`: 495 keys, 81 enums, 11 tabs); **no JSON schema**; `omp config list --json` → `{path: {value, type, description}}` (enums only in the human listing) | read `omp config list\|get --json`; write `omp config set` (**global only**, parses by schema type) or splice the file (PiCode's engine) |
| Settings | project | `<cwd>/.omp/config.yml` + legacy `<cwd>/.omp/settings.json`; **cwd only, no ancestor walk**; `.claude/settings.json`, `.codex`, `.gemini` project files also merge in | YAML / JSON | same | the CLI never writes project keys (exception: `modelRoles.*` when `modelRoleStorage: project`); **arrays replace** the global array wholesale |
| Overlay | one run | `--config <file>` (repeatable), env `PI_CONFIG_FILES` | YAML | same | the host's best lever: a generated per-launch overlay; a missing file is a hard error |
| Runtime flags | one run | `--model/--smol/--slow/--plan`, `--approval-mode always-ask / write / yolo`, `--thinking off…max / auto`, `--tools a,b`, `--no-extensions/--no-skills/--no-rules`, `--profile`, `--api-key` | argv | — | never persisted |
| Custom providers | global only | `~/.omp/agent/models.yml` (root key `providers:` only; unknown root keys fail) | YAML | internal `ModelsConfigSchema`; rules in `docs/models.md` | PiCode already merges it (ADR-0175); validate by running `omp models --json` |
| Model catalog | global | bundled `pi-catalog/src/models.json` (72 providers, 5 268 models) + cache `~/.omp/agent/models.db` (SQLite, refreshed from models.dev) | JSON / SQLite | — | `omp models --json [--kind …]`, `omp models find`, `omp models refresh` |
| Credentials | global | `~/.omp/agent/agent.db` (SQLite: OAuth + saved keys), provider env vars, `.env` files (`<cwd>`, `~/.omp/agent`, `~/.omp`, `~`), remote `omp auth-broker` | binary / dotenv | — | never open `agent.db` (ADR-0165); `/login` is TUI-only; `omp token`, `omp usage --json` for status |
| MCP | both | `.omp/mcp.json`, `~/.omp/agent/mcp.json` | JSON | **yes**: `src/config/mcp-schema.json` (draft 2020-12) | PiCode already edits both (ADR-0150) |
| Skills | both | `.omp/skills/<n>/SKILL.md` (ancestor walk), `~/.omp/agent/skills/`, foreign `.claude/skills`, `.agents/skills`, `.github/skills`, plugin `skills/` | Markdown + frontmatter (`name`, `description`, `globs`, `alwaysApply`, `hide`) | documented, no schema | files; `skills.ignoredSkills`, `skills.includeSkills`, `disabledExtensions: [skill:<n>]` |
| Extensions / hooks | both | `.omp/extensions/*.ts`, `.omp/hooks/pre\|post/*.ts`, `extensions:` array in config, `-e`, `--hook`, `--plugin-dir` | TS/JS, not sandboxed | API types in the package | PiCode reads the list (ADR-0167 slice 5) and injects `-e` at launch |
| Plugins | both | `~/.omp/plugins/installed_plugins.json`, `<project>/.omp/plugins/`, `~/.omp/marketplaces.json` | JSON v2 | marketplace format (`.omp-plugin/marketplace.json`, Claude's too) | `omp plugin … --scope user\|project --json` (ADR-0167) |
| Subagents | both | `~/.omp/agent/agents/*.md`, `<nearest>/.omp/agents/*.md`; `omp agents unpack [--project] --json` exports the bundled set | Markdown frontmatter: `name`, `description`, `model` (selector, `@role`, or a **list = per-agent fallback chain**), `tools`, `spawns`, `thinking-level`, `advisor`, `prewalk` | `docs/task-agent-discovery.md` | files; `task.disabledAgents`, `task.agentModelOverrides`, `task.maxConcurrency` |
| Rules, commands, prompts, system prompt | both | `.omp/rules/*.md` + sticky `RULES.md`, `AGENTS.md` (walk-up; `CLAUDE.md` too), `commands/`, `prompts/`, `SYSTEM.md`, `APPEND_SYSTEM.md`, `PERSONALITY.md` | Markdown / Handlebars | — | files; `--no-rules`; `disabledProviders: [claude, gemini, …]` switches foreign sources off |
| Key bindings | global | `~/.omp/agent/keybindings.yml` | YAML, action → chord list | action ids in `docs/keybindings.md` | PiCode already writes it (ADR-0174) |
| Themes, symbols | global | `~/.omp/agent/themes/*.json`; `theme.dark`, `theme.light`, `symbolPreset`, `colorBlindMode` | JSON / keys | tokens in `docs/theme.md` | settings keys |
| Secrets redaction | both | `.omp/secrets.yml`, `~/.omp/agent/secrets.yml` | YAML array | documented | file |
| Profiles, roots | env | `--profile` / `OMP_PROFILE` → `~/.omp/profiles/<n>/agent`; `PI_CONFIG_DIR`, `PI_CODING_AGENT_DIR` (reserved by PiCode's launcher), `OMP_SKIP_SETUP=1` | — | — | the isolation recipe for a host |
| Sessions | global | `~/.omp/agent/sessions/<cwd-bucket>/*.jsonl` (schema v3), `history.db`, `plans/` | JSONL | `docs/session.md` | PiCode reads, resumes and writes handoffs (plan Fatias 2 and 6) |

**The model system, precisely.** Fifteen role ids in `src/config/model-roles.ts`
— ten in the chat section (`default`, `smol` "Fast", `slow` "Thinking",
`vision`, `plan` "Architect", `commit`, `tiny`, `memory`, `task` "Subtask",
`advisor`) and five kind roles (`image`, `web`, `speech` = tts, `dictation`
= stt, `judge`), each with an `accepts(model)` filter by kind. Custom roles
are any `modelRoles.<name>`, optionally described by `modelTags.<name>:
{name, color, hidden}`, and subagents reference them as `model: "@review"`.
The selector grammar is `provider/model-id[:off|minimal|low|medium|high|xhigh|max]`,
`@role`, `*` (= `@default`); a role may point at another role. Companions:
`modelRoleStorage global|project`, `cycleOrder` (default `[smol, default,
slow]`), `modelProviderOrder`, `enabledModels` / `enabledProviders` /
`disabledProviders` (arrays that accept **path-scoped** entries), and
`defaultThinkingLevel` (`auto` = a classifier on the tiny/smol model).
**Fallback chains** are `retry.fallbackChains`, a record whose key is a role
name, a `provider/model-id` or `provider/*`, and whose value is an ordered
list of selectors, `@role` aliases or `provider/*` entries (`[]` = never
fall back); resolution is exact model → `provider/*` → the session's role
chain → `default`; kind roles have built-in chains. Beside them sit
`retry.maxRetries` (10), `retry.baseDelayMs`, `retry.fallbackRevertPolicy
cooldown-expiry|never`, `retry.usageAwareFallback`, `retry.usageReservePct`
and `retry.usageReservePolicy confirm|auto|fail-closed`. The screenshot's
`auto → zai/glm-5.3-flash` is what the TUI header says it is — "cleared
roles fall back to auto-selection" — and `loop 1/2/3` appears nowhere in the
docs or source read (**UNVERIFIED**; chains are plain ordered lists, so the
label is the TUI's, to be read out of the overlay source before it is
drawn). The catalog's IQ column is `Model.int`, present for 1 488 of 5 268
bundled models (kimi-k3 59.7, deepseek-v4-pro 36.3 — the scale of
Artificial Analysis' Intelligence Index, but the origin is **UNVERIFIED**);
speed and latency are **local samples** from `agent.db`; `omp models --json`
emits neither `int` nor `tps`.

**Volatility, measured.** 635 npm versions since 2026-01-02 (≈ 2.4 a day;
the last forty span thirty days, ≈ 1.4 a day). 140 changelog lines in 18.x
touch settings keys or migrations; this month alone `task.isolation.mode` →
`task.isolation.enabled` (18.1.5), the `designer` role removed (18.1.5),
`inspect_image.*` → `images.*` (18.1.9), `collapseChangelog` →
`startup.changelogMode` (18.2.1), and 18.2.7 moved image, web, speech,
dictation, judge and memory *into* model roles, auto-migrating six retired
keys. omp rewrites its config on load (migrations, key stripping) and
quarantines invalid YAML to `.broken-*` with a failed start; `omp config
reset` persists the default rather than deleting the key; the default
`tools.approvalMode` is `yolo`; project settings come from the cwd only,
which surprises monorepos.

## 3. What PiCode already does for omp, and the doors it has

| Pane / surface | omp today | Declared in |
|---|---|---|
| Launch | catalog card, PATH wrapper + presence lease, activity extension via `-e` (events measured on 18.2.4/18.2.6), quick controls `--model` and `--add-dir`, the `--trusted-extension` conflict guard | `internal/clilaunch`, `internal/server/cli_plan.go`, `web/shared/domain/cliLaunchPresets.js` |
| Sessions | list, resume, read, write — handoff source and target | `internal/clisession/omp*.go` |
| Providers | vault rows plus custom definitions merged into `models.yml` (ADR-0175); OAuth stays TUI-only | `internal/catalog/modelsyaml.go` |
| Settings | four fields (`modelRoles.default`, `memory.backend`, `symbolPreset`, `theme.dark`), two layers, splice writer, scalars only | `internal/clisettings/specs.go` |
| Keyboard | 70 actions in `keybindings.yml`, pickup **restart** (measured) | `internal/clikeys`, `omp_catalog.go` |
| Memory | editable tier; the folder is probed, not documented (open debt) | `internal/climemory` |
| Packages | plugin verbs through `omp plugin … --json`; the extensions list from `.omp/settings.json` plus `omp config get extensions --json`; `discover` shown as information | `internal/clipkgs/omp_extensions.go` |
| Connectors | `~/.omp/agent/mcp.json` and `.omp/mcp.json`, entry-level toggle | `internal/connectors/omp.go` |
| Dashboard | tokens, cost, duration and ttft from the session JSONL | `internal/climetrics/omp.go` |
| picode-mcp | not injected at launch — the `-e` pi-package path is unverified live; the owner's A/B question is open (ADR-0154 N2) | — |

Five doors exist for the next omp feature, and each has a precedent the
owner approved:

1. **A declaration behind a shared engine (core).** Data per CLI, engine
   per concern: connectors (ADR-0150), settings and memory (ADR-0163),
   packages (ADR-0167), key maps (ADR-0174), custom providers (ADR-0175).
   Every row cites where the vendor fact was read, a CLI without a
   declaration gets one honest line instead of an invented control
   (ADR-0099 §5), and a drift probe (`make keys-drift`) re-reads the
   installed vendor against the catalog.
2. **A first-party app** (ADR-0036/0109): a manifest plus host-rendered
   primitives (list, detail, form, actions) or a native surface compiled
   into the shell; the doors are a closed list (tile, tab, body, icon,
   `host`, `#/app/<id>`). Docker and tmux are vendor-named tiles already.
3. **A package on the CLI side** (ADR-0010/0028/0119): `packages/*` under
   MIT; a `picode.config` descriptor makes its config file a host-rendered
   form; pi-roles paired with its roles editor is the "package + GUI"
   precedent (ADR-0036 §5). omp's activity extension is already a
   package-shaped thing PiCode injects.
4. **The vendor's binary as the engine** (ADR-0167's bounded exception):
   `omp config list|get|set --json`, `omp models --json`, `omp plugin`,
   `omp agents unpack`. omp's own binary answers with types and
   descriptions for every key — no other CLI in the roster does.
5. **Launch injection** (ADR-0069/0154): arguments, environment and files
   per terminal — `--config <overlay.yml>`, `--profile`, `PI_CONFIG_FILES`.

Not doors: a **plugin runtime for host UI** (ADR-0036's refuse table:
in-process third-party JS "can never be walked back"), and **managed omp
over RPC** (ADR-0091's hold; ADR-0160 keeps `Runtime.Start` Pi-only). The
[Zed study](2026-09-22-zed.md) of the same day records that ADR-0091's
trigger has arguably fired (omp ships `omp acp` first-party) — a separate
decision, and one that would not configure roles anyway: omp's RPC has
`get_available_models` and `set_model` but no command for roles or
fallback chains (`docs/rpc.md`); files remain the way.

## 4. Benchmarks

### 4.1 Where mature hosts put vendor-specific configuration UI

| Placement | Who ships the UI | Who ships the schema | Coupling to the vendor's cadence | Failure modes | Examples |
|---|---|---|---|---|---|
| **A. Vendor-declared schema, host-rendered form** | host, one renderer | vendor manifest / flow / JSON Schema | low — a new key is a manifest change | vocabulary ceiling (VS Code drops nested objects to raw JSON; Airbyte bans half of JSON Schema); secrets need their own channel | Home Assistant config and options flows; VS Code `contributes.configuration`; Raycast `preferences`; Gemini CLI extension `settings[]`; **Obsidian 1.13 moved plugins from DOM-built tabs to `getSettingDefinitions()`** so the host renders, searches and validates |
| **B. Driver table in the core** | host | host, transcribed and verified | high — every rename is a host commit, unless pulled from an open catalog (models.dev) or the CLI's own schema | silent staleness (SchemaStore's Claude Code schema "out of date"); wrong defaults (PiCode's Hermes `show_reasoning` incident); tables that grow logic | nvim-lspconfig `lsp/*.lua` (the framework moved into Neovim 0.11, the repo is now data); Homebrew formulae; mise registry; models.dev TOML; `internal/clisettings/specs.go` |
| **C. Plugin-shipped UI mounted by the host** | plugin (React, Swing, webview) | plugin | medium | design drift (Docker mandates its MUI theme); "used sparingly" (VS Code webviews); maintenance nobody owns | Grafana `ConfigEditor`; JetBrains `Configurable`; Headlamp `registerPluginSettings`; Docker Desktop extensions |
| **D. First-party module through the public door** | host team, same API as third parties | host team | high but owned, tested in CI | privileged shortcuts creep in unless the tier is public | VS Code `typescript-language-features`; Zed `extensions/`; IntelliJ `com.intellij.java` (extracted as a plugin in 2019.2); Obsidian core plugins; Grafana core data sources; Home Assistant core integrations |
| **E. Companion app** | companion vendor | reads the CLI's files | very high | abandonment (Crystal deprecated 2026-02; Terragon shut down 2026-01-16); config forks when the companion keeps its own store | Conductor; Claude Desktop (shares `~/.claude`); Amp's web Model Routing page beside its CLI `settings.json` |
| **F. CLI-side package the host only installs** | nobody — no CLI package format contributes host UI (Agent Plugins 1.0 "explicitly leaves … user interface design" to clients) | package author | low for the host | behaviour without a form (oh-my-openagent's fallbacks are hand-written JSONC); "unknown keys are silently ignored" | Claude Code plugins; Codex plugins; Gemini extensions; OpenCode's `config` hook; Pi's `pi` key; omp's `extensions` |

**When one vendor matters more than the others, the answer is "bundled,
not privileged".** VS Code's TypeScript support is a built-in extension on
the public language API and "nothing in the editor core knows TypeScript";
Zed's officially maintained extensions "use the same zed_extension_api
available to all"; JetBrains extracted Java into a plugin other plugins
depend on; Obsidian's Licat: "our internal functionality all use that API
which we eventually stabilized to expose"; Grafana found AWS-specific code
"lived next to generic Prometheus code" and moved it into `@grafana/prometheus`
so vendors could build their own. Home Assistant is the cautionary half:
its favourites (MQTT, ZHA, Z-Wave) do get dedicated panels, and the frontend
maintainers refuse that as a mechanism — "these dedicated panels … need to
be maintained, so we don't want to add them for every single integration"
— while its quality scale (Bronze → Platinum; custom integrations "cannot
claim official tiers") makes first-class a **public tier with named
requirements**, not a hidden code path.

**Surviving a vendor that changes its format** (omp changes it weekly):
versioned entries migrated on read (Home Assistant `VERSION`/`MINOR_VERSION`,
Terraform `SchemaVersion`, Kubernetes served/storage versions); freezing the
version an artefact was written with (n8n keeps "version 1 in that
workflow"; Airbyte publishes `upgradeDeadline`s); preserving unknown keys
(VS Code greys "Unknown Configuration Setting" and keeps the line; PiCode's
splice writer; Kubernetes prunes by default — the cautionary opposite); a
raw escape hatch that never closes (VS Code's "edit in settings.json";
Continue deprecated both `config.json` and `config.ts` and issue #5817 shows
the cost); an effective-config view where layers replace instead of merge;
reading the vendor's own machine-readable schema at runtime (OpenCode
`$schema`, Codex `config.schema.json` generated from `ConfigToml`, Gemini
`settings.schema.json`, omp `config list`) instead of transcribing it;
secrets on a separate channel (Grafana `secureJsonData` is write-only after
save; Gemini `sensitive` → `.env`).

### 4.2 How agent products configure models, roles and fallbacks

| Product | Config lives in | GUI | Per-role models | Fallbacks | Effective view / doctor |
|---|---|---|---|---|---|
| Continue | `config.yaml` + hub blocks | sidebar: a dropdown **per role** | **7 roles**: chat, autocomplete, embed, rerank, edit, apply, summarize | — | schema v1 |
| Zed | `settings.json` | Settings Editor AI page; profile modal | **5 feature slots**: default, inline assistant, commit message, thread summary, compaction (+ subagent) | — | — |
| JetBrains AI / Junie | IDE settings | Models Assignment | 3 slots: Core / Instant helpers / Completion | — | — |
| Aider | `.aider.conf.yml` home → git root → cwd | none | `model`, `weak-model`, `editor-model`, `architect` | — | "Unknown context window … using sane defaults", "Did you mean…" |
| Hermes Agent | `~/.hermes/config.yaml` | `hermes model`, `hermes fallback add/ls/rm`, dashboard Models page | **11 auxiliary slots** (`provider: auto` = main) | `fallback_chain` per slot, per-turn restore, 429/5xx after retries | — |
| OpenCode | `opencode.json` (`$schema`) | TUI + Electron desktop read the same file | `model`, `small_model`, per-agent `model` | — | `opencode debug config` prints the merge |
| Claude Code | settings hierarchy, `$schema` on SchemaStore | `/config`, `/model`, `/mcp`, desktop dropdowns | `model`, `fallbackModel`, subagent `model:` | `fallbackModel` chain "taken whole" from the winning file | `/status` lists files but not which supplied each key; `claude doctor` / `/doctor` |
| Codex | `config.toml`, profiles, trusted `.codex/` | app Developer settings: "in-app controls for common settings, or edit config.toml for advanced options" | model + effort per profile | UNVERIFIED | `/debug-config`: layers in precedence order + policy source |
| VS Code Copilot | `.github/{agents,prompts,skills}`, `mcp.json` | **Agent Customizations editor** (instructions, skills, agents, prompts, MCP, hooks, plugins, models; validation) | per custom agent `model:` **priority list** | the list is the chain | trust dialog, input variables for secrets |
| Cursor | `.cursor/rules`, `mcp.json`, dashboard | Customize page (Rules, MCP, Router) | removed in 2.1 (Auto with Cost/Balance/Intelligence) | Router | — |
| Amp | `settings.json` user/workspace | web Model Routing: drag connection order, mappings, "Check Access", a routing graph | modes | connection order | "Check Access" |
| Vibe Kanban | `profiles.json` | Settings → Agents: **form editor + JSON editor** per executor, `DEFAULT` + variants | per executor variant | — | — |
| OpenRouter | account Routing, presets | catalog with modality chips; rows tokens/week, context, $/M; per-model provider table (throughput, latency, uptime) | preset = models + routing + params | `models[]` + Auto Router | — |
| Requesty | policies | Fallback card: drag to reorder, 0–10 retries each, **live latency/success/speed panel** | — | ordered chain | live panel |
| LiteLLM proxy | YAML + DB (UI overrides file) | Routing Groups, Router Settings → Fallbacks | model groups | fallbacks, context-window fallbacks | none — see §4.3 |
| Portkey | config JSON | JSON playground with lint and version history | targets | fallback / loadbalance / conditional + retry `on_status_codes` | lint |
| models.dev | TOML per provider/model → `api.json` | site table | — | — | fields: cost (input, output, cache read/write), limits, modalities, `reasoning`, `tool_call`, `open_weights`, `status` |
| Artificial Analysis | site | table: Intelligence Index, Coding Index, output speed, latency, price, context; filters open-weights / reasoning / multimodal | — | — | — |

The patterns worth carrying into PiCode, with their receipts:

1. **Effective-config and provenance view** (Codex `/debug-config`,
   Claude `/hooks` tags each hook User/Project/Local/Plugin, OpenCode
   `debug config`, Amp's routing graph). The top support question is "why
   isn't my value applied"; for omp the answer is often "a project array
   replaced the global one" or "a `.claude/settings.json` merged in".
2. **Role-to-model matrix** (Continue, Zed, JetBrains, Aider, Hermes,
   OpenCode): one row per role the adapter declares, `auto` rows showing the
   inherited value greyed. Predictability per task beats a router.
3. **Fallback chain with live health** (Requesty, Amp, Hermes `fallback`,
   OpenRouter `models[]`): an ordered list with drag handles and a test
   button per entry — and Claude's rule that a chain is "taken whole" from
   the winning layer, never merged across layers.
4. **Catalog table with facet chips, hide and pin** (OpenRouter modality
   chips, Artificial Analysis filters, VS Code's Language Models editor with
   hide/pin, OpenCode whitelist, Kilo favourites): omp's own kinds are the
   chips, `enabledModels` is the pin.
5. **Profiles and presets as objects** (Codex profiles, Roo profiles bound
   per mode, Warp profiles, Zed profiles, Vibe Kanban `DEFAULT` + variants,
   OpenRouter `@preset/slug`, Amp modes): "duplicate the default" is the
   create flow.
6. **Scope asked at install, shown side by side** (VS Code "MCP: Add
   Server" → Workspace/Global; Kilo Project/Global with separate remove;
   Claude `--scope`).
7. **Doctor with fix actions and a clean-config test** (`claude doctor`
   read-only vs `/doctor` proposing fixes; Claude's Settings Error dialog;
   `CLAUDE_CONFIG_DIR`; Aider's "Missing these environment variables"; Amp
   "Check Access"; Home Assistant Repairs with `is_fixable`).
8. **Common controls in the app, the rest in the file, with a visible
   seam** (Codex app; Claude `/config` "isn't a view of your settings.json";
   Zed feature models JSON-only): say "N keys in the file are not shown
   here" and offer the file.
9. **Default versus lock as two controls** (Cursor Soft/Hard, Claude
   managed `model` vs `availableModels`, Amp "restrict to workspace
   connections"): for orchestration, "default for new agents" is not
   "allowed set".
10. **Vendor-declared settings prompted from a manifest** (Gemini
    extension `settings[]` with `sensitive` → keychain; VS Code MCP
    `inputs`): ADR-0119's descriptor is this idea; `sensitive` is the field
    it lacks.

### 4.3 Anti-patterns to design against

- **The picker fights the file.** Codex's model picker "briefly changes and
  then returns to the project-configured value" (openai/codex#36163);
  Claude's VS Code extension ignores project `model` (claude-code#90408).
  For omp: a launch flag (`--model`) and a role in `config.yml` are two
  different things, and the pane must say which one a terminal is running.
- **The UI saves what the runtime rejects.** LiteLLM's Admin UI accepts a
  model in two routing groups; startup rejects the config, all groups
  vanish, and the proxy "reports as healthy" (BerriAI/litellm#36310). omp
  quarantines invalid YAML and refuses to start — a write must be validated
  by omp's own reader (`omp config list --json` in the workspace) before it
  is called saved.
- **Scope invisible in lists; installs land in a scope nobody chose**
  (microsoft/vscode#336646, anomalyco/opencode#30933). `omp config set`
  writes global only — a project row must never route through it.
- **Merge semantics that surprise** (claude-code#19487 — a project file
  replaced the global one). omp's arrays replace by design; the pane has
  to draw that, not hide it.
- **Migrations strand users** (continuedev/continue#5817 after the YAML
  move). omp migrated `settings.json` → `config.yml` and `models.json` →
  `models.yml` this year; this repository still runs on the legacy project
  file.
- **The wizard disappears** (Claude removed its `/agents` creation wizard
  in v2.1.198 — "ask Claude or edit `.claude/agents/` directly"). A helper
  that shells out to a vendor wizard inherits that risk; one that writes
  documented files does not.
- **Export leaks secrets** (Roo's exported JSON "includes API keys in
  plaintext"). omp's `models.yml` carries `apiKey` inside the definition;
  ADR-0175 already never serialises it back.
- **Unknown model = guessed metadata** (Aider-AI/aider#3266). A catalog row
  without `int` shows no IQ, not a dash that reads as zero.

## 5. The placement decision, in PiCode's terms

| Option | What it would be | Precedent | Fits the two screenshots? | Cost of being wrong | Verdict |
|---|---|---|---|---|---|
| **A. Declarations in the omp page's panes** | a `roles` field kind in Settings; a Models tab in Providers; richer Launch quick controls | ADR-0163/0167/0174/0175 | directly | low — a declaration is data, its vendor facts pinned by tests | **the default door** |
| **B. A tenth pane, "Project" (CLI-neutral contract)** | what this CLI reads from this workspace: files, foreign sources, a doctor, scaffolds | Copilot's Agent Customizations editor; Home Assistant Repairs and diagnostics; Cursor's Customize page | for skills, agents, rules, hooks, `AGENTS.md` — not the screenshots | medium — a new pane contract for nine CLIs, most answering with one honest line at first (the Memory tiers precedent) | **the second step; one decision** |
| **C. A first-party "Omp" app tile** | a studio outside the CLI page | Docker and tmux tiles; Home Assistant's mqtt/zha panels (counted, capped) | works, but every omp setting would then have two homes | medium — a bespoke panel is a maintenance commitment | only for a surface with no pane home; none identified |
| **D. An omp-side package in `packages/`** | behaviour inside omp: publish the live role and model; apply a preset | pi-roles + the `-e` activity extension; OpenCode's `config` hook | a live chip, not a form | low–medium — tracks omp's daily releases | for behaviour only |
| **E. The vendor binary as schema and catalog** | `omp config list --json` as the settings schema; `omp models --json` as the catalog | ADR-0167; Codex `config.schema.json`; Gemini `settings.schema.json` | yes — it feeds A and B | low — a drift test measures the CLI | **use it; a generated form amends ADR-0163** |
| **F. A plugin runtime for host UI** | third-party JS or webviews in the page | ADR-0036 refuse table; VS Code "used sparingly" | not needed | high | refused, unchanged |
| **G. A companion app** | a separate product | Conductor; Crystal (dead); Amp's web routing | no | very high | refused |
| **H. Managed omp over RPC** | omp as a second managed backend | the 2026-08-25 dossier; ADR-0091/0160; the Zed study | no — RPC sets models, not roles or chains | large | horizon, its own decision |

Eight rules follow, each borrowed from a platform that learned it the hard
way (§4.1):

1. **Vendor knowledge is data, host behaviour is code.** One declaration
   per CLI, keys in omp's own names, every key citing where it was read at
   the installed version — the rule `specs.go`, `omp_catalog.go` and
   `modelsyaml.go` already follow.
2. **The favourite walks through the public door.** If the settings engine
   cannot express a roles matrix, widen the engine for all nine CLIs; never
   bypass it for one (Grafana's AWS-inside-Prometheus is the smell).
3. **Schema from the vendor, form from the host.** omp publishes its
   schema through its binary; read it there, pin a version, add an
   `omp-drift` probe beside `keys-drift`. Hand transcription rots
   (SchemaStore's Claude schema).
4. **Never destroy what you do not understand.** Splice writes, unknown
   keys and comments survive, refusal beats guessing, and the raw file
   stays one click away in the Files view — never a JSON box in the pane
   (ADR-0099 §5, the [settings UX study](2026-09-12-cli-settings-ux.md)).
5. **Validate with the vendor's reader before calling a write saved.**
   omp refuses to start on a bad file; `omp config list --json` in the
   workspace is the cheapest proof a write is loadable.
6. **Show provenance and the effective value on every row.** "Set here /
   From Global / omp default / From `.claude/settings.json` / From this
   launch's overlay" — omp arrays replace, so a merged guess is a lie.
7. **Behaviour ships as a package; forms ship in the host.** A live chip or
   a preset that needs omp's process rides the `-e` extension; a form a
   terminal-averse user fills lives in the pane.
8. **Tiers, not exceptions.** Publish what "first-class" means per CLI so
   omp's extra helpers are a visible tier Hermes or Muse can reach by
   meeting the same bar — the Memory pane's four tiers are the template.

## 6. The menu of helpers, ranked

Cost: **S** one declaration or one quick control; **M** a new field kind,
tab or command reader with tests; **L** a new pane or an ADR.

| # | Helper | What the user sees | Door | Data (read / write) | Adapts | Cost | Gate before building |
|---|---|---|---|---|---|---|---|
| 1 | **Roles matrix** | one row per role (15 built-in + custom, with `modelTags` colour), selector picker fed by the catalog, thinking suffix, `auto` rows greyed with the resolved value, per-layer provenance | A: Settings, new kind `roles` | `modelRoles.*` (scalars, splice); `modelTags.*`; `cycleOrder`, `defaultThinkingLevel` | Continue, Zed, Hermes; pi-roles editor (`PackagesConfig.jsx`) | M | how a running omp picks up `config.yml` edits (restart vs reload); what `modelRoleStorage` changes for a reader |
| 2 | **Fallback chains** | ordered list per role, model or `provider/*`; drag, add `@role`, `[]` = never; the retry knobs as one group with danger notes (`usageReservePolicy: auto`) | A: same kind | `retry.fallbackChains.*` (string lists — the key-map engine's `listLiteral` precedent), `retry.*` scalars | Requesty, Amp, Hermes `fallback`, Claude "taken whole" | M (with 1) | what the TUI's `loop N` label means; no merge across layers |
| 3 | **Models catalog tab** | kind chips, search, availability (keyed provider), context, cost pair, `enabledModels` / `disabledProviders` as pin and hide, the role badge at the point of use | A: Providers, a Models tab (the [providers v2 study](2026-09-03-providers-view-v2.md) Tab 2, never built for pi either) | `omp models --json --kind all` run in the workspace (path-scoped arrays differ per cwd); IQ only if read from the bundled `models.json` | OpenRouter, VS Code Language Models editor, Roo `ModelInfoView` | M | origin and licence of `int`; whether `omp models` touches files (hash before/after) |
| 4 | **Launch quick controls** | `--approval-mode` (default `yolo` carries its cost line), `--thinking`, `--smol` / `--slow` / `--plan`, `--config <overlay>`, `--profile`, `--no-extensions` / `--no-skills` / `--no-rules` | A: Launch (`cliLaunchPresets.js`) | flags verified from `--help` on 18.2.8 | Codex profiles, Warp profiles; PiCode's own "Launch and Settings stay separate" | S | `--flag --version` parse check per flag, the existing ritual |
| 5 | **Effective config drawer** | "Resolved for this workspace": every key with its source layer, foreign files that merged in, arrays that replaced, keys the pane does not show ("N keys in the file are not shown here") | A: Settings (the provenance line exists) | `omp config list --json` in the workspace vs the two files | Codex `/debug-config`, Claude `/hooks` sources, OpenCode `debug config` | S–M | that `config list` performs no migration write |
| 6 | **Project doctor** | one line + one action per finding: legacy `.omp/settings.json` in use; `.pi/` present but unread; approval mode unset in a shared repo (= `yolo`); `.env` tracked by git; a key in `config.yml` that omp retired (18.2.7 list); a project array that replaces a global one; cwd-only settings in a monorepo | B: Project pane (or A: a card on Settings until B exists) | file scan + `omp config list --json` | Home Assistant Repairs, `claude doctor` vs `/doctor`, Aider warnings | M (L with B) | the checks are measured against a real repo, this one first |
| 7 | **Project pane** | what omp reads from this workspace, grouped: settings, MCP, skills, agents, rules, `AGENTS.md`, hooks, extensions, secrets, plugins; foreign sources (`.claude/`, `.cursor/`, `.codex/`, `.gemini/`) with their switch; scaffold actions (`config.yml` from a preset, `omp agents unpack --project`) | B, one decision: a CLI-neutral pane whose omp declaration is the richest and whose Hermes row is one sentence | files; `disabledProviders`, `skills.*`, `disabledExtensions` | Copilot Agent Customizations, Cursor Customize, Kilo scope picker | L (ADR: it writes files into the user's repo) | owner's yes on the pane; tier per CLI |
| 8 | **Subagents editor** | list of `.omp/agents/*.md` and the user-level set; frontmatter as a form (`model` selector or `@role` or a list = its own chain, `tools`, `spawns`, `thinking-level`), body as markdown | B (inside 7) | files, `task.*` keys | Copilot `.agent.md`, Claude `/agents` (the wizard that vanished), Roo modes | M | frontmatter fields re-read at the pinned version |
| 9 | **Live role and model chip on the terminal card** | "deepseek-flash · DEFAULT · loop 2" while a session runs | D: the injected `-e` activity extension publishes it | extension events (`model_change` exists in the transcript; as an event, **UNVERIFIED**) | pi's `RoleChip`, Cursor's routed-model display | M | the event exists and carries the role |
| 10 | **Presets** | "frugal", "quality", "local-first": a role set + chains written to `.omp/config.yml`, applied by name, diffed before write | B action (or A: an Apply preset menu on 1) | YAML fragments shipped in the binary | oh-my-openagent's opinionated agents; Codex profiles; OpenRouter presets | S after 1 | the owner's opinions, not the agent's |
| 11 | **Per-role spend** | cost and turns per role on the omp dashboard tile | Dashboard | session JSONL carries the model per turn; the role, **UNVERIFIED** | Claude `/usage` by subagent | M | the transcript names the role |
| 12 | **Managed omp** | composer, model chip, permission cards for omp sessions | H | `omp --mode rpc` or `omp acp` | the Zed study, the 2026-08-25 dossier | L + ADR | ADR-0091 re-measure; not a config helper |

**Why 1–3 first.** They are the two screenshots, and they are pure door A:
a field kind the engine renders for any CLI that declares roles (Hermes has
eleven auxiliary slots and per-slot chains; OpenCode has `model` and
`small_model`; Claude has `fallbackModel`), so the widening pays for three
CLIs, not one. **Why 5 and 6 before 7.** The doctor's most valuable lines
are known today from this repository alone (legacy `settings.json`, unread
`.pi/`, `yolo` by default), they need no new pane, and Home Assistant's
lesson is that diagnosing beats configuring. **Why 7 is one decision.** A
pane that scaffolds files into a repository crosses the persistence
boundary and needs an ADR; its contract must be neutral (every CLI has a
project directory — `.claude/`, `.codex/`, `.gemini/`, `opencode.json` +
`.opencode/`, `.grok/`, `.agents/`, `.pi/`, `.omp/`) so omp is the richest
row, not the only one.

## 7. What PiCode refuses

| Temptation | Why not |
|---|---|
| A generic YAML or JSON editor in the pane | ADR-0099 §5 and the settings UX study: two writers already; the raw file is the Files view |
| Mirroring omp's 495-key TUI settings by hand | ≈ 1.4 releases a day and monthly renames; a transcribed schema rots (SchemaStore's Claude schema). Read the vendor's listing instead |
| Opening omp's SQLite files | `agent.db` holds credentials (ADR-0165); `models.db` is a cache the CLI answers for with `omp models --json` |
| A PiCode-owned copy of roles or chains | ADR-0129/0150: files stay the only source of truth; a second store forks the user's config (the companion-app failure) |
| An "Omp" tile for what is a pane | Home Assistant's cap on dedicated panels; two homes for one setting |
| Routing `omp config set` for project rows | it writes the global file only, and `reset` persists a default instead of deleting |
| Proxying model traffic to implement fallbacks in PiCode | omp already has chains; ADR-0003 keeps PiCode out of the token path |

## 8. Open questions for the owner

1. **Schema source.** Keep curated declarations (today's four omp fields,
   each hand-verified), or generate the omp form from the installed
   binary's own listing (`omp config list --json`, enums from the human
   listing, pinned by an `omp-drift` probe)? The second is an ADR-0163
   amendment: the declaration is still PiCode's, but its rows come from
   the vendor at runtime.
2. **The Project pane.** A tenth pane on every CLI page with a neutral
   contract (omp the richest row), or the doctor as a card on Settings
   until the demand is measured?
3. **The IQ column.** Read `int` from the bundled catalog in the installed
   package (origin unverified — plausibly Artificial Analysis) or ship the
   Models tab without it until omp exposes it in `--json`?
4. **Order.** Roles and chains first (the screenshots), or the doctor
   first (the cheapest lines are known today)?
5. **The live chip** changes the injected activity extension — is the
   package door acceptable for a read-only signal?
6. **Managed omp** stays a separate decision (ADR-0091, re-measured by the
   Zed study); nothing in this menu depends on it.

## 9. Receipts

omp: https://github.com/can1357/oh-my-pi (docs `settings.md`, `models.md`,
`config-usage.md`, `rpc.md`, `task-agent-discovery.md`,
`extension-loading.md`); npm `@oh-my-pi/pi-coding-agent` (635 versions);
`src/config/settings-schema.ts`, `model-roles.ts`, `mcp-schema.json`,
`src/session/retry-fallback-chains.ts`, `pi-catalog/src/{types.ts,models.json}`,
`pi-tui/src/overlays/model-browser.ts`, `CHANGELOG.md` — all read from the
installed 18.2.8 on 2026-09-22; `~/.omp/agent/config.yml`,
`~/picode/.omp/settings.json`.
Placement: https://code.visualstudio.com/api/references/contribution-points ·
https://code.visualstudio.com/api/extension-guides/webview ·
https://developers.home-assistant.io/docs/core/integration/config_flow/ ·
https://developers.home-assistant.io/docs/core/platform/repairs ·
https://developers.home-assistant.io/docs/core/integration-quality-scale/ ·
https://github.com/home-assistant/frontend/discussions/7820 ·
https://docs.airbyte.com/platform/connector-development/connector-specification-reference ·
https://docs.n8n.io/connect/create-nodes/build-your-node/reference/versioning ·
https://backstage.io/docs/conf/defining/ ·
https://grafana.com/blog/grafanas-prometheus-libraries-how-we-built-libraries-to-create-a-truly-vendor-neutral-data-source/ ·
https://github.com/neovim/nvim-lspconfig · https://mise.jdx.dev/registry.html ·
https://github.com/anomalyco/models.dev ·
https://docs.obsidian.md/plugins/guides/migrate-declarative-settings ·
https://robhaisfield.com/notes/building-community-in-obsidian-with-licat ·
https://developers.raycast.com/information/manifest ·
https://plugins.jetbrains.com/docs/intellij/plugin-compatibility.html ·
https://github.com/zed-industries/zed/tree/main/extensions ·
https://docs.docker.com/extensions/extensions-sdk/design/design-guidelines/ ·
https://headlamp.dev/docs/latest/development/api/plugin/registry/functions/registerpluginsettings/ ·
https://github.com/agentplugins/agent-plugins-spec · https://github.com/stravu/crystal ·
https://github.com/terragon-labs/terragon-oss · https://github.com/code-yeongyu/oh-my-opencode.
Products: https://docs.continue.dev/customize/model-roles/00-intro ·
https://zed.dev/docs/ai/agent-settings ·
https://www.jetbrains.com/help/ai-assistant/use-custom-models.html ·
https://aider.chat/docs/config/aider_conf.html ·
https://hermes-agent.nousresearch.com/docs/user-guide/features/fallback-providers ·
https://opencode.ai/docs/config/ · https://code.claude.com/docs/en/settings ·
https://code.claude.com/docs/en/debug-your-config · https://code.claude.com/docs/en/sub-agents ·
https://learn.chatgpt.com/docs/config-file/config-reference ·
https://learn.chatgpt.com/docs/developer-settings ·
https://code.visualstudio.com/docs/agent-customization/overview ·
https://code.visualstudio.com/docs/copilot/customization/language-models ·
https://cursor.com/docs/cursor-router · https://ampcode.com/docs/customize/model-routing ·
https://vibekanban.com/docs/configuration-customisation/agent-configurations ·
https://openrouter.ai/docs/guides/routing/model-fallbacks ·
https://docs.requesty.ai/features/fallback-policies ·
https://docs.litellm.ai/docs/proxy/reliability ·
https://portkey.ai/docs/product/ai-gateway/configs ·
https://artificialanalysis.ai/models · https://geminicli.com/docs/extensions/reference/ ·
https://kilo.ai/docs/customize/marketplace ·
https://roocodeinc.github.io/Roo-Code/features/settings-management.
Anti-patterns: openai/codex#36163 · anthropics/claude-code#90408 · #19487 ·
#66474 · BerriAI/litellm#36310 · #25261 · microsoft/vscode#336646 · #332072 ·
anomalyco/opencode#30933 · continuedev/continue#5817 · #5545 ·
Aider-AI/aider#3266 · cline/cline#4815 · SchemaStore/schemastore#5484.

## 10. Recommendation (owner asked, 2026-09-22: native omp features only)

Everything below writes only omp's documented files and keys, runs only
omp's documented commands, and puts nothing new inside omp's process. The
one thing asked *of* omp is upstream, not built here: `int` and `tps` in
`omp models --json`.

| Slice | What ships | omp-native mechanism | Door | Cost |
|---|---|---|---|---|
| 1. **Model roles** in `#/clis/omp/settings` | a `roles` field kind: one row per role id read from the installed bundle (`model-roles.ts`, the `omp_catalog.go` precedent), selector picker fed by `omp models --json`, `@role` and `*` accepted, unset rows greyed "auto"; a chain editor per role, model or `provider/*`; `modelRoleStorage` as a switch ("where the TUI saves a picked role"); `cycleOrder`, `defaultThinkingLevel` and the `retry.*` knobs as declared rows | `modelRoles.*` scalars, `retry.fallbackChains.*` string lists (the key-map engine's `listLiteral`), both layers — the project layer is the one `omp config set` cannot write | A | M |
| 2. **Models tab** in `#/clis/omp/providers` | kind chips, search, keyed-provider availability, context, cost pair; pin = `enabledModels`, hide = `disabledProviders`; no IQ column until the JSON carries it | `omp models --json --kind all`, run in the workspace | A | M |
| 3. **Launch quick controls** | `--approval-mode` (with the `yolo` cost line), `--thinking`, `--smol` / `--slow` / `--plan`, `--profile`, `--config <file>` for a user-supplied overlay | flags verified on 18.2.8 | A | S |
| 4. **Resolved config + doctor card** on Settings | every key with its source; foreign files that merged; arrays that replaced; "N keys not shown here"; the findings known from this repository: legacy `.omp/settings.json`, unread `.pi/`, approval mode unset, `.env` tracked, retired 18.2.7 keys | `omp config list --json` in the workspace + a file scan; read-only | A | S–M |
| 5. **Project pane** | agents, skills, rules, hooks, `AGENTS.md`, scaffold via `omp agents unpack --project` | files omp reads natively | B — after the owner's yes and an ADR | L |

**On the two open questions.** Schema source: a hybrid that needs no
ADR-0163 amendment now — *curated write, generated read*. The pane renders
as controls only what a declaration names and tests pin (roles, chains,
the settings rows); the resolved-config drawer shows all 495 keys straight
from `omp config list --json` without transcribing one. Project pane: not
yet — slice 4's doctor card on Settings first, a pane when its findings
outgrow a card.

**Gates before slice 1** (measured, the repo's ritual): how a running omp
picks up a `config.yml` edit (restart or live — the pane's pickup sentence
depends on it); what the TUI's `loop N` label means; whether `omp config
list --json` enumerates `modelRoles.<role>` and what it does with an
unknown key or an invalid selector (warn or quarantine); what
`modelRoleStorage` changes for a *reader*; whether `omp models --json`
differs per cwd (path-scoped arrays). A drift probe (`omp-drift`, beside
`keys-drift`) re-reads the role ids and retry keys from the installed
bundle so a rename lands as a red test, not a wrong form.

**Out, under this constraint:** a package or extension that adds behaviour
to omp; a PiCode-owned overlay store; reading `int` from the bundled
catalog file; managed omp over RPC (ADR-0091, its own decision).

## 11. The five measurements (2026-09-22, omp 18.2.8)

§10 named five gates before slice 1. All five were measured the same day and
all five are answered. Method: an **isolated agent dir**
(`PI_CODING_AGENT_DIR=<scratch>`, confirmed with `omp config path` before the
first write) plus two fixture projects; the live half ran in a tmux server of
its own (`TMUX_TMPDIR=<short dir>` + `-L`, killed by exact session name). The
owner's `~/.omp` received no write from this probe — every file this session
wrote is under the scratch dir. Source line numbers are from the installed
bundle's shipped TypeScript.

| # | Gate | Answer |
|---|---|---|
| M1 | how a running omp picks up a `config.yml` edit | **restart**, measured live |
| M2 | what `loop N` means | the role's place in the **ctrl+p quick-switch cycle** (`cycleOrder`), not a fallback |
| M3 | what `omp config list --json` enumerates, and what it does with bad input | `modelRoles` is **one `record` key**, not one key per role; an invalid selector passes silently; **invalid YAML is quarantined and the file is moved**, exit code still 0 |
| M4 | what `modelRoleStorage` changes for a reader | **nothing** — it is writer-side, and TUI-only |
| M5 | whether `omp models --json` differs per cwd | **yes**, decisively; so does `config list --json` |

### M1 — the pickup is a restart

An omp TUI ran in the isolated dir with `modelRoles.default:
openai/gpt-4o-mini`; its status line read `[M] GPT-4o mini … 128K`. With it
running, the config was rewritten to `openai/gpt-5-nano`, `composer.shape:
minimal`, `symbolPreset: unicode`. Nothing changed — not passively after four
seconds, not after two window resizes, not after a keystroke. The same binary
restarted in the same directory came up `[high] GPT-5 Nano … 400K`, which
proves the edit was valid and only the restart applied it.

The source agrees and sharpens it: there is **no file watcher** in
`src/config/` (no `fs.watch`, `watchFile` or `chokidar`), and
`Settings.reloadFromDisk()` has exactly **one** caller in the whole tree —
`src/task/structured-subagent.ts:266`, before a structured subagent runs. So
a long session does re-read at that one moment, which is not something a pane
can promise. The pane's sentence is the one the key-map pane already uses:
**restart it**.

One thing the source gives for free and the probe did not exercise: a write
by omp re-reads the file under a YAML write lock and merges only the keys
*that process* changed (`#saveNow` at `settings.ts:3036`,
`#reloadPersistedLayers` at `:907`). So a PiCode write is not clobbered by a
running omp's later save. Read from source, not run — worth a live test the
day a pane writes under a live TUI.

### M2 — `loop N` is the quick-switch cycle

`pi-tui/src/overlays/model-hub.ts:2181` computes `cycleOrder.indexOf(role)`
and line 2182 draws `⟳ ${cycleIndex + 1}`, under the comment *"Quick-cycle
membership badge (`⟳ 2` = second stop of the ctrl+p cycle)"*. `cycleOrder`
defaults to `[smol, default, slow]`, which is exactly the screenshot's SMOL 1,
DEFAULT 2, SLOW 3. The hub edits it in place — toggle membership (appended at
the end), move a role one slot earlier or later — and persists through
`settings.set("cycleOrder", order)` (`selector-controller.ts:1261`), with the
status line *"Quick-switch cycle: smol → default → slow"*.

For PiCode this is a **separate control from fallbacks**: an ordering over a
subset of roles, bound to `app.model.cycleForward`. Drawing it as a fallback
chain would be a lie. The roles matrix should carry a membership toggle and an
order, and the Keyboard pane already owns the chord that drives it.

### M3 — what the binary will and will not tell a host

`omp config list --json` answers 495 keys as `{path: {value, type,
description}}`. Three shapes matter:

```
modelRoles           => type "record", value {"default":"…","smol":"…"}
retry.fallbackChains => type "record", description carries the whole grammar
modelRoleStorage     => type "enum",   description "Where model selector role
                                        assignments are saved"
```

`modelRoles` is **one key**, not one key per role — `omp config get
modelRoles --json` returns the whole record and `omp config get
modelRoles.default` answers *"Unknown setting: modelRoles.default"*. A host
reads the record and addresses a role inside it; PiCode's own engine already
does that, because it splices the file rather than calling the CLI.

What the binary does **not** do is validate: `modelRoles.default:
nosuchprovider/nosuchmodel:bogusthinking` came back through `config list
--json` verbatim, no warning, exit 0. An unknown top-level key
(`totallyUnknownKey: 42`) survived in the file untouched and was refused by
name on `get`. A valid file was **not** rewritten by any read (md5 identical
before and after), which is what a resolved-config drawer needs.

The finding that changes a rule: **a config omp cannot parse is destroyed, not
preserved.** A file with broken YAML was *moved* to
`config.yml.broken-1790098763721-2491413-<uuid>` with this on stderr —

```
error: Settings config is invalid: …/config.yml (moved to …/config.yml.broken-…):
SyntaxError: YAML Parse error: Unexpected character
```

— and the process still exited **0**. It happened on a read-only `config
list`. PiCode's engine already refuses to write a document that does not
parse, so PiCode cannot cause this; what follows is for the doctor: a
`*.broken-*` beside a CLI's config is a finding with a date and a one-click
"open it", and a host must never read omp's exit code as the verdict.

### M4 — `modelRoleStorage` is writer-side, and TUI-only

For a reader the layer order is fixed, and omp names it itself
(`settings.ts:1373`):

```
getModelRoleProvenance(role) -> "runtime" | "overlay" | "project" | "global" | "default"
getModelRoleSource(role)     -> "project" | "global" | "default"
```

`modelRoleStorage` never enters that walk. It decides where the **model hub**
saves a role the user picks: `targetScope = configuredStorage === "project" ?
(scope ?? "project") : "global"` (`selector-controller.ts:1072`). Measured:
with `modelRoleStorage: project` set globally, `omp config set modelRoles
'{"default":"openai/gpt-5-nano"}'` run inside a project still wrote the
**global** file; the project file was untouched, and `--scope` is not an
option (*"Unknown option '--scope'"*). So the project layer is reachable only
by writing the file — which is exactly the gap PiCode's pane fills.

Two gifts here. The pane's provenance line should use **omp's own five
words**, and the row for `modelRoleStorage` should read as what it is: *where
the omp TUI saves a model you pick in its hub*, not a scoping rule.

### M5 — the workspace decides the answer

Same isolated agent dir, same dummy credential, two fixture projects that
differ only in their `.omp/config.yml`:

| Run from | project config | `omp models --json` | `config list --json` → `disabledProviders` |
|---|---|---|---|
| `projA` | `symbolPreset: ascii` | **55 models**, provider `openai` | `[]` |
| `projB` | `disabledProviders: [openai]` | **0 models** | `["openai"]` |

Every omp read a pane makes — the catalog, the resolved config, a role's
effective value — must run **in the workspace directory**, or it answers about
somewhere else. (The empty list in `projB` is the project's own switch: with no
credential at all the same command also returns 0, so availability and
disablement look identical from outside. A pane must say which one it is.)

### Four findings the gates did not ask for

1. **`omp config set` prints the effective value, not what it wrote.** Run
   inside a project whose layer sets `symbolPreset: ascii`, `omp config set
   symbolPreset nerd` wrote `nerd` to the **global** file and printed `[ok]
   Set symbolPreset = ascii`. The write happened and had no effect, and the
   confirmation says so only if you know to read it that way. A pane that
   shells out to `config set` would report success for a change nobody will
   see.
2. **omp's own writer drops comments.** A file carrying `# keep me` and a
   trailing `# trailing note` lost both after one `omp config set`. PiCode's
   splice writer preserves comments; omp will remove them on its next save
   anyway, so the pane should not sell comment survival as a promise about
   the file's future.
3. **Flat and nested forms of one key coexist, and PiCode sees only the
   nested one.** With `composer.shape: band` written flat, `omp config set
   composer.shape minimal` **added** `composer:\n  shape: minimal` and left
   the flat line in place — the key now exists twice, and the nested one
   wins. PiCode's reader (`lookup`, `internal/clisettings/doc.go:79`) walks a
   decoded map segment by segment, and its writer (`yamlWalk`, `:377`)
   matches only nested blocks, so a file in the flat form reads as **not
   set** for every declared key and a save adds the same duplicate. The
   owner's real config is nested today, so nothing is broken now; the doctor
   should flag a key present in both forms, and the omp declaration deserves
   a fixture in the flat form.
4. **One config file, many live omp processes.** While this probe ran, five
   omp TUIs launched by PiCode were live on this machine (worktree terminals
   for auth-byok, delivery, custom, browser, keyboard). PiCode gives each its
   own `--session-dir` but they **share `~/.omp/agent`**, so they share one
   `config.yml` and one `agent.db`. At 14:41:55 one of them saved a new
   default model into that shared file. With M1's answer, the consequence is
   plain: a role changed in one terminal is invisible to the other four until
   each restarts, and the last writer wins the file. The pane's restart
   sentence is therefore about *every* open omp terminal, not just the one in
   front of the user — and the doctor can say how many are running.

### What this changes in §10

Nothing in the slice order, and three sentences in the panes: the roles matrix
gains a **cycle membership and order** control that is not a fallback (M2);
the `modelRoleStorage` row is labelled as the TUI's save target, not a scope
(M4); and every omp read runs **in the workspace** (M5). The doctor gains
three findings that need no new mechanism: a quarantined `*.broken-*` file
(M3), a key written in both the flat and the nested form (finding 3), and
`N omp terminals are running; a change here reaches them when each restarts`
(M1 + finding 4).
