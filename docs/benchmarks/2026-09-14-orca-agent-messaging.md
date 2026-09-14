# Study: Orca — agent messaging and idle evidence for native TUIs

- **Date:** 2026-09-14
- **Sources:** clone `/tmp/orca-study` at stablyai/orca HEAD (**MIT**,
  © Lovecast Inc.). Files ADR-0107 already cited:
  `skill-guides/orchestration/references/messaging-and-gates.md` and
  `src/main/runtime/orchestration/mailbox-pointer-delivery.ts`.
- **Scope:** what Orca teaches the communication family
  (ADR-0104/0107/0110) — mailbox, pointer delivery, idle detection.
  Orca's orchestration layers (coordinator, DAG, escalation, workers)
  are **out of scope by refusal** (ADR-0104, alternatives considered).
- **Why now:** the only surveyed product that combines agent↔agent
  messaging **and** our full CLI set; its README roster covers Pi,
  Claude Code, Codex, Grok, Hermes Agent and OpenCode alongside Cursor,
  Gemini, Copilot, Devin, Goose, Auggie and more.

## What it is

An Electron **agent workbench**: panes run guest CLIs as real TUIs, a
mailbox holds messages per agent, and delivery is a **pointer nudge**
typed into the live pane — the same shape we shipped. Two delivery
lanes: a **PTY lane** (types into a live pane, reads the idle edge off
the terminal title) and a **structured-session lane** (chat sessions;
the nudge is a session turn, the idle edge is the session journal).

## The idle-evidence stack (`tui-idle-evidence.ts`)

Three ranked tiers, explicitly reasoned in the source:

1. **POSITIVE** — an explicit idle marker the agent published itself:
   OSC/window title (`EXPLICIT_IDLE_TITLE_RE` = `ready|idle|done`),
   per-vendor prefixes (`✳` Claude, `◇` Gemini, `π - ` **Pi**),
   OpenCode's native name-only title.
2. **VETO** — the agent's own first-party status stream (OSC 9999)
   saying working/blocked/waiting; *"the agent's own account of itself
   outranks anything inferred."*
3. **ABSENCE** — a quiet output stream, sustained; last resort only,
   because *"a thinking TUI and a finished TUI are both silent, so the
   absence of a working marker can never prove completion."*

Per-vendor pattern catalogue (`terminal-wait-detection.ts`): ready-prompt
finders for Codex/Cursor/Antigravity, a braille-spinner busy regex for
Cursor, blocked-signal scanning for permission prompts, startup-modal
dismissal tracking, and a measured note that *an idle Grok pane repaints
its banner about four times a second forever*, so stream-quiescence never
settles for Grok.

## Pointer delivery (`mailbox-pointer-delivery.ts`)

The gate is `lastAgentStatus === 'idle'` **observed live**. The pointer
is typed, then Enter is scheduled after a delay; cancelling the timer
before it fires means the submission *provably* never happened and the
reserved mail is released — an Enter that *may* have landed stays
attempted and is never repeated.

## What it teaches PiCode

| Orca | PiCode today | Lesson |
|---|---|---|
| 3-tier idle evidence, title first | composer-frame match + native hooks | Our hooks are their tier-2 (first-party evidence), already shipped |
| Per-vendor prompt/title regexes | per-vendor frame catalogue (border, gutter, footer) | Per-vendor patterns are the **state of the art**, not a PiCode wart — both codebases hand-maintain them |
| Enter on a timer; provably-not-submitted → release | re-screenshot the frame before Enter; `uncertain` stays, no retry | Two defences of the same rule: an ambiguous submission is never repeated |
| Pane title as tier-1 | **measured and refused** (2026-09-14): none of our six vendor versions emits working/idle in the title; Orca's Claude `✳`-idle rule would read our mid-turn title as idle | The signal depends on vendor builds that publish it — re-measure if one starts (architecture, attention section) |
| Grok repaints ~4×/s at idle | matches our grok attention Enter-withheld under load (open finding) | Independent corroboration that Grok is the hardest pane to gate |

## Position

The only surveyed product with both agent↔agent messaging and our six
CLIs. Vibe Kanban covers most CLIs but agents never talk; Herdr is a PTY
runtime without an agent channel; A2A targets API agents; CrewAI/AutoGen
are frameworks, not TUIs. Our differentiators remain durable per-
conversation credentials, explicit ACKs, workspace consent and the
guarded-delivery contract — with Orca's orchestration deliberately
refused (ADR-0104).
