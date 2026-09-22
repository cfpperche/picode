# 2026-09-21 — keep-old-login: "Keep both" in the sign-in flow

Shipped: when Check now's preview says the write would replace an unnamed
login, the strip offers **Name and add** (askPrompt → import with `as` →
Adopt re-keys the live row) or **Keep both** (`keep: true` — the saved login
is named from what the vault knows, its email or label, else "Previous
login"; the new login lands as its own unnamed row, which then matches the
live file). The strip carries the choice instead of writing silently.

Verified: scratch walkthrough on a fresh port — Sign in with login A in the
file, swap to login B, Check now shows both doors; Keep both → two rows
("Previous login" inactive, "Account 2" active and matched to the live
file). Server test `TestCredentialImportKeepKeepsBothLogins` pins it. The
naming door was exercised in the previous QA round. Blind spot: the
scratch's vault carried leftovers from earlier QA runs, so labels in the
final roster are QA-history, not product defaults.

visual-review: PASS (keep-doors.png + keep-both.png read in-session: both
doors rendered, both rows after, active flag on the new one)
Not done: the strip photograph for the guide (same fixture limitation as
guide-second-account).

## Next up

- none in this slice
