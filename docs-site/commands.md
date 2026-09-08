# Commands

Type `/` in the composer. PiCode opens **its** UI — it does not type into the terminal.

Each slash-menu hint opens this page at that heading.

Canonical pi reference: [Sessions](https://github.com/earendil-works/pi/blob/main/packages/coding-agent/docs/sessions.md) · [Settings](https://github.com/earendil-works/pi/blob/main/packages/coding-agent/docs/settings.md) · [Usage](https://github.com/earendil-works/pi/blob/main/packages/coding-agent/docs/usage.md)

## /tree {#tree}

Prompts on a timeline. Each card is a user prompt.

The current prompt says **Now**. Pick another to continue from there (new session; this one stays). Duplicate copies the timeline.

| | pi TUI | PiCode |
|---|---|---|
| View | tree of this JSONL | same (cards) |
| Click a prompt | in-place leaf jump (`navigateTree`) | **new session** from that prompt |
| Duplicate | new file | new file |

RPC has `get_tree` / `fork` / `clone`, not `navigate_tree`. Asked upstream: [pi#8645](https://github.com/earendil-works/pi/issues/8645). PiCode will not write a private leaf into pi's JSONL.

## /fork {#fork}

Same timeline as `/tree`, then pick a prompt to continue from.

| | pi TUI | PiCode |
|---|---|---|
| Result | new session file from that turn | same (RPC `fork`) |

## /clone {#clone}

Duplicate this timeline into a new session. This one stays.

| | pi TUI | PiCode |
|---|---|---|
| Result | new file, same branch | same (RPC `clone`) |

## /settings {#settings}

Opens `#/settings` — pi JSON for the selected agent. Not PiCode chrome (`#/preferences`).

See [Settings](/guide/settings).

## /scoped-models {#scoped-models}

Opens Settings, scrolled to scoped models (`enabledModels` patterns and default tools).

Canonical: [pi Settings](https://github.com/earendil-works/pi/blob/main/packages/coding-agent/docs/settings.md) (`enabledModels`).

## /model {#model}

Focuses the model chip in the composer.

## /thinking {#thinking}

Focuses the thinking-level chip.

## /provider {#provider}

Focuses the provider chip.

## /new {#new}

Starts a new session for this agent (same as the session bar).

Canonical: [pi Sessions](https://github.com/earendil-works/pi/blob/main/packages/coding-agent/docs/sessions.md) (`/new`).

## /resume {#resume}

Opens the session picker.

Canonical: [pi Sessions](https://github.com/earendil-works/pi/blob/main/packages/coding-agent/docs/sessions.md) (`/resume`).

## /name {#name}

Renames the current session.

Canonical: [pi Sessions](https://github.com/earendil-works/pi/blob/main/packages/coding-agent/docs/sessions.md) (`/name`).

## /compact {#compact}

Compacts context (confirm first).

Canonical: [pi Compaction](https://github.com/earendil-works/pi/blob/main/packages/coding-agent/docs/compaction.md).

## /login {#login}

Opens Add provider. See [Providers](/guide/providers).

Canonical: [pi Providers](https://github.com/earendil-works/pi/blob/main/packages/coding-agent/docs/providers.md).

## /logout {#logout}

Opens `#/providers`. Sign out removes that provider. See [Providers](/guide/providers).

Canonical: [pi Providers](https://github.com/earendil-works/pi/blob/main/packages/coding-agent/docs/providers.md).

## /copy {#copy}

Copies the last assistant reply to the clipboard. Same text as the bubble copy button.

## /quit {#quit}

Stops **this** agent and closes its tab. Does not close the browser.

| | pi TUI | PiCode |
|---|---|---|
| Effect | quit the pi process | stop this agent |

## /trust {#trust}

Trust this agent's folder (`trust.json`). Needed before PiCode can write workspace `.pi/settings.json`.

Canonical: [pi Settings](https://github.com/earendil-works/pi/blob/main/packages/coding-agent/docs/settings.md) (project trust).

## /session {#session}

Dialog: name, file, folder, git, model, tokens, context, cost.
Actions: copy path, rename, new, compact, tree.

Canonical: [pi Sessions](https://github.com/earendil-works/pi/blob/main/packages/coding-agent/docs/sessions.md) (`/session`).

## /reload {#reload}

Restarts this agent so skills and config reload. Session JSONL stays. RPC has no `reload`.

## /export {#export}

Downloads this session as JSONL.

Canonical: [pi Usage](https://github.com/earendil-works/pi/blob/main/packages/coding-agent/docs/usage.md) (`/export`).

| | pi TUI | PiCode |
|---|---|---|
| Default | HTML | JSONL download |

## /import {#import}

Pick a `.jsonl` file and resume it as this agent's session.

Canonical: [pi Usage](https://github.com/earendil-works/pi/blob/main/packages/coding-agent/docs/usage.md) (`/import`).

## /share {#share}

Secret GitHub gist of this session (JSONL + HTML). Needs `gh auth login`. Not the phone QR.

Canonical: [pi Usage](https://github.com/earendil-works/pi/blob/main/packages/coding-agent/docs/usage.md) (`/share`).

| | pi TUI | PiCode |
|---|---|---|
| Upload | `gh gist create` (HTML; Radius if signed in) | `gh gist create` (JSONL + HTML) |

## /hotkeys {#hotkeys}

PiCode shortcuts (palette, composer, send). Not the TUI keymap.

## /changelog {#changelog}

Changelog of the **installed pi** package. Not PiCode's CHANGELOG.

Canonical: pi repo CHANGELOG.

## /llama {#llama}

Opens a dialog: router URL, load/unload, Hugging Face download. Setup (install/start the server) is in [llama.cpp](/guide/llama).

Canonical: [pi llama.cpp](https://github.com/earendil-works/pi/blob/main/packages/coding-agent/docs/llama-cpp.md).

| | pi TUI | PiCode |
|---|---|---|
| Login | URL + optional key | same |
| Manage | `/llama` load/unload/download | load/unload/download HF |

## Skills and templates

`/skill:name` and `/templatename` appear in the composer picker (global + trusted project). Choosing one **inserts** the command; Send lets pi expand it (RPC already expands skills and templates).

Canonical: [Skills](https://github.com/earendil-works/pi/blob/main/packages/coding-agent/docs/skills.md) · [Prompt templates](https://github.com/earendil-works/pi/blob/main/packages/coding-agent/docs/prompt-templates.md).

## /automate {#automate}

Draft an automation from a sentence: `/automate every weekday at 9, summarize what changed since yesterday`. The open agent looks at the repository, proposes a name, a prompt, a schedule and limits, and PiCode opens the Automations editor pre-filled for you to review and create. Bare `/automate` asks for the description first. PiCode-only; the agent's reply stays in its session like any other turn. Guide: [Automations](/guide/automations).
