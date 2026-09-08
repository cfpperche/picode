# 2026-09-08 — feat/git-graph-wt-pill: remove a clean worktree from its pill

The owner's first real use of the graph's write phase — "delete the
test-graph branch and its worktree, through the graph" — had no door: a
checkout with nothing uncommitted draws no row, and the worktree menu hung
only on dirty rows and detached chips. The branch pill of a branch checked
out in a sibling worktree now carries that worktree's rows (prune, remove,
remove --force); the reader's own branch and an unchecked-out one do not.
One shared-module change, one test (54 pass), CHANGELOG line.

Found by use, not by review — the adversarial pass missed it because it
read the menus per target and never asked "how do I finish this task".
