# 2026-09-24 — feat/ci-ceilings: the two numbers the run produced

The corrected split came back green on ubuntu, macOS and Windows: Go 7 min on ubuntu and 6 on macOS, Go heavy 36 and 32, wall 37 against 41.5 before.
That measured the two ceilings I had set an hour earlier and shown both wrong in opposite directions: 35m for the rest was sized on an unmeasured macOS half (it is 6 minutes — three times that, with room for a cold build cache, is 25m), and 40m for the pair was 90% of a suite that has measured 26 and 36 minutes on the same runner — I had built the cliff I set out to remove.
Both are derived from the run now, and the comments say so. Nothing else changed; the guard test asserts the shape, not the numbers, so it still passes 29/29.
