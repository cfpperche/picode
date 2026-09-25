# Markdown preview — GitHub parity, then an editor

The owner asked (2026-09-25) for a markdown viewer/editor on par with the
reference tools. Researched options, cheapest first: A) GitHub-grade preview,
B) VS Code-style source | preview split with scroll sync, C) Obsidian-style
live preview inside CodeMirror 6, D) WYSIWYG (Milkdown/MDXEditor/Tiptap) —
D refused for repository files because it re-serializes the file and rewrites
tables, lists and emphasis on save. A shipped on `feat/md-preview`, B (Split with scroll sync) on `feat/md-split`, C (Live) on `feat/md-live`
(`docs/architecture/file-preview.md`, "Markdown as a document").

## Next

- Live on the phone: a way to follow a link by touch (long-press?) — Ctrl/⌘+click has no touch equivalent; Preview follows links today.

## Debts

- [ ] Source views (Raw, Edit, Split, Live on the cursor line): table pipes take `tok-meta`'s italic and read like `/`; fenced code is not highlighted inside the editor (no `codeLanguages` given to `markdown()`).
- [ ] Live → Split: the first source scroll can land a few hundred px off while CodeMirror re-measures Live's taller lines; the sync itself stays on the right section.

- [ ] Live: a top-level bullet sits ~3px from its text (the source's own space in the reading face); a bullet margin would read better but brings back a ~5px shift when the line reveals `- `.

- [ ] Relative file links are inert text in canvas file panels and chat file cards (no `onOpenPath` at those mounts); tree, file tab and mobile Files open them.
- [ ] Opening `other.md#section` opens the file at its top; the fragment is not scrolled to after load.
- [ ] Raw HTML in markdown now renders after GitHub's sanitizer allow-list (no script, style, event handlers or unprefixed ids). No ADR was written: the owner may want one, since it widens what file content reaches the app DOM.
- [ ] `.workspace-view.file-on` is `overflow: hidden` with content taller than its box, so any `scrollIntoView` inside a file tab can scroll the clipped ancestor and push the file toolbar off-screen; MarkdownDoc avoids scrollIntoView for this reason.
