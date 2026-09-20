# 2026-09-20 — feat/omp-needs-you: omp terminals report Needs you
Shipped: omp extension template listens to omp's own event set
(`tool_approval_requested`/`resolved`, ask card `tool_execution_start`/`tool_result`,
`agent_end` guarded by `willContinue`); same Inbox item other CLIs file
(`internal/server/term_intercept.go`, `cli_wrapper_test.go`, changelog fragment).
Verified: events checked against omp 18.2.6 bundle; probe PROBE-OK;
TestWrapperInstallShape + server battery green; `make close` ci-scoped PASS.
visual-review: n/a
Not done / debts: none known.
Merge: fast-forward ready.

## Next up

- Owner to deploy when wanted (`make deploy` from root).
