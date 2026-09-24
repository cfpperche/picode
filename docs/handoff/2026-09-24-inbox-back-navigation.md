# 2026-09-24 — feat/inbox-back-navigation: desktop Inbox Back button

Changed: the desktop Inbox header now uses Back beside its title, matching Agent CLIs, and returns to `#/`; the former Close control is removed. Architecture notes, changelog fragment, and affected docs screenshots were updated.
Verified: `make ci-scoped` passed; `node scripts/docs-check.mjs --strict` passed. On a scratch instance, clicking Back reached `#/`.
visual-review: PASS — read empty and fetch-error Inbox screenshots plus the Agent CLIs Back reference. Back alignment, one-line error, and Try again were readable without clipping; scratch overlay audit was ok. Recaptured Canvas and mobile Inbox docs images were read and showed no visual defect.
Limit: scratch Chromium review does not establish native Windows shell behavior.
Deploy: none; deployment remains the owner's call.
