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

One pane serves all nine CLIs (ADR-0169): pi's own editor is gone and this
roster renders **one row per account**, grouped under the provider that owns
it, as a grid (`Provider · Account · Usage · 7d spend · actions`) — the same
columns for pi and for a guest, with the identity chips (`Subscription · in
use`, `API key · Vault`, `Unverified`) inside the Account cell. Its geometry lives in
`web/shared/styles/providers.css`, imported last by both apps' `index.css`:
the two components stay per-app (ADR-0072), the roster's layout does not.
The column template drops the identity column below a 1000 px container and
folds the cells into a stacked card below 840 px, so one markup serves the
desktop window, a squeezed window and the phone.

**The pane (ADR-0165, unified by ADR-0169).** The store, the API and the backup
rules have their own file ([credentials.md](credentials.md)); this is the pane.
All nine CLIs render `CliCredentials.jsx` from
`GET /api/credentials?cli=<id>`, which serves pi's providers from the catalog
(native and custom) and a guest's from the `clicreds` declarations, with the
same account rows either way. Omp's declaration is extended at read time (ADR-0183) from
its installed package: `internal/clicreds/omp_catalog.go` finds
`@oh-my-pi/pi-catalog/src/compat/rules.json` next to the `omp` on PATH
(`PICODE_OMP_RULES` overrides) and appends every `/login` provider the
declaration lacks, in Omp's order, with Omp's name (`name` on the row), the
first variable it reads a key from (`env.api_key`, which the launch injects)
and `oauth` where Omp has a sign-in flow, done in Omp's own `/login`. Omp's
aliases of declared ids (`meta`, `moonshot`, `kimi-code`, `opencode-zen`,
`opencode-go`, `xai-oauth`, `openai-codex-device`) are skipped, and so is a
provider with neither a variable nor a sign-in. The file is Omp's internals,
not an API: when it moves, the roster is the declaration alone. `supportsCliCredentials` in
`web/shared/domain/cliProviders.js` names the eight that read the vault, and
the pane's words —
provider names for the vault's ids, the kind and source chips, a Verify
answer as a label with a tone, the row order — live in
`web/shared/domain/credentials.js` so both apps say one thing (ADR-0072).
Each provider the CLI declares renders its saved accounts as rows
(`label · identity · kind chip · source chip · masked hint · health chip with
the vendor's own line · one menu`): the menu carries Rename, Verify — only
where the provider has a verifier, and the item names the cost ("Verify with
the provider (1 request)"), because the click spends one listing call —
Pause/Resume and Sign out behind a confirmation that names the account.
Accounts in play sort first, then by the name the person gave them, with the
id as tie-break; the vault's `active` slot is pi's alone and never reorders a
guest CLI's rows.

A login the CLI already holds is not a vault row and is not counted as one:
it renders as the provider's own highlighted line ("me@example.com is signed
in here") with the single action **Import into the vault**, and the row
disappears as the account appears above it. The bar's primary action is
per CLI (`add` in the roster) and is called **Add provider** for all of
them: pi's dialog (`AddProviderDialog.jsx`) runs pi's own OAuth or stores a
key and carries the door to the custom-endpoint page. pi's flow is the model
for every CLI (the owner's call, 2026-09-22); omp is the first guest on it
(`add.kind: "provider"`). Fed the guest's roster instead of pi's catalog, it
offers only the doors that CLI has natively: a key where the row names the
variable it is passed in (`env.api_key`), saved with `POST /api/credentials`;
an account where the row's `signin` says how — `browser` (PiCode's OAuth
engine, polled like pi's) or `terminal` (the CLI's own login, handed to the
pane's sign-in strip) — through `POST /api/credentials/signin`; and Custom
provider where the roster's `custom.available` says the CLI keeps
definitions. A provider with none of these is not offered. Claude Code has a
dialog of its own (`ClaudeCodeLoginDialog.jsx`, `add.kind: "claude-code"`,
ADR-0187) shaped like its `/login`: a Claude subscription through the browser
(PiCode's OAuth with Claude Code's client; the login is written into
`~/.claude/.credentials.json` when no Claude Code terminal runs), an Anthropic
Console key ("Save and use": the key becomes the login in use — the setting
`credentials.claude-code.key`, its tail approved in `~/.claude.json`, and
`ANTHROPIC_API_KEY` at launch, since a key outranks the subscription there),
and "Sign in from a terminal" as the fallback. Use switches between them. A
third option, "3rd-party platform" (ADR-0189), sets Claude Code up on Amazon
Bedrock, Microsoft Foundry or Google Vertex AI with the sign-in methods its
own wizard lists, by writing the same `env` block into `~/.claude/settings.json`
(`PUT`/`DELETE /api/claude-code/platform`); the roster's `platform` field says
which is in use, never a secret, and the pane shows "Claude Code uses …" with
Edit and Stop using. A platform outranks the subscription and the key, so
choosing one clears the key and choosing either removes the platform. The
same option offers an Anthropic-compatible gateway (ADR-0190): base URL,
token and optional models in the same `env` block; a URL with no token is a
proxy and stays untouched, and with a gateway in use no Console key is ever
injected — Claude Code would send it to the gateway as `x-api-key`. Codex has
its own dialog too (`CodexLoginDialog.jsx`, `add.kind: "codex"`, ADR-0191),
shaped like its `/login` and run through Codex's own `codex app-server`
(`POST`/`GET`/`DELETE /api/codex/login`): ChatGPT in the browser, a device
code, an API key, Amazon Bedrock. Codex writes its own files; a ChatGPT or
key login is then imported into the vault; a non-Bedrock login removes the
`model_provider = "amazon-bedrock"` Codex leaves in `config.toml`. The
other guests'
dialog still holds a provider `<select>`
limited to that CLI's providers, the key field with the line that says the
key stays on this machine, Save — and, for a provider the CLI signs into
(an `oauth` kind plus a declared sign-in), **Guided sign-in**, which closes
the dialog and starts the ADR-0168 strip, because the vendor's OAuth
happens in the CLI, never here. The table lists only providers the user
has something for — an account, a custom definition, or a CLI login waiting
to be imported; the rest of the roster (pi's whole catalog, a guest's
declaration) stays reachable through the Add dialog's select. With nothing
to list, the pane says `No provider credentials for <CLI> yet.` under the
bar's Add. A provider shown for its login alone reads
`Name · No accounts yet.`, never an empty well, and a provider whose
declaration carries a `note` shows
that one line instead of a Verify control that could not answer.
`#/clis/<cli>/providers/new` opens the dialog, and closing it rewrites the
hash back to the pane. Every server error lands in the form that caused it —
a toast would leave the sheet looking fine.

**Sign in (ADR-0168, extended by ADR-0178).** The pane starts the CLI's *own* login: `POST
/api/credentials/signin` creates a terminal running that CLI's login argv
(`codex login`, `grok login`, `muse login`, `opencode auth login`, `hermes auth
add`; the CLIs whose login lives in their TUI get no arguments and a hint
naming the command inside it — `/login`). For that terminal path PiCode
performs no vendor OAuth and presents no other product's client id. The
exception is the Add-provider dialog's **Guided sign-in** for a provider the
OAuth engine supports (`oauth.Supports`: anthropic, openai-codex,
github-copilot, kimi-coding, xai): there PiCode opens the vendor's authorize
page in a tab with the CLI's own public client id — the same handshake pi's
TUI runs (`internal/oauth`) — captures the loopback/device callback, and
lands the credential in the vault; for omp it travels back by env at spawn
(ADR-0178). The roster carries
`signin: {available, hint}` so the button exists only where it is honest, and
the pane shows the hint plus **Check now** — one click that runs the import and
files the result. A sign-in whose credential the store cannot name (Claude
Code, Muse, offline) would replace the unnamed row, so the pane asks for a name
and `Store.Adopt` re-keys that row instead of leaving a second copy of one
credential. A vendor CLI's own login is one of the four actions' worth of
material for the vault, and the least surprising one: the person uses the flow
the vendor documents, PiCode files it.

**Usage and 7d spend travel with the account, not the CLI.** Each row carries
`usage` when the cache has a report for that provider+account
(`internal/usage`, no vendor call on a load — ADR-0031), rendered by the same
`QuotaStrip` pi's table used, and **Check** spends one listing call through
`GET /api/providers/{id}/accounts/{aid}/usage`. 7d spend is PiCode's own
number: one `/api/sessions/stats?range=7d` per pane load through
`spendByProvider`. A row with no source says `unknown` and offers Check; a
provider with no session spend shows a dash, never `$0.00`.

Identity is what the roster matches on. A row's key is the account the store
names (Grok's `principal_id` for `Identity`, Codex's `tokens.account_id`,
Hermes' `account_id`, Antigravity's `id_token` subject), else the name the
person gave, else the fingerprint — which for a credential whose bytes name
nothing (`oauth` without an account id, an env-shaped `api_key`) is a constant,
one row per provider. The vendor's profile endpoint
(`usage.Identity`, Anthropic today) answers the email for the row's second
line and nothing else: a key the roster cannot recompute offline would cost the
pane its "in use" line on every load. For rows keyed by a name the offline
reader cannot compute, `inUseID` matches the live file by the token the row was
saved with, and matches nothing when the store says nothing — never a guess.

Honest states: a load skeleton with the real shape; a failed load is one line
with **Try again**, and the last good roster stays on screen; a vault that
cannot be read is one line carrying the vault's own problem and **no
controls at all** (nothing there could succeed); a CLI the server declares no
credential mechanism for is one line, with no Add button. Geometry lives in
`web/shared/styles/credentials.css`, imported after `providers.css` by both
apps' `index.css`: one container query drops the masked-hint column, a second
folds the same cells into a stacked card — the pane's container tops out near
940 px inside Agent CLIs, so the aligned row is what a desktop window usually
shows and the card is what a phone gets. **Use** writes the CLI's own
file for every CLI under ADR-0166 — `auth.json` for pi, the vendor's file for a
guest — and the pane never returns a secret: the masked hint is all the server
sends.

**Custom providers (ADR-0129, extended to omp by ADR-0175).** Add provider's
picker carries a fixed **Custom provider** door onto its own page
(`#/clis/pi/providers/custom`, `#/clis/pi/providers/custom/<id>` for Edit —
benchmarks.md refuses modals for flows longer than 2 fields): a named
definition (baseUrl, API type, compat flags, model ids) that merges into pi's
`~/.pi/agent/models.json` under the schema's `providers` wrapper, and an API
key stored in `auth.json` like any native sign-in — never inside the
definition. omp carries the same door on its own pane
(`#/clis/omp/providers/custom`): its definitions live in `~/.omp/agent/
models.yml`, merged node-level so unknown fields and comments survive, and
the key rides inside the definition as `apiKey` — the one channel a custom
omp id has (no auth.json, no env mapping), never serialized back: the
roster's row carries the editable shape plus `keyed`, and the pane renders a
definition line (Edit / Verify / Remove) instead of an account row. The omp
form offers omp's accepted subset (two compat flags, five thinking formats,
no chat-template objects, no thinking levels); the server refuses the rest.
`PUT/DELETE /api/providers/custom/{id}` take `cli` and dispatch to the right
file (`internal/catalog/modelsjson.go`, `internal/catalog/modelsyaml.go`);
built-in ids are refused on both, and a file the CLI would reject is never
written into. The catalog marks these rows `custom` and carries their
editable shape (`baseUrl`/`api`/`compat`/`definitions`, key material
excluded); an unsigned definition still appears so it can be picked up
again. Row actions map to the files: **Edit provider** reopens the page,
**Sign out** (pi) removes only the credential, **Remove provider** deletes
both with the blast radius named.

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
dropping one). Limits are **per model**: one listing can carry 1M/384k for one
id and 200k/131k for the next, so agreeing is not a precondition and no number
is invented. Each id gets its own row (Models → Model limits) and a load
fills the rows the endpoint reported. An endpoint that lists nothing is an
answer, not an error. The copy lives in Go, so both apps say
the same thing, and `web/shared/client/modelLoad.js` (pure, no React) holds
the line-building and merge both apps share.

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
`thinkingFormat` is a string among compat's bools, which exposed a read-path
bug: decoding the whole `compat` object as `map[string]bool` failed the entry
and the provider disappeared from the catalog. `LoadCustomDefinitions` now
decodes compat key by key (the two managed bools, the format string,
everything else left alone), pinned by
`TestLoadCustomDefinitionsWithStringCompat`.

**What a load writes (P5).** The merge is condition-driven, and the rows below
are the whole table — each is pinned by `web/shared/client/modelLoad.test.js`
or `web/shared/domain/customProviders.test.js`:

| Condition | What the form writes |
|---|---|
| Id already typed and reported by the endpoint | kept where it is; its row filled |
| Id typed, not reported | row kept, numbers blank — blank is pi's default, never zero |
| Number already typed by hand | survives the load; a load never overwrites an edit |
| Id removed from the list | its limit row goes with it |
| Endpoint reports no limits for an id | the row appears with blanks; the line counts what it filled, and stays silent about limits when it filled none |
| Listing fails | no row changes; the Model ids line turns red with the fix |

**Form rhythm (P5).** One field is one unit — label above, control, one line
of help below, 8 px apart, 24 px between units — adapted from shadcn's Field
scale (`web/shared/styles/providers.css` names the adaptation in its section
comment). Helper prose that is a lecture rather than a state (a file path, a
protocol note) belongs in docs-site, not under a control. The page groups the
form into Identity, Connection and Models sections (Name + API type share a
two-column row at 720 px and up; Verify key sits beside the key it checks),
and fields only some providers need sit behind an **Advanced** disclosure
whose sections carry a legend and a hairline — Request compatibility,
Thinking — so a collapsed Advanced is one line and an open one reads as
structure. Every state the exchange can be in is written in the unit that owns
it, never in a second surface: a load reports in the Model ids unit ("Found 3
models. Filled 6 limits from the endpoint.") and a failure takes the same line
in red with the fix ("The endpoint refused the key (401): … Check the API key
and try again."). The actions sit in normal flow at the form's end — no
sticky footer, nothing to overlap.

**Verify (P4).** For a built-in, Verify still asks pi (`pi auth check`), which
costs nothing and is the code path that runs the agent. A custom provider
cannot be answered that way — pi only reports that a credential is present, so
a wrong key on a gateway reads green — and `POST
/api/providers/{id}/verify` therefore sends **one minimal real request**
(`internal/modellist.Probe`): one word in, the smallest output ceiling each API
accepts (`max_tokens: 1`, or `max_completion_tokens` after one retry when the
gateway says so, 16 for the Responses API, `maxOutputTokens: 1` for Google).
The row's action names the cost before it is spent ("Verify with the provider
(1 request)"); the page verifies what the form holds, before saving. The
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
schema and again by `validateCustomDef`, so the GUI is not the only guard.
The catalog hands the stored `thinkingLevelMap` back inside `definitions`, so
Edit prefills the same chips it would write (`customThinkingLevels`). The
levels row was the control that finally outgrew the create dialog: the form
moved to its own page instead of growing the dialog again. A selected level
may also carry its own provider string (`xhigh` → `high`): the value shows
beside the chip only when it differs from the level name, a blank keeps the
identity, and the server refuses values for levels the form does not manage.

**Per-model details.** Each model row also carries pi's optional `name`
(display name, ≤ 120 chars), `input` (the only modalities pi's `Model`
knows: `text`, `image`) and `cost` (USD per 1M tokens, pi-ai `models.js`
divides the rates by 1e6). Cost is all or nothing — four rates or none, so
a half-filled row cannot silently zero real money — and zeros are fine. A
blank row field deletes the stored key on save, so clearing a hand-set
value removes it instead of hiding it; a row that never had one writes
nothing. The third managed compat bool is `supportsUsageInStreaming`, which
the Responses API needs before pi reads usage off the stream. The API key
stays exactly the cheaperinference flow: a literal credential into
`auth.json`, never an env reference inside the definition.
