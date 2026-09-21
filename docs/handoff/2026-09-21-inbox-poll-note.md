# 2026-09-21 — inbox-poll-note: the recorded-answer note stops promising a pickup nobody made

Earlier today (`picode inbox notify|ask`, feat/inbox-cli) a question whose source has no reply
channel stopped being refused: the answer is **recorded on the item** (`internal/apps/inbox.go`,
`internal/server/inbox.go`), so the human's Reply always closes it. That fix shipped a note/toast
promising "the CLI that asked will pick it up here" — true only for `picode inbox ask --wait`. A
plain `ask` and a pi launched outside the launcher (unmanaged identity, ADR-0060) read the durable
queue instead, so the promise was a lie the human acts on.

Changed: one honest note for every case. `InboxAnswerRecordedNote` = "Answer recorded on the item —
`picode inbox ask --wait` reads it here. An asker that is not polling (a plain ask, or a pi launched
outside the launcher) must be told another way." Both answer surfaces (app action and respond route)
append it; `InboxAnswerRecordedToast` says the same in human words ("a waiting asker reads it here;
a non-polling asker must be told another way"). Why no `poll` flag: per-case wording was dropped —
the store has no payload column (ADR-0037 describes a payload the implementation never grew), so a
flag would cost a schema migration for copy alone; the single note covers both askers.

Debt: the stale line in `docs/handoff/open/communication.md` ("`system`-sourced items … honest reply
refusal") is paid 2026-09-21 — the answer is recorded and the note/toast name who must be told
another way (the old refusal kept in the copy), landing beside the claim (ADR-0149). The data-plane
caveat (daemon death between park and JSONL row) is untouched.
Verified: `go test ./internal/apps/ ./internal/server/ -run Inbox -count=1` green — each new test
pins **both clauses** in its string (server: note's `--wait` + `not polling`; app: toast's `waiting
asker` + `non-polling`), so wording that drops either path fails. `make close` green. Fragment
`docs/changelog.d/inbox-poll-note.md` (Fixed). Integration: landed on main by the coordinator session.
