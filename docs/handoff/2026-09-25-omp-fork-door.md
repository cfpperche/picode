# 2026-09-25 — omp-fork-door: Omp's fork task through the prompt door

Paid the last agent-fork task debt. `forkAgent` sends the task through the door for a
CLI in `doorReaderCLI` or `unattendedReaderCLI` (Omp, ADR-0217, another session's
reader): `deliverForkTask` is an unattended sender. Omp's launch carries no task now.
Live (`TestLiveOmpForkTaskThroughTheDoor`, gated, own tmux socket, real omp, real HOME):
delivered at 100 columns, but refused at 44 — the 70-column gate in `peerInputMatches`.
Omp's frame (`╰─` under the ` > ` status bar, nothing below) holds at 44 (real capture),
so Omp is checked before that gate from 30 columns, as Pi is. Delivery reads
`unconfirmed/diverged` for a two-line task (the composer spans rows) but the screen
shows both lines submitted and Omp working. forkPrompt keeps a unit test as the fallback
for a future flag-forking CLI without a reader; the 8192 refusal subtest is gone (no CLI
reaches it now). First live runs hit Omp's setup wizard: the package isolates HOME.
