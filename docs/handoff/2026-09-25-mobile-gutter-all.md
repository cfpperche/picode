# 2026-09-25 — feat/mobile-gutter-all: every phone sub-route uses the 14px screen gutter only
Shipped: phone CSS only, extending feat/mobile-page-width to the remaining routes.
Stacked side padding removed (now 14px): Missions (.m-missions-body, phone-only
class in web/shared/styles/missions.css), New pin/New snippet (.m-pin-form,
web/mobile/src/mobile.css), terminal status messages (.m-tool-state inside a
.m-screen, web/mobile/src/styles/mobile-tools.css; standalone "Opening agent…"
keeps its padding). Intentional insets left alone: text after icons/checkboxes,
right-aligned values, centred empty states, CLI catalog tiles, CLI Keyboard rows
(bordered, changed-marker lane), app group cards.
Verified: `make ci-scoped` PASS; scratch at 390px, invisible-padding probe over 45
phone routes (work lists, workspace, agent, inbox, missions, automations, more/*,
pins/snippets new, preferences, system, devices, integrations, history, every
clis/codex section, clis messages/settings/new, llama/*, app/tmux, app/docker,
terminal). Blind spot: inbox items, agent chat messages and files/git probed empty only.
visual-review: PASS (Missions, New mission, New pin, New snippet, Terminal state)
Not done / debts: none. Pre-existing, noted not fixed: New pin/snippet show both a
header Save and a form Create; New mission has an empty band above a rule under the header.
Merge: fast-forward ready.
