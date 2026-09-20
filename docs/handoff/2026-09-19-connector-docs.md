# 2026-09-19 — connector-docs: Docs link on marketplace cards
Shipped: marketplace cards show **Docs** when `docsUrl` is http(s) —
registry `websiteUrl` else repository, seed cookbook pages for Gmail /
DeepWiki / picode-*. Cards without a URL hide the link (not a disabled pill).
Verified: `make close` PASS; catalog + gallery + `connectorDocsUrl` tests;
scratch :8471 Marketplace (seed Docs), empty search, adapter-missing
blocked; overlayAudit ok; Docs click opened mcp-deepwiki. Blind spot:
registry Docs need a catalog refresh (old cache is marked stale).
visual-review: PASS (connector-docs-marketplace.png + empty + blocked)
Merge: fast-forward ready.
