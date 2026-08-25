# PiCode documentation

Living documentation — evolves with the code, by contract ([AGENTS.md](/AGENTS.md)).

| Doc | What it holds |
|---|---|
| [guidelines.md](guidelines.md) | **How we write docs** — internal vs public Pages; pi correlation |
| [architecture.md](architecture.md) | Components, protocols, security model |
| [philosophy.md](philosophy.md) | Moat, values, the "door not cage" principle |
| [benchmarks.md](benchmarks.md) | Engineering + UI/UX *bars* (enforced by skills) |
| [benchmarks/](benchmarks/) | **Who we study**: Cursor, t3code, paseo — dated notes |
| [benchmark-cursor.md](benchmark-cursor.md) | Cursor product patterns + aesthetic/density north star |
| [handoff.md](handoff.md) | **Project state right now** — start here |
| [design/voice-mode.md](design/voice-mode.md) | Dictation + voice composer: V1 shipped, V1.1–V3 phases |
| [design/slash-parity.md](design/slash-parity.md) | TUI `/` vs PiCode composer — 12/24 UI, rest planned |
| [design/pi-settings.md](design/pi-settings.md) | Plan: `#/settings` = pi GUI; `#/preferences` = PiCode |
| [decisions/](decisions/) | ADRs — architectural decision records (0011: workspaces vs agents) |
| [handoff-archive.md](handoff-archive.md) | Archived handoff activity (created when needed) |

User-facing docs: Markdown in [`www/`](../www/), VitePress → GitHub Pages.
Slash hints open `/commands#{id}` in a new tab. Rules:
[guidelines.md](guidelines.md).

Docs style follows our own benchmark: short paragraphs, tables for
comparisons, diagrams over prose, dates on status lines.
