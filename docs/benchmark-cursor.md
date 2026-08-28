# Benchmark: Cursor — product & aesthetic north star

> Status: v1 — 2026-08-23. Enforced by `/skill:uiux-review` (Cursor bar section).
> Cursor sets the bar for agent-era developer tools. We hold ourselves to it —
> adapted to our center of gravity, which is **agents, not files**.

## Why Cursor

Cursor defined what developers now expect from agent tooling: instant
feedback, dense dark surfaces, trustworthy agent actions (diffs,
checkpoints, permissions), and polish that makes a power tool feel light.
PiCode serves a different mission — orchestrating Pi agents across
workspaces — but the same audience expectation bar applies.

## The adaptation rule

Cursor is an IDE with agents; PiCode is an ADE for agents. **Borrow
patterns, never the mission.** One-sentence test for every feature idea
taken from Cursor:

> "Does this make agent *control* better — or does it make us a worse Cursor?"

If the latter, reject it. We are not building an IDE. A file in the agent cwd (ADR-0015) is agent control; LSP is not.

## Product patterns we adopt

| # | Cursor pattern | PiCode adaptation | Milestone |
|---|---|---|---|
| 1 | Agent activity feed | Collapsible rows per tool call (`read`/`edit`/`bash`) with status, duration, expandable output — never walls of text | M2 |
| 2 | Checkpoints | Pi's session tree (branching via `id`/`parentId`) surfaced as checkpoints: every turn is a restore point; scrub and branch from any point | M2 (view), M3 (branch) |
| 3 | Diff review | Inline diffs for agent edits with per-hunk Keep/Undo; file pane in the cwd (ADR-0015). Cross-agent view later | Track E |
| 4 | Per-chat model picker | Provider/model/thinking selector per agent, reachable in ≤2 clicks from any agent view | M2–M3 |
| 5 | Background agents | Agents as first-class **cards with status** (running / idle / blocked / waiting-input), not chat tabs | M1 (cards) → M4 (fleet) |
| 6 | @-mentions | Prompt composer: `@file`, `@agent`, `@session`, `@skill` to inject context into the prompt | M3 |
| 7 | Rules management | GUI over Pi primitives: `AGENTS.md`, `.pi/` skills, extensions, per workspace — inspectable files, not a proprietary format | M3–M4 |
| 8 | Usage visibility | Per-agent token/cost telemetry where providers expose it | M4 |
| 9 | Command palette | `Ctrl+K` global: switch agent, send task, toggle panels, new workspace | M2 |
| 10 | Streaming markdown | Agent output renders streaming markdown with syntax-highlighted code | M2 |

## Aesthetics & density bar

### Design tokens (v2 — full spec in [design/benchmark-visual-anatomy.md](design/benchmark-visual-anatomy.md))

The M2 redesign implements the anatomy: conversation as centered hero
(~760px column), tool pills inside the conversation, rounded composer with
kind chip, terminal as a bottom dock. Tokens: bg `#0e0e11`, panel `#131317`,
elevated `#1b1b21`, border `#232329`, accent `#7c8cf8`, UI 13px, pill 28px,
radius 8/12.

```
Surface
  --bg-base #0d0f12 · --bg-panel #15181d · --bg-elevated #1b1f26
  --border #23272f (1px, subtle — depth comes from surface steps, not shadows)
Text
  --text-primary #e7eaf0 · --text-secondary #8b93a1
Accent & status
  --accent #7aa2f7 (focus, links, primary actions)
  --ok #9ece6a · --warn #e0af68 · --danger #f7768e
Type
  UI: sans 13px / 1.5 (Inter-class) · headings 600 weight, tight tracking
  Code/ids/paths: mono 12.5px (ui-monospace stack)
Shape & space
  4px spacing grid · radius 6px controls / 10px panels
Motion
  100–200ms ease-out · skeletons over spinners · streaming never blocks input
```

### Density rules (the Cursor tell)

- **13px base UI font.** Power-tool density; never marketing-airy spacing.
- Activity/tool-call rows **24–32px** collapsed; expand on demand.
- Chat: line-height 1.5, 8px paragraph gap, tight code blocks.
- Layout: left nav + central work area + right inspector — all resizable
  and collapsible; panels remember state.
- Long outputs (logs, sessions) are **virtualized lists**.
- Iconography: 16px thin line icons (lucide-class), `currentColor` only.
- Status is compact and truthful: `● running 42s` beats a paragraph.

## What we deliberately reject

- **Building an IDE.** No LSP, no explorer as home, no replacing the Pi TUI.
  Files in the agent cwd do open in the browser (ADR-0015, Track E). Diffs
  Keep / Undo write those same files. Worktrees are later.
- **Hiding the harness.** Cursor abstracts the agent away; PiCode exposes
  it — the real Pi TUI stays one tab away. Every Cursor-inspired
  convenience keeps its terminal escape hatch (philosophy: door, not cage).
- **Proprietary agent formats.** Rules and configs remain Pi primitives —
  plain files in the repo, portable and inspectable. Lock-in is not our
  moat; lifecycle control is.

## Checklist hookup

The `/skill:uiux-review` skill enforces the Cursor bar (product patterns
as they land per milestone; tokens and density from day one).
