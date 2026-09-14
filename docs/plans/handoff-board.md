# The handoff board is full — two defects and three options

**Decided**: option A + C, recorded in [ADR-0131](../decisions/0131-handoff-board-index.md)
(2026-09-13). This file keeps the measurement that motivated it and the
options as they were weighed; the ADR holds what was chosen. The rule it touches is **ADR-0123** (the board is a
generated view with a pruning budget) and `docs/handoff/open/README.md`
(bullets under `## Next` / `## Debts` are the unit).

## What was measured (2026-09-13)

| Measure | Value | Cap | Headroom |
|---|---|---|---|
| Board lines | 97 | 120 | 23 |
| Board characters | 12 232 | 12 288 | **56 (~½ bullet)** |

Average source bullet: 101 characters; rendered with its attribution
(` — *topic*`) about 126, and the two caps assume ~102 per line. So the
**byte cap binds about 20 lines before the line cap** and, in practice,
the board is full: a session that files one honest debt can fail a gate
that belongs to every topic at once — the last writer pays. That is what
happened on `feat/snippets-v2c`: 141 characters of headroom against 452
needed, and the only legal escapes were pruning other topics' debts or
saying nothing.

## Defect 1 — a topic file can be silently invisible

`make handoff` reads bullets that sit under `## Next` / `## Debts`. A file
titled with an `#` H1 and bullets beneath it contributes **nothing**:

```
docs/handoff/open/snippets.md:  0 visible / 6 invisible
```

Six snippet debts (shell-door gates, `snip.ran` 404s, uncapped decode,
the pixel-review debt) were invisible to every session that read the
board. Fixed in the same branch by giving the file its `## Debts`
heading and consolidating to the three that carry the meaning.

## Defect 2 — the two caps contradict each other

120 lines at the observed ~126 chars/line is ~15 KB, so the 12 KB cap is
unreachable-by-line-count by design of the data. Either the caps are
meant to disagree (bytes as the real pressure) or one of them is wrong;
today the disagreement is what makes "the board is 20% empty" and "the
board is full" both true.

## Options

| | Change | Cost | Effect |
|---|---|---|---|
| **A** | The generator **fails** on a topic file whose bullets sit outside `## Next` / `## Debts` (naming the file) | ~10 lines in `handoff-board.mjs`; a process-boundary change, so it amends ADR-0123 | Kills Defect 1 for good: no debt can hide again |
| **B** | Align the caps: either raise bytes (~16 KB) or keep 12 KB and enforce brevity (bullets ≤ ~140 chars, linted) | Raising it removes the pruning pressure ADR-0123 deliberately kept; linting needs a sweep of the long bullets | The board stays long but stops contradicting itself |
| **C** | The board renders **per topic: the count and the `Plan:` line**, with the bullets living in the topic file; a session opens the topic it needs | Bigger change to ADR-0123's "one place to read"; costs one extra open for a session that wants the detail | The board stops growing with the number of debts — the only option that scales |

## What was chosen (ADR-0131)

**A + C**, with one refinement: **`## Next up` keeps its bullets inline** (it
is bounded and actionable) and only **debts** become a per-topic count with
the topic's `Plan:`. Implemented in `scripts/handoff-board.mjs`, tested in
`scripts/handoff-board.test.mjs`; the board went from 12 232 to ~6 300
characters.

The reasoning as it stood before the decision:

**A + C.** A closes the silent-hole class cheaply and permanently. C is
the one that ends the monthly pruning ritual, because the detail already
lives in the topic files and the board is read once per session; a count
plus a link is what a reader needs to decide whether to open it. B only
if the long-form board is wanted on purpose — in that case raise the byte
cap *and* lower the line cap so the two stop disagreeing.

Not done here: this branch only fixed the two defects it could fix inside
the current policy (the `snippets.md` shape, and pruning what was
verifiably shipped or duplicated to buy the room).
