# 2026-09-14 — browser-fullurl

Slice 3 increment 3.2a: the Show full URL pref.

## Done

- `GET/PUT /api/browser/prefs` (settings KV, key browser.showFullUrl,
  default true); PUT announces setting.updated.
- BrowserPage: Address bar section with the On/Off switch. WebTab reads
  the pref and trims the meta URL to its origin when off (re-fetches on
  the feed event; effect re-subscribes on the pref flip).
- Scratch: toggle round-trips to the store, switch reflects it, overlay
  audit ok. Screenshots in var/screenshots/show-full-url-off.png.

## Next up

- 3.2b: the address-bar history dropdown (typed URLs first; the store
  and endpoints are live).
