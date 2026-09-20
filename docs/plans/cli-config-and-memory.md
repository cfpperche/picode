# Agent CLIs: native settings for every CLI, and a Memory pane

Status: **approved by the owner 2026-09-20; slices 1, 2, 4 and 5 implemented on
`feat/cli-config-memory`.** The decision is ADR-0163; the subsystems are
`docs/architecture/cli-settings.md` and `docs/architecture/cli-memory.md`.

Owner's four answers, given as "follow your recommendations": all nine CLIs in
v1; memory read-only shipped together with its writes rather than after them,
because the tier split already removes the destructive risk from the two
generated stores; the Claude Code pane scopes to the workspace (its folder is
keyed on the repository, so a worktree reads the same memory as its checkout);
cross-CLI transfer refused in writing rather than left as a later study.

- **Date:** 2026-09-20
- **Owner request:** evolve the Agent CLIs manager so users configure their
  CLIs from the GUI — expand Launch so the user can apply each CLI's
  available configuration, and add a tab that manages those CLIs' memories.
- **Receipts:** owner screenshots of `#/clis/claude-code` and `#/clis/pi`
  (2026-09-20 10:19); the nine CLIs installed on this machine, probed
  2026-09-20; vendor docs fetched 2026-09-20 (listed per CLI below);
  in-house: ADR-0069, ADR-0070, ADR-0099, ADR-0101, ADR-0103, ADR-0150,
  `docs/benchmarks/2026-09-12-cli-settings-ux.md`,
  `docs/benchmarks/2026-09-17-cli-launch-quick-presets.md`,
  `docs/benchmarks/2026-09-13-ai-memory.md`.

---

## 1. Where the product stands

`#/clis/<cli>` has eight panes. Only two of them are multi-CLI:

| Pane | Coverage today | Source |
|---|---|---|
| Launch | **9 CLIs**; quick controls for 7 | `web/shared/domain/cliLaunchPresets.js` |
| Terminals | 9 | ADR-0069 |
| Sessions | 9 | ADR-0088/0094 |
| Connectors (MCP) | **9, with native file writes** | ADR-0150, `internal/connectors/` |
| Providers | **Pi only** | `cliProviders.js:2` |
| Settings | **Pi only** | `cliSettings.js:3` |
| Keyboard | Pi only | ADR-0101 |
| Packages | Pi only | ADR-0099 |

The screenshots show the asymmetry precisely. On `#/clis/pi` the Launch pane
edits Model and Thinking level and previews the argument vector. On
`#/clis/claude-code` the same pane is a read-only summary behind *Customize*,
and the Settings tab beside it has nothing to render, because
`supportsCliSettings("claude-code")` is false.

So the request is not a new concept. It is **doing for Settings what
ADR-0150 already did for Connectors**, plus one new pane.

## 2. The distinction the plan must not collapse

The owner phrased it as "expand Launch so the user can apply each CLI's
configuration". There are two different mechanisms behind that sentence, and
`docs/benchmarks/2026-09-17-cli-launch-quick-presets.md` explicitly refused
merging them: *"Editing the CLI's own config files in the pane … quick
controls only compose launch arguments."*

| | Launch quick controls | Settings pane (new) |
|---|---|---|
| Writes | the argument array of the next terminal | the CLI's own config file |
| Lives | PiCode's `cli_launch` / profiles | `~/.codex/config.toml`, `~/.hermes/config.yaml`, … |
| Applies to | launches made from PiCode | every launch, including outside PiCode |
| Reverts | Restore defaults | Use inherited / Clear |

Keeping them separate is the honest model, and it is already ADR-0069's
contract. What the plan adds is a **bridge**, not a merge: each side names
the other. A Settings row whose value a saved launch flag overrides says so;
a Launch control says which file holds the default it is overriding. That is
the whole of "apply the CLI's configuration from the GUI" without breaking
a refusal that was made on purpose eight days ago.

## 3. What each CLI actually exposes

Probed on this machine 2026-09-20 (`~/.<cli>`), cross-checked against vendor
docs the same day. Paths marked **verified** exist here.

### Config files (the Settings driver's targets)

| CLI | User file | Project file | Format |
|---|---|---|---|
| Pi | `~/.pi/agent/settings.json` **verified** | `<ws>/.pi/settings.json` | JSON |
| Claude Code | `~/.claude/settings.json` **verified** | `.claude/settings.json`, `.claude/settings.local.json` | JSON |
| Codex | `~/.codex/config.toml` **verified** | none (single file) | TOML |
| Grok | `~/.grok/config.toml` **verified** | `<ws>/.grok/config.toml` | TOML |
| Hermes | `~/.hermes/config.yaml` **verified** | none (single file) | YAML |
| OpenCode | `~/.config/opencode/opencode.json(c)` | `<ws>/opencode.json(c)` | JSON/JSONC |
| Muse Code | `~/.config/muse/settings.json` **verified** | none | JSON |
| Antigravity | `~/.gemini/antigravity-cli/settings.json` **verified** | none | JSON |
| Omp | `~/.omp/agent/config.yml` **verified** | `<ws>/.omp/config.yml` | YAML |

Every one of those paths, and a codec that preserves unknown keys, comments
and `${VAR}` placeholders, **already exists in `internal/connectors/`**
(`codec.go`, `toml.go`, and the per-CLI drivers). The Settings driver reuses
them rather than growing a second parser.

### Native memory (the Memory pane's targets)

| CLI | Store | Shape | Tier |
|---|---|---|---|
| Claude Code | `~/.claude/projects/<slug>/memory/` **verified: 37 projects, 513 files** | `MEMORY.md` index + one topic file per memory, YAML frontmatter `type` / `modified` | **Editable** |
| Hermes | `~/.hermes/memories/` **verified (empty)** | `MEMORY.md` (2200 chars), `USER.md` (1375 chars) | **Editable** |
| Omp | agent dir, backend `local` | `MEMORY.md`, `learned.md`, `skills/<n>/SKILL.md` | **Editable** (backend `off` here) |
| Muse Code | `.agents/memory/` (project) | markdown | **Editable** |
| Grok | `~/.grok/memory-v2/global/` and `/workspaces/<slug>-<hash>/` **verified** | `MEMORY.md` ("Generated by Grok. Do not edit"), `topics/`, `observations/`, `archive/`, `index.sqlite`, `memory_state.sqlite` | **Read + vendor command** |
| Codex | `~/.codex/memories/` **verified: git-backed, 49 KB index** + `~/.codex/memories_1.sqlite` | `MEMORY.md`, `memory_summary.md`, `raw_memories.md` | **Read + toggle** |
| Pi | — | — | **None** |
| OpenCode | — | — | **None** |
| Antigravity | `~/.gemini/…/brain/<conversation-id>/` (transcripts, knowledge items; CLI support unconfirmed) | JSONL | **Unknown — report honestly** |

Their on/off switches are config keys, so the Memory pane's toggle **is** a
Settings write:

- Claude Code — `autoMemoryEnabled`, `autoMemoryDirectory`, env `CLAUDE_CODE_DISABLE_AUTO_MEMORY=1`
- Codex — `[features] memories = true` **verified**, plus `memories.generate_memories`, `memories.use_memories`, `memories.disable_on_external_context`
- Hermes — `memory.memory_enabled`, `memory.user_profile_enabled`, `memory.memory_char_limit`, `memory.user_char_limit`, `memory.write_approval` **verified**
- Omp — `memory.backend` ∈ `off | local | hindsight | mnemopi | sharpshooter`
- Grok — `--experimental-memory` / `GROK_MEMORY=1`; clearing is `grok memory clear --workspace|--global|--all`
- Muse — `runtime_capabilities` toggles the memory-recall observer

## 4. Benchmarks

| Source | Pattern worth taking | Where it lands |
|---|---|---|
| **Ruler** (2.9k★) — one `.ruler/` distributed to 30+ agents | `.bak` before every overwrite; `ruler revert`; a managed block in `.gitignore` | Backup-before-write and a named revert for the Settings driver |
| **rulesync** | **import** from existing native files before generating | First open of a CLI's Settings pane reads, never seeds |
| **Vibe Kanban** | per-agent form with agent-specific fields (`sandbox`, `approval`, `model_reasoning_effort`…), default variant preselected; **composes flags, does not edit config files** | Confirms the Launch/Settings split; its field set matches `cliLaunchPresets.js` |
| **VS Code settings** (already adopted 2026-09-12) | scope is a tab; modified marker per row; reset per row; search filters | The layer switcher generalizes to every CLI |
| **Windsurf / Cursor Memories panel** | list → open → **Edit** → **Delete**; "promote to rules" | The Memory pane's verb set |
| **OpenMemory / mem0 dashboard** | per-app access control, pause an app's access | The per-CLI enable toggle, not a shared store |
| **claude-mem viewer** | delete one observation with confirm; stats refresh | Destructive action shape |
| **Grok `/memory` browser, Codex `/memories`** | the vendors ship a **read-only** browser over a generated index | Grok/Codex tier is read + vendor command, by their own design |
| **agent-memories catalog** | documents exactly these paths per agent | Cross-check for the table in §3 |

### What we refuse

- **A PiCode-owned memory store synced into every CLI.** Already decided:
  `2026-09-13-ai-memory.md` refused a wiki brief as a substitute for a native
  session, and ADR-0150 refused a PiCode-owned MCP registry injected at
  launch, because it breaks every launch made outside PiCode. Same argument,
  same answer. Each CLI's own store stays authoritative.
- **Copying a memory from one CLI into another.** ADR-0088's create-only
  write into a guest store was a narrow, verified exception for sessions.
  Cross-CLI memory transfer is a separate decision and a separate ADR, not a
  button in v1.
- **Hand-editing a generated index.** Grok's `MEMORY.md` says "Generated by
  Grok. Do not edit this file directly" and keeps its truth in
  `index.sqlite`; Codex's directory is git-backed with a sidecar SQLite. For
  those two the pane reads and calls the vendor's own command. Writing them
  would corrupt state PiCode does not own.
- **A generic JSON editor for a CLI without a declared schema** (ADR-0099 §5)
  and **inventing live state for a CLI with no headless signal**
  (ADR-0150 point 4) — Muse and Antigravity report "configured", not "active".
- **A second width or a new page** (ADR-0103): both panes live inside
  `AgentClisFrame`.

## 5. The invariants that are expensive to add later

1. **Two writers on one file.** The CLI writes while PiCode edits.
   ADR-0150 already set the contract — re-read before write, refuse when the
   file changed underneath, atomic tmp+rename. Memory needs one more signal:
   PiCode knows which terminals run which CLI (ADR-0062 presence). Editing a
   memory file while that CLI is mid-turn is the failure mode, so the pane
   names the running terminal and asks before writing.
2. **Scope identity.** Claude Code derives `<project>` from the git
   repository (shared by all worktrees); Grok uses `<name>-<hash>`. Deriving
   the wrong slug silently shows another project's memory. The driver either
   reproduces the vendor's rule exactly or refuses to guess — never a
   best-effort match.
3. **Secrets.** Memory files hold whatever the agent learned. Reads redact
   secret-shaped values the way `internal/mcp` does, and memory content never
   enters feed events, logs or diagnostics.
4. **Delete is not always reversible.** `grok memory clear --all` has no
   undo. Scoped reset only, blast radius named, confirm required
   (ADR-0099 §3).
5. **A file the parser rejects is never silently overwritten** — 409 plus an
   explicit Replace, exactly as ADR-0099 §4 does for `roles.json`.

## 6. Decision table (the rows tests must cover)

| Conditions | Action |
|---|---|
| CLI has one config file, user scope | Edit it; no layer switcher, file named under the header |
| CLI has user + project files, workspace trusted | Both layers; provenance and Use inherited per row |
| CLI has user + project files, workspace untrusted | Project layer blocked with one line + one action (existing pattern) |
| CLI has no declared schema | Pane says so; no generic JSON editor |
| Config file changed on disk since read | Refuse the write, re-read, show the diff |
| Config file unparseable | 409 + explicit Replace; never a silent overwrite |
| Setting also set by a saved launch flag | Row names the override and links to Launch |
| Memory tier editable, CLI idle | List, open, edit, delete with confirm |
| Memory tier editable, CLI mid-turn in a terminal | Name the terminal; confirm before writing |
| Memory tier read-only (Grok, Codex) | Read + vendor command; write actions absent, not disabled |
| Memory absent (Pi, OpenCode) | One line naming that this CLI has no native memory |
| Memory enabled but store empty (Hermes here) | Empty state + the one action (open the setting) |
| Memory scope cannot be resolved for this workspace | Say unknown; never show another project's memory |
| Toggle memory on/off | Writes the CLI's own config key through the Settings driver |

## 7. Slices

| # | Slice | Gate |
|---|---|---|
| 1 | ADR (persistence boundary, extends ADR-0150 / amends ADR-0101's Pi-only registry); `internal/clisettings` driver interface + schema type; Settings for the three single-file CLIs (Codex TOML, Hermes YAML, Muse JSON) reusing the connectors codecs | golden-file codec tests, HOME-swapped suite |
| 2 | Settings for the two-layer CLIs (Grok, Omp, OpenCode) + Antigravity; layer switcher generalized; Claude Code (`settings.json` + project/local) | decision-table rows 1–6 |
| 3 | Launch ↔ Settings bridge: provenance both ways; quick controls extended to Muse and Antigravity once their flags are verified against the installed binaries | row 7; no new persistence |
| 4 | Memory pane, **read-only**, all nine: tiers, list, search, open, empty/blocked/error states | rows 8–14 read paths; visual-review |
| 5 | Memory writes: edit/delete for the editable tier, vendor-command clear for Grok/Codex, enable/disable through the Settings driver | rows 8–10, 15; confirm + blast radius |
| 6 | *(optional, separate decision)* an Agent CLIs-level memory overview across CLIs | not in v1 |

Slices 1–2 are the owner's "configure the CLIs from the GUI". Slice 4 is the
"memory tab" as a read surface, which is where most of its value sits and
where none of the destructive risk does. Slice 5 is the part that deserves
the owner's explicit yes.

## 8. Open questions for the owner

1. **Scope of v1.** All nine CLIs at once (consistent, bigger), or the four
   whose config is a single file plus Claude Code (faster, and the pane stays
   honest about the rest)?
2. **Memory writes.** Ship slice 4 (read-only) alone first, or 4+5 together?
3. **Claude Code's memory is the biggest corpus here** — 513 files across 37
   projects on this machine. Should the pane scope to the current workspace,
   or list every project it finds?
4. **Cross-CLI transfer** (take a Claude Code memory into Hermes) — out of v1
   by default. Worth its own study later, or a refusal to write down now?
