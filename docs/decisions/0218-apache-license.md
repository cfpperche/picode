# ADR-0218: Apache-2.0 license with a commercial ee directory rule

- **Status**: accepted
- **Date**: 2026-09-25
- **Boundary**: process — the terms under which anyone may use, change and
  redistribute PiCode, and where paid code may live in the tree
- **Amends**: [ADR-0028](0028-model-roles.md) (the root is no longer
  PolyForm Noncommercial; the MIT carve-out for installable packages stays)

## Context

On 2026-08-25 PiCode moved from MIT to PolyForm Noncommercial 1.0.0 plus a
signed commercial license (commit `bd2ae1790`, no ADR). Under those terms a
developer who tries PiCode at work breaks the license, and a company has to
sign a contract before anyone on the team can evaluate it.

The business model the owner chose on 2026-09-25 is free for developers,
paid by companies that run PiCode for a team. It depends on developers
adopting PiCode at work first, which the license forbids.

The market was measured the same day:

| Tool | License | What is paid |
|---|---|---|
| Orca (stablyai) | MIT | nothing; the company sells another product |
| T3 Code | MIT | nothing |
| Paseo | Apache-2.0 | hosted team Hub, per seat |
| Superset | Elastic-2.0 | seats, remote access, mobile |
| Conductor | closed | cloud, mobile app, team seats |

Every direct competitor with tens of thousands of GitHub stars is MIT or
Apache-2.0, and none charges for local orchestration. Paid tiers are team
seats, hosted services and enterprise controls (SSO/SCIM, audit, SLA).

Non-OSI licenses (FSL, FCL, BUSL, Elastic-2.0) have concrete costs:

- company legal review, where MIT and Apache-2.0 are usually pre-approved;
- exclusion from Homebrew core (it stopped accepting Terraform releases
  after BUSL) and from distribution repositories;
- fewer outside contributors.

The copyright is held by one author, so relicensing needs nobody else's
consent. A dependency audit on 2026-09-25 found no copyleft code in what
ships: the Go modules linked into `cmd/...` are MIT, BSD and Apache-2.0;
the web production dependencies are permissive, plus OFL-1.1 fonts and
dompurify under MPL-2.0 OR Apache-2.0; the desktop shell's crates are
permissive or MPL-2.0.

## Decision

Everything in the tree is licensed under the Apache License 2.0, except the
installable packages under `packages/`, which keep their own MIT license
(ADR-0028). Paid features will live only under a top-level `ee/`
directory, under a commercial license stated in `ee/LICENSE` when the first
file lands. `ee/` holds features whose buyer is an organization controlling
a team: the team's fleet view, team inbox routing, central configuration of
skills, instructions and models, company credentials with budgets, audit
export, and SSO/SCIM. Anything one person needs stays outside `ee/`,
including the existing multi-user gateway (ADR-0051). No `ee/` directory
exists yet.

## Consequences

- Individuals and companies may use, change and redistribute PiCode without
  asking, which is the adoption path the business model needs.
- Revenue can come only from `ee/`, hosted services and support. Nothing
  sells until the first `ee/` feature ships.
- The grant is irrevocable. Every version published under Apache-2.0 stays
  under Apache-2.0. A later restriction binds only future versions, and a
  fork can continue from the last open one (OpenTofu, OpenSearch, Valkey).
  For the same reason, code published outside `ee/` cannot be moved into it
  and taken back.
- Anyone may fork and rebrand. The name is protected by trademark, not by
  the license (Apache-2.0 section 6 grants no trademark rights).
- Windows signing through SignPath Foundation stays out of reach once `ee/`
  exists: it requires an OSI license and forbids proprietary components
  from the maintainer. The unsigned `install.ps1` path of ADR-0098 stays.
- Contributions are Apache-2.0 (inbound equals outbound, Apache-2.0
  section 5). No CLA is required, because Apache-2.0 already lets the
  licensor ship contributed code next to `ee/`.
- Versions released before this change keep the license they shipped with:
  MIT before 2026-08-25, PolyForm Noncommercial from then until this change.

## Alternatives considered

| Option | Why it lost |
|---|---|
| Keep PolyForm Noncommercial + commercial license | Forbids the at-work trial the business model depends on; the most restrictive license among the direct competitors |
| FCL-1.0-ALv2 (Fair Core License) | Stops competitors from copying the code and protects a license key in the same tree. Given agent-assisted rewrites and a different stack from Orca and Paseo, that protection is worth little, and being non-OSI costs the legal pre-approval, Homebrew core and contributors |
| FSL-1.1-ALv2 or BUSL 1.1 | The same non-OSI costs; BUSL also carries HashiCorp's reputation |
| AGPL-3.0 + commercial dual license | OSI and blocks closed SaaS forks, but many companies ban AGPL internally, which blocks the at-work trial |
| MIT | No explicit patent grant or trademark clause, which companies' legal teams look for |
| Apache-2.0 with no paid code at all (money from hosted services only) | Would keep SignPath eligible, but would make company deployment free, and company deployment is what the model sells |
