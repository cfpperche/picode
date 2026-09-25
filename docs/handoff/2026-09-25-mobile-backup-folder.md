# 2026-09-25 — feat/mobile-backup-folder: phone Backup Folder field gets a full row
Shipped: on the phone, Preferences › Backup's Folder field shared one row with
Browse and Reveal, leaving the input ~150px at 390 ("External drive or other
fol…"). web/mobile/src/styles/mobile-settings.css now wraps .folder-field-wrap
(FolderField is display: contents there): input takes the full row (362px at
390; the placeholder needs 222), Browse and Reveal sit below as equal halves
at 36px. Desktop unchanged.
Verified: `make ci-scoped` PASS; scratch instance at 390×844, geometry probe
plus a screenshot read in a subagent. Not checked on a physical phone.
visual-review: PASS
Not done / debts: none.
Merge: fast-forward ready.
