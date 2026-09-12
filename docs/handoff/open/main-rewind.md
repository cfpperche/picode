# Main rewind guard

Git guard from the 2026-09-12 stale-ff incident (a concurrent session's
fast-forward onto a divergent tip erased the deployed bootstrap fix from
main). The reference-transaction hook refuses any move of
refs/heads/main that is not a pure fast-forward; rollback requires
PICODE_ALLOW_MAIN_REWIND=1. AGENTS.md §5 documents it; hooks selftest
covers reset --hard, the override, forward ff, and the divergent
incident replay. Note: git update-ref does not invoke the
reference-transaction hook (git 2.53, verified with a spy hook).
