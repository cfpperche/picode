# 2026-09-24 — pane-scope: a scope carried between setup tabs in each tab's words

Owner report: after opening Skills, Packages and Connectors showed "This … link is
invalid". Cause: `cliPaneSetupContext` keeps the route's `scope`, and `cliSetupHref`
passed it raw; Skills says machine/workspace/agent while Packages and Connectors accept
only user/project/agent, so `scope=workspace` made their links invalid (latent since the
Skills tab carried a scope; the scope chips made it common). `paneScope(pane, scope)` in
`shared/domain/cliLaunch.js` translates through `scopeKind` (packages/connectors
user/project/agent, skills machine/workspace/agent); both apps' `cliSetupHref` use it and
Skills now receives the carried scope too. Memory keeps its old rule: setupCtx defaults
an empty scope to "user", and translating that would pin Memory to Global.
Verified: table test in cliLaunch.test.js; scratch: Skills workspace/agent → Packages
and Connectors land on scope=project/agent with no invalid notice; Packages project →
Skills scope=workspace.
