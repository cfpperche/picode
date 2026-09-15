# 2026-09-15 — browser-autofill-dialogs

Owner review: Autofill and passwords must use dialogs, not the reference's
sub-routes.

## Done

- Shell: `wipe_mask` gains `passwords` → PASSWORD_AUTOSAVE, so the Password
  manager dialog can clear them (the Clear browsing data checklist still
  never sends it).
- Page: the two rows carry **Manage** (reference shape); the switches moved
  inside the dialogs, each with its own destructive action (two-step) and
  an honest lede.
- Visual: both dialogs read (screenshots), overlay audit ok, toggle and
  two-step delete exercised.

## Notes

- Platform limit, verified in the bindings: WebView2 exposes only
  Is/SetPasswordAutosaveEnabled and Is/SetGeneralAutofillEnabled — no
  enumeration, no per-entry edit, no import. The reference's list/Add/
  import screens are the Windows/Edge password manager (Windows Hello,
  passkeys), which a host app cannot host.
- A real list with add/edit means PiCode owning the store and injecting
  fills — a security-model decision (ADR) like the annotations backlog.
  Worth registering there if the owner wants it.

## Next up

- Downloads section, then Browser permissions (camera/mic), then Developer
  mode.
