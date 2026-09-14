# 2026-09-13 — main-red-peers: `make ci` is red on main, from a pre-existing failure

Shipped: nothing but the record. `make ci` on `main` failed after the
snippets-v2d merge; the failure is **not** from that merge — it reproduces at
`f3df855e`, the commit that was `main`'s tip before it, and in isolation:

    --- FAIL: TestPeerStopStubbornChildStaysPending (0.09s)
        peer_stop_linux_test.go:73: stubborn child reported stopped: <nil>

`TestPaneRootSurvivesSIGHUP` failed in the same run and passes alone (load
flake). The peer test asserts `stopPeerPane` reports "still closing" for a
stubborn child and gets a nil error instead, so either the behaviour changed
under it or the expectation is stale; the test landed today at 14:29
(`d9bc2dd9`, "server: exercise the stubborn-child stop over a real tmux pane").
`feat/tmux-isolation` (dirty on `internal/tmux/tmux.go`) and `feat/tmux-app`
are working in exactly this code, so the line was recorded in
`docs/handoff/open/terminal.md` for whoever lands first rather than fixed from
under them.
Verified: this branch is docs-only; `make ci-scoped` PASS; the failing test
reproduced at `f3df855e` in a throwaway worktree.
Merge: fast-forward ready.
