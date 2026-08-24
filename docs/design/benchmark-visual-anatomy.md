# Visual benchmarks — reference anatomy (Cursor-class agent IDEs)

> Status: v2 — 2026-08-24. **Provenance note (honesty rule):** written from
> trained knowledge of Cursor's design language; NOT captured from live
> screenshots (no web access in the authoring session). `t3code` unknown to
> the authoring agent; paseo.sh knowledge is thin. **Reference screenshots
> from the owner live in `docs/design/references/`** — vision-capable
> sessions MUST verify/refine this spec against them (see "Open questions").

## The anatomy (what makes an agent IDE look like one)

```
┌──────┬─────────────────────────────────────────────┐
│ side │  header: agent name · model chip · status    │ 36px
│ bar  ├─────────────────────────────────────────────┤
│      │                                             │
│ list │         CONVERSATION (center col)           │
│      │   user block / assistant block / tool pill  │
│      │                                             │
│      ├─────────────────────────────────────────────┤
│      │  composer: [kind chip] textarea [send ⏎]    │ rounded box
│      ├─────────────────────────────────────────────┤
│      │  terminal dock (tabs · statusbar)  ~40%     │ optional
└──────┴─────────────────────────────────────────────┘
```

**The hierarchy rule (the actual fix):** the conversation is the hero — a
centered ~720px column like Cursor's chat panel. The terminal is a *dock*
at the bottom (Cursor's terminal pane), never a competing full view.
Tool calls are *pills inside the conversation*, not rows in a separate
stream. Nothing stacks: one center surface, one dock, explicit toggles.

## Concrete spec (v2)

| Token | Value | Note |
|---|---|---|
| bg-base | `#0e0e11` | near-black, Cursor-dark class |
| bg-panel | `#131317` | sidebar, header, dock |
| bg-elevated | `#1b1b21` | composer container, pills |
| border | `#232329` | hairline; depth = surface steps |
| text-primary | `#ececf1` | |
| text-secondary | `#9b9ba7` | |
| accent | `#7c8cf8` | links, focus, streaming dot |
| ok / warn / danger | `#63d297 / #e5c07b / #f7768e` | |
| Type | UI 13px/1.5 · mono 12px | |
| Radius | 8px pills/inputs · 12px composer | |
| Density | header 36px · pill 24-28px · grid 4px | |

## Component patterns (Cursor-inspired)

- **Conversation blocks**: full-width, small actor label ("You" / agent
  name / tool) in 11px secondary + content. No chat bubbles — flat blocks
  with hairline separators, like Cursor's composer panel.
- **Tool pill**: one line: icon + tool name (mono, accent) + one-line arg
  summary + status glyph; click expands mono detail block.
- **Composer**: rounded-12 elevated container; textarea grows; bottom row:
  kind chip (Prompt/Steer/Follow-up) left, send button right; Enter sends,
  Shift+Enter newlines; "queued" feedback inline, never modal.
- **Model/status chip row**: agent header is ONE 36px row: name · provider/
  model chip · streaming dot + verb ("streaming"/"idle"/"stopped").
- **Terminal dock**: bottom pane with its own compact tab strip and a
  toggle in the header row; closing the dock detaches (agent keeps running).

## Open questions (for vision sessions)

1. Verify against owner screenshots in `references/` (Cursor, t3code,
   paseo.sh): exact sidebar density? chat column width? composer chrome?
2. t3code: extract patterns (unknown to authoring agent).
3. paseo.sh: confirm/refute remembered "warm dark terminal-first" vibe.
