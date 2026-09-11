# ADR-0119: Package configuration by descriptor, for every extension

- **Status**: proposed (direction approved by the owner, 2026-09-11; C0/C1/C2
  shipped on this branch)
- **Date**: 2026-09-11
- **Boundary**: persistence + process — the server now writes config files
  for any descriptor-declared package (not only pi-roles), at paths the
  descriptor declares, with the extension still owning the file's meaning
- **Extends**: [ADR-0099](0099-package-configuration-gui.md) (supersedes the
  deferred-manifest clause of its "Alternatives considered")
- **Plan**: [../plans/package-config-manifest.md](../plans/package-config-manifest.md)

## Context

ADR-0099 built the roles editor and set the contract: files stay the only
source of truth, unknown keys survive writes, saves are atomic, a file the
parser refuses is never silently overwritten, reset is scoped, saves
announce on the feed. Its adapters were code: `pipkg.ConfigKindOf` matched
`pi-roles` by substring, and the Packages view showed Configure for that
kind alone. The declarative manifest was named "the right end state for
third-party packages" and deferred until a second adapter proved the shape.

The second case arrived and it was not pi-compact-shaped: `pi-web-search`
reads `~/.pi/agent/web-search.json` (`{provider, model}`, provider must back
native search). A roles-style adapter (two layers, overlay inheritance,
scoped resets) would have been mostly dead code for it — and meanwhile the
file was never created on any machine, because no surface said it existed;
the tool names it only inside an error message.

## Decision

A package is configurable in the GUI when a **config descriptor** resolves
for it, in this order: a `picode.config` object in the installed package's
`package.json` (upstream-friendly, Raycast-preferences shape), then
PiCode's catalog (`internal/pipkg/configdescriptor.go`, first entry
`pi-web-search` → `web-search.json`), then nothing — no descriptor, no
Configure button, never a generic JSON editor (ADR-0099 §5 unchanged).
The server generalizes `GET/PUT/DELETE /api/packages/config` to
descriptor-driven configs: it reads and writes the declared file (v1:
agent-global scope), validates values against the descriptor's typed fields
(required, enum from the tool's own supported-provider list), merges them
onto the raw document so unknown keys survive, writes atomically, answers
409 on an unparseable file unless the request is an explicit force, and
publishes the same `packages.config` feed event. The desktop renders a
generic form from the descriptor (`PackageConfigGeneric.jsx`) and validates
with the same grammar in Zod (`descriptorValuesSchema`). pi-roles keeps its
bespoke two-layer editor untouched; the descriptor system is additive.

## Consequences

- Easier: any extension with a simple JSON config becomes GUI-configurable
  with data (a catalog entry, or upstream, a manifest) — pi-web-search ships
  configured in this branch, and the next package is a JSON blob, not a
  server + client change.
- Harder: the descriptor grammar (`ConfigField` types) is now a contract —
  growing it (per-agent overlays, workspace scopes, cross-field validation
  like "model belongs to provider") is a coordinated server + client + schema
  change. Two grammars now exist by design: roles' rich code adapter and the
  descriptor engine; drift between them is caught by their separate tests.
- If wrong: a descriptor that misdescribes a package writes a file the
  extension ignores — recoverable by fixing the descriptor; the extension
  still validates its own file, and unknown-key preservation means a wrong
  write cannot destroy data it does not understand.

## Alternatives considered

- **Second hand-written adapter (pi-web-search roles-style)** — rejected:
  most of the adapter (layers, merge, resets) would be dead code for a
  single-file config; each new package would stay a server + client change.
- **Generic JSON editor per package** — rejected again (ADR-0099 §5): no
  validation, no layer semantics; a text box is how files get mangled.
- **Wait for pi-compact to prove the shape first** — rejected this time:
  the discoverability failure (config files no one knows exist) is live in
  production for packages already installed, and web-search's flat shape is
  a *better* seed for the generic grammar than pi-compact's would be.
