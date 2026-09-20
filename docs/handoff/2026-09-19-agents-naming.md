# 2026-09-19 — agents-naming: "guest" is gone
Owner: PiCode supports multiple agents — stop calling the non-Pi ones
"guests". Swept code, copy and living docs to **CLI agent**:
identifiers (`attachAgentTerminal`, `CliAgentFace`, `cliScope`,
`cliEntry`/`cliAcc` in climetrics, `cli.go`), comments, one user-visible
error ("connectors reveal needs a CLI agent"), architecture prose and the
cli-as-agents plan. Applied migrations, ADRs, session handoffs and the
published changelog keep their original words — history is not rewritten.
Verified: go build/vet, store+apps+connectors+climetrics+mcp+rpc+
clisession+session tests, server package (one known shard flake passed on
rerun), node tests, `make web`, ci-scoped PASS (162 paths). No UI pixel
changed beyond copy — visual-review n/a.

## Next up

- Fatia F: automations and Inspector reach CLI agents through the prompt
  door (ADR-0089/0107).
