# 2026-09-25 — fork-task-latency: a Pi fork no longer waits 30 s to start

Owner report: a Pi fork opened with the history but "never got its task". The copy
shows the task delivered 47 s after the fork, 16 s before the owner removed the agent.
Cause, measured on a scratch instance with a 1.1 MB conversation: the headless
`pi --fork` run (pi_fork.go) exits on stdin EOF while an async startup credential
refresh holds pi's `auth.json.lock` (proper-lockfile, `stale: 3e4`); the lock outlives
the run, and the new agent's pi waits out the 30 s staleness on "Starting Pi..." (no
CPU, no children, main thread in epoll). Ruled out by measurement: session size, the
injected extensions, the agent env, PiCode's PATH, tmux server and HOME, the prompt
door. Fix: the headless run sets `PI_OFFLINE=1` (lock left on 5/5 runs before, 0/8 with
it, scratch and real HOME). After: Pi ready at 1.3 s, task in the pane at 4.4 s.
Test: TestPiForkIntoDir asserts PI_OFFLINE (a mutation removing it fails).
