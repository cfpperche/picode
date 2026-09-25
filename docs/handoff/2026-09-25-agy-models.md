# 2026-09-25 — agy-models: Antigravity's Model picker lists its own models

Shipped (98c83ce68): an Antigravity climodels reader (`internal/climodels/agy.go`)
that runs `agy models` (id<TAB>display name) with `HOME` set to a throwaway
temp dir holding only a copy of `~/.gemini/antigravity-cli/antigravity-oauth-token`;
the temp dir is removed after. The selector is the display name (the `model`
setting holds e.g. "Gemini 3.8 Flash (Medium)"), so the Choose… picker
(browser + mobile `CliNativeSettings.jsx`) now omits the id hint when it equals
the row's label. Owner asked to try it on 2026-09-25. cli-settings.md updated.

- Measured (agy 1.2.11): signed out it refuses ("Please sign in"); with the
  copied token 3.8 s, 14 models. The copy was rewritten (access_token, expiry,
  id_token changed, refresh_token unchanged); the owner's file kept the same
  size, mtime and sha256, checked directly and again through the scratch API.
- Scratch QA: the list showed, picking wrote `"model": "Gemini 3.1 Pro (High)"`;
  signed out shows "Antigravity is not signed in — sign in to Antigravity
  first". Visual review PASS, overlay audit ok.
- Hermes is now the only CLI without a model reader (no public listing).

## Debts

- The picker's error state (Grok "open Grok once", Antigravity "sign in
  first") is one line with no action button; the blocked-state rule wants one
  action (e.g. Sign in).
