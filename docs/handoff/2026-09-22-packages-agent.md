# 2026-09-22 — feat/packages-agent: the agent scope reaches Omp (ADR-0176, slice 4)
Shipped: the agent scope stops being Pi's alone. It is PiCode's own list on the agent row, so what a
CLI needs to declare it is a launch that can pass the entries on. `clipkgs` gives omp the row
("PiCode passes this agent's own entries to Omp at its next start"); `pkgs` answers that layer from
the store with no vendor call — under both names the pane uses (`scope=agent` and `vendor=agent`) —
offers the radio only when the read named an agent, answers an empty badge read there rather than
asking a vendor for a catalog the layer lacks, and carries the agent's name so the row is badged with
it. `Caps.IsolatedSwitch` is offered exactly where a launch honours the agent row's flag (pi, omp);
the Omp launch appends `-e <entry>` per stored entry plus `--no-extensions --no-skills` when the agent
is isolated — the two flags that CLI has, never one it would refuse — and refuses the launch when a
`--trusted-extension` flag would meet them (the integration guard covers the activity extension; this
is the same refusal, injected after that guard runs). Pi's vector is unchanged.
`web`: three rules moved into the contract where they are testable — `directMutation` (an agent-layer
write is PiCode's own call on any CLI, while a vendor's own layers stay on the job lane), `paneTabs`
(no vendor catalog to switch to on that layer) and `rowToggle`/`rowInspect` (the vendor's own verbs do
not know an entry it only learns about at launch).
Verified: `make close` green (fmt,vet,hooks,go,test-js,build; 11 path(s)); 27 JS contract tests;
engine, launch and route tests, including `TestOmpLaunchCarriesTheAgentsScope` over the generated
script (no scope / one entry / isolated) and the conflict case. Live on a scratch instance of this
worktree: the pane's own route wrote the entry, `GET /api/packages/report?cli=omp&…&vendor=agent`
answered `["Global","This workspace","This agent"]`, the agent's name and the entry row, and the same
read with no agent offered no agent radio. Not run: `PICODE_PKGS_LIVE=1`.
visual-review: PASS (pane4-omp-agent.png — the agent radio selected, the layer note, the entry card
with Remove alone and no marketplace tab; `__picodeOverlayAudit()` ok; card 5/5)
Merge: main had moved; merged, re-closed, then fast-forward from the root.

## Debts

- Omp's roster extension rows still draw Enable/Disable and the CLI's disable verb does not know
  extensions — unchanged by this slice, owned by `docs/handoff/open/packages.md`.
