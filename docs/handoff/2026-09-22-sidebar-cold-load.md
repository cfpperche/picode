# 2026-09-22 — sidebar-cold-load: no "nothing yet" before the fleet is read

Shipped: the sidebar's Workspaces, Agents and Terminals lists and the main pane's empty card no longer say
"No … yet" while the boot is still reading the fleet; they show skeleton bars until `bootstrapped` (set even when
the boot fails). `web/browser/src/lib/sidebarBody.js` holds the three-state rule (skeleton / empty / list), tested.
Skeleton bars in those two places use `--border-strong`; `--bg-hover` was nearly the sidebar's own ground.
The mobile Work screen already had a loading state; nothing changed there.
Verified: `make ci-scoped` PASS; visual-review PASS on scratch over two rounds with a document-start DOM timeline
(skeleton → list, never the empty line; the true empty state still appears), light and dark captures read.
Found and left alone: the gap itself is the boot awaiting `/api/catalog` (2.2–4.4 s measured) before
`/api/workspaces` (`App.jsx` boot); the account chip reads "local" until `/api/system` lands; the skeleton card is
smaller than the empty card it can turn into.

## Next up

- Boot: read the fleet in parallel with `/api/catalog` instead of after it — the skeleton would then last ~30 ms
