# ADR-0175: Custom provider definitions for omp

- **Status**: accepted (owner approved 2026-09-21, in session)
- **Date**: 2026-09-21
- **Boundary**: persistence — PiCode writes `~/.omp/agent/models.yml`, an omp-owned file it never touched; security model — for omp the API key lives **inside** the definition file, amending ADR-0129's "credentials never in the definition"; amends ADR-0169's "custom endpoints stay pi-only".

## Context

The owner asked why pi's Providers pane carries **Custom provider / Add
provider** and omp does not. omp is pi-family (`@oh-my-pi/pi-coding-agent`)
and supports the same user-defined gateways — but in its own file,
`~/.omp/agent/models.yml` (YAML, `providers:` wrapper, schema-validated), not
pi's `models.json`. Its credentials otherwise live in a live SQLite database
(`agent.db`) PiCode deliberately does not open (ADR-0165), and a custom id has
no OAuth flow, no login command and no env-var mapping. Measured against the
installed omp 18.2.8 docs (`models.md`, `provider-compat-reference.md`) and
bundle.

Options: (a) keep the custom surface pi-only (ADR-0169's standing rule);
(b) write omp's `models.yml` by merge, as ADR-0129 does for pi; (c) a
PiCode-side store injected at terminal spawn. (a) leaves the pane lying by
omission — the pane's promise is "what this CLI can talk to", and omp can talk
to gateways the GUI cannot express. (c) is the second-source-of-truth design
ADR-0129 already refused for pi. omp's schema notes sharpen (b): it accepts
only `providers` at the top level, has no `thinkingLevelMap` (zero occurrences
in the 18.2.8 bundle), documents five `thinkingFormat` values and no
`include_usage` compat switch — so the pi form's advanced fields cannot be
offered to omp unchanged without writing keys omp's schema refuses.

## Decision

PiCode creates and edits omp's custom provider definitions by merging into
`~/.omp/agent/models.yml` — node-level YAML (yaml.v3), upsert by provider id,
untouched providers, unknown fields inside a touched one (`headers`,
`discovery`, `samplingParams`, …) and the file's comments survive; a file omp
would reject (unknown root keys) is never written into. The definition and the
credential are one row: `apiKey` is written inside the provider entry, which
is omp's own documented channel and the only one a custom id has. The key is
never serialized back — the roster carries `keyed`, and the saved key travels
only server-side, to the gateway the user configured (verify, model listing),
never to the browser. The form's omp variant offers the omp subset — two
compat flags, five thinking formats, no chat-template objects, no thinking
levels (the server refuses the rest in depth) — and omits every pi-only
field. The roster gains omp's `custom` door and serves each definition as a
row (`definition` + `keyed`); the pane renders a definition line with
Edit / Verify / Remove, and Remove deletes definition + key with the blast
radius named. Scope: pi and omp only; every other guest keeps the ADR-0169
rule.

## Consequences

PiCode now writes a second tool's file. The merge is conservative for the
same reason ADR-0129's is (tests pin field and comment survival), and
hand-edits keep working: upsert never deletes what it does not own. Because
the key sits in the definition file, Edit's blank-key rule ("keep the stored
credential") and Remove ("definition and key together") are the only
credential verbs — there is no vault row, no Use, no Pause for an omp custom
provider, and the pane says so with a definition line instead of an account
row. Schema drift is the standing risk: if omp adds a `thinkingLevelMap` or
new compat flags, PiCode's offered set stays narrower until someone re-reads
omp's docs — the offered subset is vendored knowledge pinned by tests and
dated comments, the same trade `LoginMethods` makes for pi. If omp's schema
one day rejects unknown provider keys strictly, a hand-edited field next to
ours becomes the failure, not our write — the root-key guard keeps the
file-loadable invariant we control.

## Alternatives considered

- **Keep it pi-only (status quo of ADR-0169).** Refused by the owner: the
  pane's doors should follow each CLI's real capability, and omp has one.
- **Vault-stored key + spawn-time env injection.** No env var exists for a
  custom omp id; injecting at spawn is the translation layer ADR-0129 refused,
  and it would not cover terminals launched outside PiCode.
- **Write omp's `models.yml` with the full pi form (all formats, levels,
  template objects).** Refused: half the fields would be keys omp's schema
  refuses or ignores — a control that cannot work is a lie the form should
  not tell.
