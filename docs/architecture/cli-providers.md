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

**Load models (P1).** `POST /api/providers/custom/models`
(`internal/server/custom_models.go`, `internal/modellist`) asks a custom
endpoint what it serves, so nobody copies ids by hand. This is the one place
PiCode speaks to a user-configured host, and it is a listing call only: no
prompt, no completion, no model traffic (ADR-0003). The request carries the
key the form holds, or — for an existing endpoint — the one the server reads
from `auth.json` (`catalog.ActiveAPIKey`), and it never travels back: the
answer is ids, limits and the URL that replied, and a message that echoed the
key has it redacted before it leaves the server (`internal/modellist`
`redact`, pinned by tests on both sides). The API type picks the shape and
the header: `{baseUrl}/models` with `Authorization: Bearer` for the OpenAI
types, `x-api-key` + `anthropic-version` for Anthropic's `{"data":[…]}`,
`x-goog-api-key` for Google's `{"models":[{"name":"models/…"}]}` (the
prefix is stripped). A base URL that does not already end in `/v1` (or
`/v1beta`) is offered `/v1/models` too, but only a missing list falls through
to it — a refused key would be refused twice. Failures are classified
(`auth`, `billing`, `missing`, `upstream`, `transport`, `input`) and each
carries the fix in one line: a 401 says whether a key was sent at all, a 404
suggests `/v1`, a transport failure names the host and the 12s budget. The
form merges what comes back into the ids already typed (never reordering or
dropping one) and fills context window and max output only when **every**
listed model reports the same number — the form writes one value for the
whole list, so agreeing is the only honest source; a list whose models differ
says so instead of leaving a blank field unexplained. An endpoint that lists
nothing is an answer, not an error. The copy lives in Go, so both apps say
the same thing, and `web/shared/client/modelLoad.js` (pure, no React) holds
the line-building and merge the dialogs share.

**URL hints per API type (P2).** The base-URL field carries a hint and a
placeholder that follow the API type (`customApiHint` in
`web/shared/contracts/schemas.js`), because pi appends the route itself and the
root differs: an OpenAI-style root usually ends in `/v1`, Google's in
`/v1beta`, and Anthropic-compatible endpoints differ — Anthropic's own root is
`https://api.anthropic.com/v1` while some proxies take no suffix. A test pins
that every offered API type has a hint, so a new type cannot leave the field
unexplained.

**Thinking formats (P1 + P3).** The Advanced section offers pi's own
`thinkingFormat` union: `openai` (the branch that sends `reasoning_effort`),
`openrouter`, `deepseek`, `together`, `baseten`, `zai`, `qwen`,
`chat-template`, `qwen-chat-template`, `string-thinking` and `ant-ling`. The
form writes `openai` rather than the undocumented `reasoning_effort` an earlier
version of this form produced; both reach the same pi branch, and the read path
maps the old value to `openai` so a hand-edited file prefills honestly. The two
formats driven by an object reveal it only when they are chosen
(`THINKING_FORMAT_NEEDS`): `chat-template` shows the `chatTemplateKwargs`
editor and `baseten` the `chatTemplateArgs` one, both plain JSON, validated in
the browser (`chatTemplateObjectError`) and again in Go
(`validateTemplateObject`) with pi's `$var` references (`thinking.enabled`,
`thinking.effort`, `thinking.budget`) and an optional `omitWhenOff` allowed and
anything else refused. The objects are compat's only nested values, so the
merge writes them whole and clears the key when the editor is emptied.

**Verify (P4).** For a built-in, Verify still asks pi (`pi auth check`), which
costs nothing and is the code path that runs the agent. A custom endpoint
cannot be answered that way — pi only reports that a credential is present, so
a wrong key on a gateway reads green — and `POST
/api/providers/{id}/verify` therefore sends **one minimal real request**
(`internal/modellist.Probe`): one word in, the smallest output ceiling each API
accepts (`max_tokens: 1`, or `max_completion_tokens` after one retry when the
gateway says so, 16 for the Responses API, `maxOutputTokens: 1` for Google).
The row's action names the cost before it is spent ("Verify with the endpoint
(1 request)"); the dialog can verify what the form holds, before saving. The
answer body is discarded — only the model, the milliseconds and the token
counts the endpoint reported travel back, and the verdict line shows them
("glm-4.6 answered in 43ms (4 in, 1 out)").

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
schema and again by `validateCustomDef`, so the GUI is not the only guard.**Thinking format.** The same section names how thinking travels on the wire
(`compat.thinkingFormat`), because gateways disagree: OpenAI-style endpoints
take `reasoning_effort`, DeepSeek and Z.AI their own fields, Qwen a top-level
`enable_thinking`, OpenRouter `reasoning: {effort}`, Together
`reasoning: {enabled}`. The select offers pi's documented formats
(`CUSTOM_THINKING_FORMATS`, mirrored by `customThinkingFormats` in
`internal/catalog/modelsjson.go` where an invented value is refused); the
empty choice means pi's own default for the API type and removes the key
instead of writing an empty string. `chat-template` and
`qwen-chat-template` are deliberately absent: both are driven by
`chatTemplateKwargs`/`chatTemplateArgs` objects this form does not edit, so
offering them would be a switch that does nothing. `thinkingFormat` is a
string among compat's bools, which exposed a read-path bug: decoding the
whole `compat` object as `map[string]bool` failed the entry and the provider
disappeared from the catalog. `LoadCustomDefinitions` now decodes compat key
by key (the two managed bools, the format string, everything else left
alone), pinned by `TestLoadCustomDefinitionsWithStringCompat`.

The catalog hands the stored `thinkingLevelMap` back inside `definitions`, so
Edit prefills the same chips it would write (`customThinkingLevels`). The row
made the create dialog taller than the viewport, and `.dlg-create` centres
with `top: 42%` and no height cap: it now caps at `80dvh` and scrolls inside
(2% margin top and bottom at that anchor) instead of clipping the title off
the top of the window.
