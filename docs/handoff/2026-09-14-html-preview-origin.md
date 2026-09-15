# 2026-09-14 — feat/html-preview-origin: the preview's own origin (ADR-0137)

v2 of the HTML preview, shipped as the owner approved it (study D1–D5 in
`docs/plans/html-preview-v2.md`).
A ticket also carries a DNS label (26-char base32) and `previewHostHandler`
routes `<label>.localhost:<port>` **before** the auth gate: that origin serves
the preview namespace only — `/api`, the app shell and unknown labels answer
404 — while the sandboxed path form stays as the fallback. Origin responses
drop the CSP sandbox and the iframe drops its `sandbox` attribute, name the
minting origin in `frame-ancestors` and narrow CORS to it; Reload keeps the
ticket (storage survives), Retry mints again, Save `DELETE`s the overlay so
the document returns to disk on the same origin.
Verified: `make ci-scoped` PASS; Go tests (host-form routes, header split, CORS
on every answer, overlay PUT/DELETE, off-loopback mints, label table; store
label/ByLabel/ClearOverlay); frontend 788+357+338. Scratch `origin` (:8473):
origin form renders (own origin, service worker registered, `data.json` ok,
same-origin `/api/health` 404, credentialed PiCode API blocked), storage
survives its own reload and a Save, live reload updates the frame, the sandbox
fallback (gateway-style proxy, non-loopback Host) shows the note + `unavailable`
rows, mobile renders; overlay audit ok.
Two in-branch fixes: the overlay PUT/DELETE answers needed the CORS allowance
(invisible to a Go test; a browser saw "Failed to fetch" and re-minted, wiping
storage), and Raw → Preview used to mint a new ticket. Measured and documented:
in-frame cookies are dropped as third-party (Chrome), and the origin's storage
is partitioned from a top-level visit.
visual-review: PASS (5/5) · Merge: fast-forward ready.
