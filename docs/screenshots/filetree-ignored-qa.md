# File Tree ignore decoration acceptance

Git owns ignore matching; desktop applies the existing secondary text token
and italics, preserving the selected row's primary text and keyboard focus.
This follows the existing Cursor-inspired explorer density without new chrome.

| Conditions | Result | Evidence |
|---|---|---|
| Ignore rule matches a file or directory | Muted italic name, accessible explanation | Go `TestBrowseIgnored`; dark/light captures |
| Expanded child inherits ignore rule | Same decoration | Go test; lazy-level JS test; captures |
| Negated rule or already tracked file | Normal name | Go test; browser assertions |
| Root is a repository subdirectory | Parent rules apply | Go test |
| Filename contains spaces or newline | Exact matching | Go test |
| Non-Git directory or Git unavailable | Ordinary browsing | Go test |
| Ignored file selected | Legible selection, inline content | selected capture |
| Empty folder or blocked read | Placeholder/action or error/reload | light and blocked captures |
| Refreshed entry loses ignore status | Decoration removed | lazy-level JS test |

Screenshots: `filetree-ignored-dark.png`, `filetree-ignored-light.png`,
`filetree-ignored-selected.png`, `filetree-ignored-blocked.png`.
All four screenshots were read; overlay audits returned `ok: true`.
