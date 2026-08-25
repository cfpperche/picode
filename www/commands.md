# Commands

Type `/` in the composer. PiCode opens **its** UI — it does not type into the terminal.

Each slash-menu hint opens this page at that heading.

Canonical pi reference: [Sessions](https://github.com/earendil-works/pi/blob/main/packages/coding-agent/docs/sessions.md) · [Settings](https://github.com/earendil-works/pi/blob/main/packages/coding-agent/docs/settings.md) · [Usage](https://github.com/earendil-works/pi/blob/main/packages/coding-agent/docs/usage.md)

## /tree {#tree}

Session tree. Each card is a user prompt; replies and tools sit on the card.

Click a card to **fork** (new session file from that prompt). Clone copies this branch.

| | pi TUI | PiCode |
|---|---|---|
| View | tree of this JSONL | same (cards) |
| Click a prompt | in-place leaf jump (`navigateTree`) | **fork** (new file) |
| `/clone` | new file | new file (RPC) |

RPC has `get_tree` / `fork` / `clone`, not `navigate_tree`. Asked upstream: [pi#8645](https://github.com/earendil-works/pi/issues/8645). PiCode will not write a private leaf into pi's JSONL.

## /fork {#fork}

New session from a previous user prompt. Opens the tree; pick a card. RPC `fork`.

| | pi TUI | PiCode |
|---|---|---|
| Result | new session file from that turn | same (RPC `fork`) |

## /clone {#clone}

Duplicate the current branch into a new session file. RPC `clone`.

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

Opens Add provider. API key is saved in pi's `auth.json`. Account/subscription still needs TUI `/login` (no RPC).

Canonical: [pi Providers](https://github.com/earendil-works/pi/blob/main/packages/coding-agent/docs/providers.md).

## /logout {#logout}

Opens `#/providers`. Sign out removes that provider from `~/.pi/agent/auth.json`.

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
