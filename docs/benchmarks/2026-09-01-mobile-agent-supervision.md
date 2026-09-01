# Study: supervising coding agents from a phone, for the mobile shell (ADR-0044)

- **Date:** 2026-09-01
- **Sources:** public product documentation, App Store listings and
  release notes for Claude Code Remote Control, Codex in the ChatGPT
  mobile app, Cursor for iOS, AgentWatch, Tactic Remote, Nimbalyst,
  Happy, and PagerDuty's mobile app; general mobile UX rules (Apple HIG /
  Material 3 tab bars, thumb-zone guides). No clones — hosted products.
- **Scope:** what a *phone* surface for agents should be. Not a bar for
  the desktop shell.

## Why these

Every one of the sanctioned benchmarks (Cursor, t3code, paseo, herdr) is
a workstation product; none documents a phone surface beyond paseo's
"full desktop parity" mobile web. The question here is different — what
do people actually do with an agent from a phone — and the products that
answer it are the vendors' own mobile companions plus the on-call app
that has spent a decade on "decide from your pocket".

## What each does

| Product | What the phone surface is | Attention model |
|---|---|---|
| **Claude Code Remote Control** (docs) | A window into the local session: send messages and photos, approve every tool call, follow subagents; `/model x` as text where the CLI has a picker | Push "when actions required" and/or "when Claude decides"; **suppressed while you are at the machine** (presence file); forwarded dialogs expire; session list with an online dot; QR to open |
| **Codex in ChatGPT** (May 2026) | Approve decisions, review diffs (with "open full file"), redirect a running task, watch terminal output; the agent keeps running on the Mac | Push on turn completion and on "needs input"; reconnection UI; QR pairing |
| **Cursor for iOS** (June 2026) | Prompt and steer cloud/local agents, review and merge PRs; "comment, check status, ask for evidence, redirect" | Runs started anywhere show everywhere; **no code editing, not a mobile IDE** |
| **AgentWatch** | Session status, permission alerts, approve + send prompts, tokens and duration | Live Activity on the lock screen, Apple Watch, completion + permission notifications |
| **Tactic Remote** | Live terminal, file browser, approvals, prompt queue | Approval-focused pushes; a tap reopens the originating session |
| **Nimbalyst** | Kanban of sessions, swipe-through diffs, reply by typing or dictation | Notify on complete / fail / needs input, per project |
| **PagerDuty mobile** | Home = top open incidents + shifts; Incidents with Mine/All, sorted by urgency; detail with Triage/Overview tabs | Swipe to acknowledge/resolve, long-press ack from the notification, bottom nav (5), remembers the tab |
| **Mobile UX rules** | 3–5 bottom tabs with labels, 44–48 px targets, primary actions in the thumb zone, bottom sheets over modals, 16 px inputs (no iOS zoom), safe areas | — |

## What PiCode adapts

| Convention | From | PiCode's version |
|---|---|---|
| Decisions first, sorted by urgency | PagerDuty home; Remote Control's "forwarded dialogs expire" | `Now` opens with **Needs you**: live dialogs (they expire) before blocking inbox items; each answerable in place |
| Approve from the phone, keep running locally | Remote Control, Codex, AgentWatch | `POST /api/agents/{id}/ui` from the card; the agent screen's ask card is the desktop's |
| Redirect / interrupt / follow up | Cursor, Codex | Composer's Stop = abort; prompt/steer/follow-up chip; a prompt while busy becomes a follow-up in the server queue |
| Notification tap lands on the session | Tactic Remote | `#/agent/<id>`, `#/inbox/<id>` deep links; the same agent hash as the desktop, so a QR opens the right screen |
| Supervision, not development | Cursor ("not a mobile IDE") | No editor, tree, git graph or terminal tab; terminal only as a segment for a TUI agent |
| Bottom tabs, thumb-zone actions, sheets | HIG / Material / PagerDuty | 4 labelled tabs, 44 px targets, the create form as the Vaul sheet it already is below 720 px |
| Status + cost per session | AgentWatch, pi-agent-dashboard | Agent screen meta line: model · cost · context %; state chip working / needs you / idle |
| Dictation to reply | Nimbalyst | Comes with the shared `Composer` |

## What PiCode does not copy

| Not copied | Why |
|---|---|
| Remembered last tab (PagerDuty) | The home is the decision queue; landing on Agents while something waits is the wrong default |
| Live Activity / Apple Watch (AgentWatch) | Out of reach for a PWA; push (next ADR) is the alert channel |
| Kanban of sessions (Nimbalyst) | PiCode has no task board concept; Now + Agents cover "what is running" |
| Diff review with merge (Cursor, Codex) | Phase 3 candidate as a read-only Changes screen from `…/gitdiff`; merging stays a workstation act |
| Swipe actions on rows (PagerDuty) | Phase 3 polish; buttons first, gestures once the rows have settled |
| Push notifications | Deliberately the **next** ADR (Web Push over VAPID, Go stdlib, presence-aware like Remote Control's "skip while at the machine") — this study is its citation too |

## Relation to ADR-0044

This is the benchmark citation the `uiux-review` checklist asks of a
redesigned surface. The shell's structure, the one server change and the
reducer decision live in the ADR.
