# Licensing

PiCode is **open source** under the [Apache License 2.0](LICENSE)
([ADR-0218](docs/decisions/0218-apache-license.md)). You may use it at
home and at work, change it and redistribute it under those terms.

| Path | License |
|---|---|
| Everything not listed below | [Apache-2.0](LICENSE) |
| `packages/*/` (installable CLI packages: `pi-roles`, `pi-inbox`, `pi-checklist`, `pi-compact`, `pi-diff`, `pi-browser`, `pi-browser-capture`, `pi-computer`, `pi-delivery`, `pi-sysadmin`, `pi-connector-deepwiki`, `pi-connector-gmail`, `omp-checklist`) | MIT — each package's own `LICENSE` ([ADR-0028](docs/decisions/0028-model-roles.md)) |
| `ee/` (does not exist yet) | Commercial license, stated in `ee/LICENSE` when the first file lands |

## The `ee/` rule

Paid features live only under a top-level `ee/` directory. It holds
features whose buyer is an organization controlling a team: the team's
fleet view, team inbox routing, central configuration of skills,
instructions and models, company credentials with budgets, audit export,
and SSO/SCIM. Anything one person needs stays outside `ee/` and stays
Apache-2.0, including the multi-user gateway. Code published outside `ee/`
is never moved into it.

## Earlier versions

Each version keeps the license it shipped with:

| Versions | License |
|---|---|
| Before 2026-08-25 | MIT |
| 2026-08-25 until ADR-0218 | PolyForm Noncommercial 1.0.0 plus a commercial license |
| From ADR-0218 on | Apache-2.0 (packages MIT) |

## Contributions

Contributions are licensed under Apache-2.0 (inbound equals outbound,
Apache-2.0 section 5); contributions under `packages/` are MIT. No CLA is
required. Do not contribute code you cannot offer on those terms.

## Trademark

The license grants no right to the PiCode name or logo (Apache-2.0
section 6). A fork is welcome under a different name.

This is not legal advice.
