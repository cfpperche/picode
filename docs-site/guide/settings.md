---
description: Each CLI's own settings, edited from Agent CLIs — Pi's JSON one layer at a time. Not Preferences.
---

# CLI settings

Two different screens. Do not mix them.

- **Where:** **Agent CLIs**, pick the CLI, then the **Settings** pane (`#/clis/<cli>/settings`). Composer `/settings` opens Pi's pane with the selected agent in the URL.
- **Not this:** not Preferences (theme, server port). That is PiCode chrome. This pane is the CLI's own configuration. Overview: [Configure](/guide/configure).

Claude Code, Codex, Grok, Hermes Agent, OpenCode, Muse Code, Antigravity and
Omp each get a pane over that CLI's own settings file: the rows its vendor
documents, the file it writes, and the CLI's own defaults. The rest of this
page is Pi's editor, which has layers.

| Hash | What | Writes |
|---|---|---|
| `#/clis/pi/settings` | **pi** JSON for the selected agent | `~/.pi/agent/settings.json` (this machine), `<cwd>/.pi/settings.json` (workspace, if trusted) |
| `#/clis/pi/keyboard` | the keyboard map | `~/.pi/agent/keybindings.json` (one map per machine) |
| `#/preferences` | **PiCode** chrome | theme, server port |

The pane edits **one layer at a time**. **Edit** picks it — global, the
workspace, or the agent — and the line under it is the file that layer writes.
A row says where its value comes from: **Set here** when this layer sets it
(accent bar on the left), otherwise **Pi default** or **From Global**.
A row this layer sets has **Use inherited**, which hands the value back to the
layer below instead of freezing a copy of it.

**Keyboard** is the pane next to Settings: the whole keyboard map of this
machine, one row per action. The **Filter keys** field narrows by action name,
group or chord, and **Find by key** takes a chord you press and shows the
actions that answer to it. The facets count what they filter — **Changed**, **Shared** (a key two
actions use; pi's contexts overlap on purpose), **Off** (unbound). One
**Add key** per row records the next chord you press; **Reset** returns that
row to pi's default, and **Reset all** returns every changed row at once. The
pane says which file it writes and which bindings *this machine* has: nine
actions differ on Windows and WSL, and the rows that use a chord a browser
keeps are labeled, because those never reach the terminal inside PiCode.

It has no layer — Pi keeps one map per machine — and the link remembers which
agent and layer you came from, so going back lands where you left.

Contextual links use `?agentId=<id>`; the pane adds the layer it is editing
(`?scope=global|workspace|agent`, the same words every setup tab uses) to the URL, so a reload or a bookmark lands on
the same view. Old `#/settings` and mobile `#/more/settings` links redirect
here, and a `?tab=keys` link from the day the map was a sub-tab lands on the
Keyboard pane. Workspace values override machine
values; agent values override both.


Workspace writes require the folder in pi's `trust.json`. Untrusted → the
workspace layer says so and offers **Open agent to trust**; the same write
returns 409. Run `/trust` in the TUI.

## Model roles (Omp)

Omp routes different jobs to different models, and its **Models** pane edits
that map directly, above the list of models it can reach. **Model roles** lists every role the CLI has — DEFAULT, SMOL,
SLOW, VISION, PLAN, COMMIT, TINY, MEMORY, TASK, ADVISOR, and the five that pick
a model by kind (IMAGE, WEB, SPEECH, DICTATION, JUDGE) — plus any role you
invented. A role nobody assigns reads **auto**: Omp picks for it.

Each row has two controls. The **model picker** searches what Omp itself
reports it can reach in this workspace, and also offers `@another-role` and `*`
so one role can point at another. The **thinking** select beside it writes the
`:level` suffix. A role in the quick-switch cycle carries the same `⟳ N` badge
Omp's own model hub draws, counting the stops of its ctrl+p cycle.

**Quick-switch cycle** is that list, in order. **Fallbacks** below it holds one
chain per role, model or `provider/*`: when a model fails, Omp tries the next
entry. An empty chain is how you say *never fall back*. **New role…** and
**New fallback…** create a row; **Use inherited** removes it from this layer.

The workspace layer is the reason this pane exists: `omp config set` writes the
global file wherever you run it, so a per-project role can only be set by
editing `<workspace>/.omp/config.yml`, which is what the **This workspace**
layer does.

Omp reads its config when it starts. A change here reaches a running Omp
terminal when that terminal restarts — and every other open Omp terminal reads
the same global file, so they each pick it up on their own next start.

## Models (Omp)

Below the roles and fallbacks, **All models** lists every model Omp reports it can reach in the
workspace you opened it from, grouped by provider, with its context size and
price per million tokens. The chips filter by kind (chat, tiny, image, speech,
search, …) and the box filters by name.

- **Allowed** limits Omp to the models you switch on. With none on, Omp may use
  every model listed. With one or more on, it uses *only* those — and if none
  of them is reachable in this folder, the pane says Omp has no model to use.
- **Hide provider** removes a provider's models from Omp in the layer you are
  editing; **Show** brings it back.

Like Settings, the pane edits one layer at a time. A workspace list replaces
the global one rather than adding to it, so the pane always writes the whole
list for the layer — including what it inherited. The first answer takes a few
seconds because PiCode asks Omp itself; **Refresh** asks again.

## Checks (Omp)

At the top of Omp's Settings pane, **Checks** lists what is wrong with Omp's
configuration for the folder you are in, one line each: a config file Omp moved
aside because it could not read it, a key written twice, a key this version of
Omp no longer knows, a committed `.env`, no approval mode set (Omp then runs
every tool without asking), and how many Omp terminals will only see a change
after they restart. **Open the file** or **Change it** takes you to the fix.

Under it, **All settings Omp resolves here** is Omp's own list of every setting
in force in this folder, with the file each value comes from. Credentials are
shown as *hidden*.

## Related pi documentation

Canonical: [pi Settings](https://github.com/earendil-works/pi/blob/main/packages/coding-agent/docs/settings.md)

| | pi TUI `/settings` | PiCode `#/clis/pi/settings` |
|---|---|---|
| File | **global only** | machine + workspace + agent, one layer at a time |
| Project file | `pi config` / `pi install -l` | the workspace layer, if trusted |
