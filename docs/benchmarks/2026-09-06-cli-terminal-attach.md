# Study: attaching images and files to Agent CLI terminals

- **Date:** 2026-09-06
- **Owner request:** after a phone session could not attach a photo to a
  prompt, architect attach for agents that run as Agent CLIs.
- **Scope:** images and artifacts (user files: screenshot, PDF, log) into
  the prompt of Pi, Claude Code, Codex, Grok and Hermes Agent terminals.
  Not Claude "Artifacts" (generated UI). Not ACP / first-class CLI agents.

## What is true in the repo today

Managed Pi chat already pastes, drops, clips from the **workspace on the
PiCode machine**, and sketches (`docs/design/composer-mcp-roadmap.md`).
There is no Photos/camera picker. Agent CLIs are terminals (ADR-0069): no
composer, no `/prompt`, no Inbox/Ask. Browser paste into xterm is
`clipboard.readText()` only. OSC 52 is write-only so a guest CLI cannot
read the system clipboard
([2026-08-30](2026-08-30-web-terminal-clipboard.md)).

ADR-0078: "PiCode never types into a CLI's input" — written so Inspector
git `type`/`run` cannot steal a live Claude prompt.

## Benchmarks

| Source | Fact | Take / refuse |
|---|---|---|
| Claude Code Remote Control ([mobile-supervision](2026-09-01-mobile-agent-supervision.md)) | Phone sends **messages and photos** into the local session; TUI stays on the machine | **Product bar.** PiCode adapted approvals and skipped photos; this study takes photos |
| Cursor iOS (same) | Prompt and steer; not a mobile IDE | Supervision, not an editor |
| Cursor composer (`benchmark-cursor.md` #6) | `@file` injects context | Paths, not dumped bytes |
| t3code ([2026-08-24](2026-08-24-adopt-t3code-paseo-cursor.md)) | Composer depth; runtime is SDKs + ACP | Composer *shape*. Refuse embedding SDKs (ADR-0003) |
| paseo (same) | PTY + provider hooks | Keep the real TUI; a side channel for input matches hooks for state |
| guest-tui ([2026-09-03](2026-09-03-guest-tui-agent-state.md)) | Level 2 = TUI + side channel; level 4 = our UI | Prompt door is level 2. ACP stays a future ADR |
| VS Code "Claude Paste" / SSH image paste | Clipboard image → file on the host → inject the **path** | Same transport, no vendor SDK |
| Claude Code (2026 docs) | Path in the prompt is the method that works everywhere; Ctrl+V is flaky on Windows/WSL | Delivery = message + relative paths |
| Codex CLI | `--image` on first prompt; mid-session the TUI treats image **paths** as attachments | Do not relaunch with `-i` |
| composer-mcp-roadmap | Refuse dumping bodies into the prompt; refuse image bytes in SQLite | Drop folder + path. Caps 4 × 4 MB |

Grok CLI and Hermes have no public image protocol. Path + "read this file"
is the portable fallback.

## Decision (→ ADR-0087)

Do **not** replace the guest TUI with PiCode chat. Give a **user-initiated
prompt door**: stage files under `<cwd>/.picode/drop/`, paste the caption
plus `@path` into `picode-sh-<id>` via the same `PasteText` ADR-0060 uses
for Pi Inbox. Inspector Ask/type/run still must not target a CLI TUI.

The managed composer also gains a device file picker (Photos / camera /
files). That picker does not need this ADR.

## Refuse

| Temptation | Why not |
|---|---|
| Fake Chat \| Terminal transcript for Claude/Grok | ADR-0079: we do not own those session files |
| ACP / vendor SDKs as v1 | ADR-0003 / 0069 |
| OSC 52 read / feeding the guest clipboard | 2026-08-30 refuse |
| Inspector typing into a CLI TUI | ADR-0078 stays for that door |
| Dump PDF bytes into the pasted text | The CLI `read`s the path |
