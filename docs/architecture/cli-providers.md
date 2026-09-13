# Native CLI providers (ADR-0103)

> Part of [PiCode's architecture](../architecture.md) (ADR-0105: one file per subsystem). Edit here; the index only links.

Providers is a pane of the selected CLI at `#/clis/<cli>/providers`; `/new`
opens Add provider. A shared route/capability helper names Pi explicitly and
redirects legacy desktop/mobile links (`#/clis/providers*`, `#/providers*`). Unsupported identities and explicit
agent/workspace scopes block editing; accounts still belong to the machine.
The editor loads its catalog independently of terminal inventory, retains
successful rows and drafts during refresh failures and offers retry. Successful
refreshes also update the app's model catalog. OAuth returns to the same app's
canonical list; closed/unmounted editors ignore late login completions.
Native provider APIs, the active Pi auth slot, extra-account vault and quota
semantics remain unchanged. The llama.cpp manager keeps its separate route.

The roster renders **one row per account**, grouped under the provider that
owns it, as a six-column grid (`Provider · Account · Identity · Usage ·
7d spend · actions`). Its geometry lives in
`web/shared/styles/providers.css`, imported last by both apps' `index.css`:
the two components stay per-app (ADR-0072), the roster's layout does not.
The column template drops the identity column below a 1000 px container and
folds the cells into a stacked card below 840 px, so one markup serves the
desktop window, a squeezed window and the phone.

**Custom endpoints (ADR-0129).** Add provider's picker carries a fixed
**Custom endpoint** door: a named definition (baseUrl, API type, compat
flags, model ids) that merges into pi's `~/.pi/agent/models.json` under the
schema's `providers` wrapper, and an API key stored in `auth.json` like any
native sign-in — never inside the definition. `PUT/DELETE
/api/providers/custom/{id}` (`internal/catalog/modelsjson.go`) merge by
provider id: untouched entries and unknown fields inside the touched one
survive, built-in ids are refused, and a file pi would reject is never
written. The catalog marks these rows `custom` and carries their editable
shape (`baseUrl`/`api`/`compat`/`definitions`, key material excluded); an
unsigned definition still appears so it can be picked up again. Row actions
map to the two files: **Edit endpoint** reopens the form, **Sign out**
removes only the credential, **Remove endpoint** deletes both with the
blast radius named.

**Thinking levels.** The form's Advanced section declares a reasoning model
and picks the levels it answers on (`minimal`…`max`), one selection for every
model listed, like context window and max output. The selection becomes pi's
per-model `thinkingLevelMap` (`internal/catalog/modelsjson.go`
`mergeThinkingLevels`): a selected level keeps its own name as the provider
value, an unselected managed level becomes `null` (pi hides it), and keys the
form does not manage — a hand-set `off`, a future pi level — survive. `off`
is deliberately absent from the form: pi's default map already covers it and
its provider value is not always the level name (several pi catalogs send
`none`), so the form never invents one. A model switched back to
non-reasoning drops the managed keys and keeps the rest; a model the user
never touched grows no map at all. An invented level is refused by the
schema and again by `validateCustomDef`, so the GUI is not the only guard.
The catalog hands the stored `thinkingLevelMap` back inside `definitions`, so
Edit prefills the same chips it would write (`customThinkingLevels`). The row
made the create dialog taller than the viewport, and `.dlg-create` centres
with `top: 42%` and no height cap: it now caps at `80dvh` and scrolls inside
(2% margin top and bottom at that anchor) instead of clipping the title off
the top of the window.
