# ADR-0181: A settings row may be a role or an ordered list, and its path may come from the vendor

- **Status**: accepted (owner, 2026-09-22, after slice 1 shipped)
- **Date**: 2026-09-22
- **Boundary**: persistence — the settings writer gains two value shapes it never wrote (a scalar at a path the vendor's catalog supplies, and an ordered list of strings), and the set of keys PiCode may write into a CLI's own file stops being a fixed declaration.
- **Amends**: [ADR-0163](0163-cli-native-config-and-memory.md), whose decision says "Fields are scalars only, which is what makes the writer safe". **Extends**: [ADR-0174](0174-guest-keymaps.md) (the list primitives and the `FlatMap.Path` idea it introduced). **Unchanged**: [ADR-0099](0099-package-configuration-gui.md) §5 — no generic JSON box, and a key nobody declares is still never written.

## Context

omp keeps per-task model routing in its own settings file, and it is the
capability the owner singled out (`docs/benchmarks/2026-09-22-omp-helpers-placement.md`).
Three of its keys carry it, all measured against the installed 18.2.8 on
2026-09-22:

- `modelRoles` — a map of role id → model selector. Fifteen roles are built
  into the CLI (`src/config/model-roles.ts`), and a user may add their own;
  the vendor's own union rule is built-ins first, then any role named by
  `cycleOrder`, by an assignment, or by `modelTags` (`getKnownRoleIds`).
- `retry.fallbackChains` — a map of role, `provider/model-id` or
  `provider/*` → an **ordered list** of selectors. `[]` is how the CLI is
  told never to fall back, so an empty list is a value, not an absence.
- `cycleOrder` — an ordered list of role ids. It is what omp's own hub draws
  as `⟳ N` beside a role (`pi-tui/src/overlays/model-hub.ts:2181`, "second
  stop of the ctrl+p cycle").

ADR-0163's engine cannot express any of them. Its fields are a fixed list of
dotted keys whose values are scalars, and that restriction is load-bearing:
one scalar is one line, and a one-line span is what the surgical splice can
replace without touching anything else.

Two facts narrow the options. First, the layer that matters is the one the
vendor's own CLI cannot reach: measured 2026-09-22, `omp config set` writes
the global file wherever it runs and `--scope` is not an option, so a
per-project role is only reachable by writing `<ws>/.omp/config.yml`.
Second, PiCode already writes ordered lists of strings safely — ADR-0174's
key-map engine has done it since 2026-09-21 through `listLiteral` and a
splice-or-reinsert dance that two adversarial reviews shaped.

A third fact bounds how much validation is honest. omp's selector grammar is
wider than it looks: `openrouter/z-ai/glm-4.7@cerebras:high` is one model id
with a routing suffix and a thinking suffix, only the first slash splits the
provider, and a model id may legitimately end in `:max`
(`parseModelString`, `splitUpstreamRouting`). A validator that rejected what
it did not recognise would refuse values the CLI accepts.

## Decision

A CLI's declaration may carry a **role matrix**, and a settings field may be
one of two new kinds beside the four scalars:

- `role` — still one scalar (a model selector), but at a path composed from
  the vendor's catalog plus whatever roles the files name. The rows are
  computed per request, in the vendor's own order, from the decoded layers.
- `list` — an ordered list of strings, written through the very primitives
  ADR-0174 already uses; `SetStrings` and this writer are now one function
  (`spliceList`), so they cannot drift.

A field may carry an explicit `Path`, because a key like
`retry.fallbackChains.openai/gpt-4.1-mini` cannot be split on dots. `Path`
never leaves the server: the pane echoes the opaque key, and the writer
rebuilds the path from the declaration that owns its prefix.

Creating a row is therefore writing a key the report did not carry, and the
declaration is the one place that decides a name is writable. A role id must
match the vendor's own rule (`^[a-zA-Z][A-Za-z0-9_-]*$`, from omp's hub); a
chain key must be addressable as one line by this package's YAML walker
(no colon, no quote, no leading indicator). Anything else is refused **by
name**, and the file is untouched.

PiCode does not judge a selector. It enforces only what its own writer must
guarantee — a value is one line — and leaves the meaning to the CLI. The
pane splits a selector into a model and a thinking suffix for display and
joins back exactly the two pieces it split, so a value PiCode did not
recognise survives a round trip byte for byte.

A value a layer holds in a shape this writer will not rewrite is reported as
**unreadable**, dropped from that layer's values, and drawn as one line plus
"Open the file" — never a control that cannot save.

To fill the picker, `GET /api/cli-models?cli=&workspace=` runs the CLI's own
read-only catalog command in the workspace being edited (`internal/climodels`;
omp is the only declaration). It runs when a picker opens, never on mount.
The workspace matters: measured 2026-09-22, the same command answered 55
models in one folder and 0 in the next, because the project disabled a
provider.

## Consequences

Easier: the roles a user actually routes with are editable per workspace,
which is the layer omp's own CLI cannot write; every role row inherits the
provenance line, the accent bar, "Use inherited" and the 409 that ADR-0163
built, because it *is* one of those rows; and the next CLI with a role map
(Hermes has eleven auxiliary slots, Claude has `fallbackModel`) is a
declaration, not a second pane.

Harder: the writer's safety argument is no longer "one token". It is now
"one line, and a list re-inserted whole when its span is a block" — a
larger claim, held by `TestRoleWrites` and by the key-map tests that already
exercise the same function. The role catalog is vendored knowledge that
rots: omp removed `designer` in 18.1.5 and moved five kind roles into
`modelRoles` in 18.2.7, at roughly 1.4 releases a day, and
`TestOmpRoleCatalogIsTheVendorsOwn` is what turns a rename into a red test
rather than a row writing a key the CLI ignores.

Accepted costs: a `role` row offers a picker fed by a subprocess, so a
machine where the CLI is missing or slow gets the CLI's own words in the
picker's footer and a text-free list — the pane still saves. And PiCode
cannot say *why* a catalog is empty: a provider a project disabled and a
provider with no credential look identical from outside, so the pane says
the list is empty and does not guess.

If we are wrong, the failure is bounded by what has not changed: a key
nobody declared is still never written, a document that does not parse is
still never overwritten, and the file remains the only source of truth.

## Alternatives considered

- **A pane of its own, like Keyboard.** Lost on the file: roles, chains and
  the scalar knobs live in the *same* document, so two panes would hold two
  revisions of one file and race each other's 409. One endpoint, one
  revision.
- **Shell out to `omp config set`.** Lost twice over, both measured
  2026-09-22: it writes the global file wherever it runs, so the workspace
  layer — the whole point — is unreachable; and it prints the *effective*
  value rather than what it wrote (`config set symbolPreset nerd` inside a
  project that sets `ascii` answered `[ok] Set symbolPreset = ascii`), so a
  pane built on it would report success for a change nobody will see.
- **A generic JSON or YAML box for the whole `modelRoles` map.** Lost to
  ADR-0099 §5, unchanged: PiCode does not ship a second editor for a file it
  already has two writers on.
- **Validate the selector against the catalog before saving.** Lost to the
  grammar: omp accepts routing and thinking suffixes, ids with slashes and
  dots, role aliases and `*`. Refusing what PiCode does not recognise would
  make the pane narrower than the CLI.
- **Fetch the model catalog on mount.** Lost to the cost: it is a
  subprocess per pane open, on a pane whose other rows need nothing. Lazy on
  first picker open keeps the pane instant and the probe rare.
