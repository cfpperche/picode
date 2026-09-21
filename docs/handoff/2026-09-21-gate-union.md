# 2026-09-21 — feat/gate-union: the debt filed an hour earlier was cheaper to fix than to record

One commit (dd6ca5f8). `feat/audit-selfreview` closed two holes in the ADR-0048 mutation gate and filed a third as a known limitation: the gate read SQL as text, so a statement assembled from a package-level constant, or a table name `[a-z_]+` could not spell, stayed invisible. Measuring it before writing the topic file showed the fix was three lines and the note describing it was five.

A write is now **either SQL the test can read or a call that runs one**. Neither half works alone — the text signal misses the constant; the call signal misses `AppendEventTx`, whose `Exec` sits on a `txRunner` it was handed. The union has no gap left: every write in `internal/store` reaches SQLite through one `Exec` or the other, and the package has no `Exec` that is not SQL (checked). It adds exactly one method, `VacuumInto`, which copies the database out to a backup snapshot and changes no row; it is listed with that reason, bringing the deliberate exceptions to seventeen.

Probes for both evasions — a write through `const probeStmt`, and `UPDATE pin_files2 SET …` — fail without the change and pass with it.

The lesson worth keeping: "recorded as a debt" read like diligence and was really an unmeasured guess about cost. The measurement took one command.

Verified: `make close` green, scoped (fmt, vet, hooks, go). visual-review: n/a — no UI diff. Nothing deployed.
