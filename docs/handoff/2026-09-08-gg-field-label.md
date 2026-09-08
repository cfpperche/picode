# 2026-09-08 — feat/gg-field-label: the worktree form names its field

The removal dialog showed "Name" over the worktree folder; `FIELD_LABELS`
had entries for the create forms only. Both `worktree-remove*` actions now
label the field *Worktree folder name*, and a test asserts every action that
collects a name has a label — so the next name-collecting action cannot ship
with a bare "Name" either. One line of copy, one test (55 pass).
