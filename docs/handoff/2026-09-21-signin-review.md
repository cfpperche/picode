# 2026-09-21 — signin-review: Check now must see a different login

Shipped: sign-in and import answers carry a stamp of the access token (not
the token). The strip keeps itself open and says the file still holds the
account already saved when Check now's stamp matches the one from Sign in.
Nameless stores keep one row id either way, so comparing ids was a false
"nothing changed" once a second login actually landed.

Verified: scratch Claude Code pane, desktop and 414px. Check now against an
unchanged file kept the strip and showed the refusal (stamp a6b9678f572e4b6c).
Overlay audit ok, three strip controls one row. Screenshots read by a
subagent. Blind spot: a real second Claude OAuth was not completed; the
stamp path is what was exercised.

visual-review: PASS (review-claude.png, review-strip.png, review-strip-mobile.png, review-still-same.png; card 5/5)
Not done: naming a login still happens after an unnamed row is overwritten,
so the previous unnamed tokens are already gone when the prompt appears.

Follow-up (same day): Sign in now reuses the CLI's live sign-in terminal
(`handleCredentialSignin` checks the tmux session before creating; a dead
record is swept). It no longer stacks orphan sessions per click — thirteen
"Claude Code sign-in" tmux sessions were the reason.

## Next up

- Offer the name before the import writes, so an unnamed second login cannot
  eat the first.
