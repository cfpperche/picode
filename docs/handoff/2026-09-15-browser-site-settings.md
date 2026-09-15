# 2026-09-15 — browser-site-settings

Third slice of Browser permissions: the visible half, on top of the store
(feat/browser-permission-store) and the shell's handler
(feat/browser-permissions-shell).

## Done

- The **Browser permissions** section with Site settings → Manage, and the
  dialog: six kinds, each with a policy select (Allow / Block / Platform
  default) written as the `*` standing, plus Recent decisions (site, kind,
  decision, Reset). The page hands the standings back to the shell on every
  load and feed event, so the policy survives a restart.
- Visual: dialog read (kinds, selects, list, honest lede); the policy write
  verified against the live API (`*=allow` beside the per-site `deny`);
  overlay audit ok, no clipping, the dialog scrolls internally.

## Not built yet

- The **Ask prompt**: `ask` as a third policy is where the deferral
  (`GetDeferral`) belongs — hold the request, ask in the tab, answer. Until
  then the select offers Allow/Block/default only, and the row copy says so.
- The **JavaScript** toggle (`IsScriptEnabled`) closes the section.
